package rest

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-on/fat"
	mt "github.com/go-on/meta"
	. "github.com/metakeule/pgsql"
)

type tableRegistry struct {
	*sync.RWMutex
	tables map[string]*Table
}

func typestring(østruct interface{}) string {
	// fmt.Printf("typestring for %T\n", østruct)
	ty := reflect.TypeOf(østruct).Elem()
	return "*" + ty.PkgPath() + "." + ty.Name()
}

func (tr *tableRegistry) AddTable(name string, t *Table) {
	tr.Lock()
	defer tr.Unlock()
	tr.tables[name] = t
}

func (tr *tableRegistry) Table(name string) (t *Table) {
	tr.RLock()
	defer tr.RUnlock()
	t = tr.tables[name]
	return
}

var TableRegistry = &tableRegistry{
	RWMutex: &sync.RWMutex{},
	tables:  map[string]*Table{},
}

type fieldRegistry struct {
	*sync.RWMutex
	fields map[string]*Field
}

func (fr *fieldRegistry) AddField(tablename, fieldname string, f *Field) {
	fr.Lock()
	defer fr.Unlock()
	// fmt.Printf("registering %s/%s\n", tablename, fieldname)
	fr.fields[tablename+"."+fieldname] = f
}

func (fr *fieldRegistry) Field(tablename, fieldname string) (f *Field) {
	fr.RLock()
	defer fr.RUnlock()
	// fmt.Printf("looking for %s\n", tablename+"/"+fieldname)
	f = fr.fields[tablename+"."+fieldname]
	/*
		if f == nil {
			panic("not found")
		}
	*/
	return
}

var FieldRegistry = &fieldRegistry{
	RWMutex: &sync.RWMutex{},
	fields:  map[string]*Field{},
}

var varcharReg = regexp.MustCompile(`varchar\(([1-2]?[0-9]?[0-9])\)`)

/*
pgsql looks if type tag matches
  []int32, []int16, []int8, []int, []float, []string, []date, []timetz, []time
or
  xml, json,
or
  uuid, varchar(x), char(x), text, int32, int16, int8, int, float, string, date, timetz, time

*/

var matcher = []string{
	"[string]string", "[string]int", "[string]time", "[string]float", "[string]bool",
	"[]string", "[]int", "[]time", "[]float", "[]bool",
	"xml", "json",
	"uuid", "text", "float", "date", "int", "bool", "timestamptz", "timestamp",
	"varchar",
}

func findType(tag string) (typ string) {
	for _, t := range matcher {
		if strings.Contains(tag, t) {
			return t
		}
	}
	return
}

func splitSpace(s string) []string {
	r := strings.Split(s, " ")
	n := []string{}
	for i := 0; i < len(r); i++ {
		trimmed := strings.TrimSpace(r[i])
		if trimmed != "" {
			n = append(n, trimmed)
		}
	}
	return n
}

func RegisterTable(name string, ptrToFatStru interface{}) error {
	val := reflect.ValueOf(ptrToFatStru)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("%T is no pointer to a struct", ptrToFatStru)
	}

	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%T is no pointer to a struct", ptrToFatStru)
	}

	valType := typestring(ptrToFatStru)

	stru, err := mt.StructByValue(val)
	if err != nil {
		return err
	}

	table := NewTable(name)

	fn := func(fld *mt.Field) {
		dbFlag := splitSpace(fld.Type.Tag.Get("db"))

		if len(dbFlag) < 1 {
			return
		}

		//fname := fld.Type.Tag.Get("db")
		fname := dbFlag[0] // fld.Type.Tag.Get("db")

		if fname == "-" {
			return
		}

		//if fname != "" && fname != "-" {
		ff := fld.Value.Interface().(*fat.Field)
		var typ Type
		ftype := findType(fld.Type.Tag.Get("type"))
		if ftype != "" {
			switch ftype {
			case "[string]string", "[string]int", "[string]time", "[string]float", "[string]bool":
				typ = JsonType
			case "int":
				typ = IntType
			case "text":
				typ = TextType
			case "bool":
				typ = BoolType
			case "date":
				typ = DateType
				/*	case "time":
					typ = TimeType */
			case "xml":
				typ = XmlType
			case "float":
				typ = FloatType
			case "[]float":
				typ = FloatsType
			case "timestamptz":
				typ = TimeStampTZType
			case "timestamp":
				typ = TimeStampType
			case "json":
				typ = JsonType
			case "[]int":
				typ = IntsType
			case "[]string":
				typ = StringsType
			case "[]bool":
				typ = BoolsType
			case "html":
				typ = HtmlType
			case "[]time":
				typ = TimeStampsTZType
			case "uuid":
				typ = UuidType
				/*
					case "ltree":
						typ = LtreeType
					case "trigger":
						typ = TriggerType
				*/
			default:
				if varcharReg.MatchString(fld.Type.Tag.Get("type")) {
					a := varcharReg.FindStringSubmatch(fld.Type.Tag.Get("type"))
					i, err := strconv.Atoi(a[1])
					if err != nil {
						panic(fmt.Sprintf("can't parse varchar value: %#v: %s of field %s", ftype, err.Error(), fld.Type.Name))
					}
					if i > 255 {
						panic(fmt.Sprintf("max number for varchar is 255, not %v in field %s", i, fld.Type.Name))
					}
					typ = VarChar(i)
				} else {
					panic(fmt.Sprintf("unknown type %#v of field %s", ftype, fld.Type.Name))
				}
			}
		} else {
			/*
				switch ff.Typ() {
				case "string":
					typ = VarChar(255)
				case "bool":
					typ = BoolType
				case "int":
					typ = IntType
				case "time":
					typ = TimeType
				case "[]string":
					typ = StringsType
				case "[]int":
					typ = IntsType
				default:
			*/
			panic(fmt.Sprintf("type: %#v has no corresponding pgsql.Type in field %s", ff.Typ(), fld.Type.Name))
			/*
				}
			*/
		}

		f := table.NewField(fname, typ)
		var isPkey bool
		//fflags := fld.Type.Tag.Get("pgsql.flags")
		//if fflags != "" {
		//	flgs := strings.Split(fflags, ",")
		if len(dbFlag) > 1 {

			//for _, fl := range flgs {
			for _, fl := range dbFlag[1:] {
				fl = strings.TrimSpace(fl)
				var fffl Flag
				switch fl {
				case "NULL":
					fffl = NullAllowed
				case "PKEY":
					isPkey = true
					fffl = PrimaryKey
				case "SERIAL":
					fffl = Serial
				case "UUIDGEN":
					fffl = UuidGenerate
				case "DELETE_CASCADE":
					fffl = OnDeleteCascade
				default:
					panic(fmt.Sprintf("unsupported flag: %#v in field %s", fl, fld.Type.Name))
				}
				f.Add(fffl)
			}
		}
		//}
		if ff.Default() != nil {
			f.Default = Sql(ff.Default().String())
		}
		if isPkey {
			table.PrimaryKey = append(table.PrimaryKey, f)
		}

		//fmt.Printf("adding field %#v, %#v, %s, %s\n", val.Type().String(), val.Type().Name(), fld.Type.Name, f.Name)
		//FieldRegistry.AddField(val.Type().String(), fld.Type.Name, f)
		FieldRegistry.AddField(valType, fld.Type.Name, f)
		//}
	}
	//	TableRegistry.AddTable(val.Type().String(), table)
	TableRegistry.AddTable(valType, table)
	stru.Each(fn)
	return nil
}

func MustRegisterTable(name string, ptrToFatStru interface{}) {
	err := RegisterTable(name, ptrToFatStru)
	if err != nil {
		panic(err.Error())
	}
}

func FieldOf(ff *fat.Field) *Field {
	return FieldRegistry.Field(ff.StructType(), ff.Name())
}

func TableOf(fatstruct interface{}) *Table {
	return TableRegistry.Table(typestring(fatstruct))
	//return TableRegistry.Table(reflect.TypeOf(fatstruct).String())
}

/*
type FatScanner struct {
	*fat.Field
}

func (ft *FatScanner) Scan(value interface{}) error {
	return ft.Field.Scan(fmt.Sprintf("%v", value))
}
*/
