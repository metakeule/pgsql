package pgsql

import (
	"fmt"
	"strconv"
	"time"
	// 	"encoding/xml"
	"github.com/metakeule/typeconverter"
	"strings"
)

type SqlType string

func Sql(s string) SqlType {
	return SqlType(s)
}

type Sqler interface {
	Sql() SqlType
}

func (ø SqlType) Sql() SqlType {
	return ø
}

func (ø SqlType) String() string {
	return string(ø)
}

func (ø SqlType) TypedValue() *TypedValue {
	return &TypedValue{TextType, ø}
}

var TypeNames = map[Type]string{
	TextType:        "text",
	IntType:         "int",
	FloatType:       "float",
	BoolType:        "bool",
	TimeStampTZType: "timestamptz",
	TimeStampType:   "timestamp",
	DateType:        "date",
	TimeType:        "time",
	XmlType:         "xml",
	IntsType:        "integer[]",
	StringsType:     "character varying[]",
}

var TypeCompatibles = map[Type][]Type{
	IntType:         []Type{IntType},
	IntsType:        []Type{IntsType, TextType},
	StringsType:     []Type{StringsType, TextType},
	FloatType:       []Type{IntType, FloatType},
	TextType:        []Type{TextType, XmlType},
	BoolType:        []Type{BoolType},
	DateType:        []Type{TextType, DateType, TimeType, TimeStampTZType, TimeStampType},
	TimeType:        []Type{TextType, DateType, TimeType, TimeStampTZType, TimeStampType},
	XmlType:         []Type{XmlType, TextType},
	TimeStampTZType: []Type{TextType, DateType, TimeType, TimeStampTZType, TimeStampType},
	TimeStampType:   []Type{TextType, DateType, TimeType, TimeStampTZType, TimeStampType},
}

func (ø Type) IsCompatible(other Type) bool {
	if (IsVarChar(ø) && IsVarChar(other)) ||
		(IsVarChar(ø) && other == TextType) ||
		(ø == TextType && IsVarChar(other)) {
		return true
	}
	compatible := TypeCompatibles[ø]
	for _, comp := range compatible {
		if comp == other {
			return true
		}
	}
	return false
}

type Type int

const (
	NullType Type = iota + 256 // starting from 256 to allow VarChar to have numbers from 1 to 255
	IntType
	FloatType
	TextType
	BoolType
	DateType
	TimeType
	XmlType
	TimeStampTZType
	TimeStampType
	IntsType
	StringsType
)

var TypeConverter = NewTypeConverter()

func Convert(in interface{}, out interface{}) (err error) { return TypeConverter.Convert(in, out) }

func ToSql(i interface{}) Sqler {
	if s, ok := i.(Sqler); ok {
		return s
	}
	out := &TypedValue{}
	err := Convert(i, out)
	if err != nil {
		panic("can't convert to sql: " + err.Error())
	}
	return out
}

func (ø Type) String() string { return TypeNames[ø] }
func (ø Type) Type() Type     { return ø }

var intInstance = int(0)
var int32Instance = int32(0)
var int64Instance = int64(0)
var intsInstance = []int{}
var float64Instance = float64(0)
var float32Instance = float32(0)
var stringInstance = string("")
var stringsInstance = []string{}
var jsonInstance = typeconverter.Json("")
var boolInstance = bool(true)
var timeInstance = time.Time{}
var mapInstance = map[string]interface{}{}
var arrInstance = []interface{}{}
var typedValueInstance = TypedValue{}
var sqlInstance = Sql("")
var typeInstance = Type(0)

type intsStringer []int

func (ø intsStringer) String() string {
	s := `{%s}`
	str := []string{}
	for _, i := range ø {
		str = append(str, fmt.Sprintf("%v", i))
	}
	return fmt.Sprintf(s, strings.Join(str, ","))
}

func (ø intsStringer) Ints() []int {
	return []int(ø)
}

type Intser interface {
	Ints() []int
}

type stringsStringer []string

func (ø stringsStringer) String() string {
	s := `{%s}`
	str := []string{}
	for _, i := range ø {
		str = append(str, fmt.Sprintf(`"%v"`, i))
	}
	return fmt.Sprintf(s, strings.Join(str, ","))
}

func (ø stringsStringer) Strings() []string {
	return []string(ø)
}

type Stringser interface {
	Strings() []string
}

/*
var intsMatcher = regexp.MustCompile(`^\{("?[0-9]"?,)*("?[0-9]"?)\}$`)
var stringsMatcher = regexp.MustCompile(`^\{([^,]*,)*([^,]*)\}$`)

func typedValForString(t string) (tv *TypedValue) {
	tv = &TypedValue{}
	if t[0:1] == "{" && t[len(t)-1:len(t)] == "}" {
		if len(intsMatcher.FindStringSubmatch(t)) > 0 {
			tv.PgType = IntsType
			tv.Value = intsStringer(stringToInts(t))
			return
		}
		if len(stringsMatcher.FindStringSubmatch(t)) > 0 {
			tv.PgType = StringsType
			tv.Value = stringsStringer(stringToStrings(t))
			return
		}
	}
	tv.PgType = TextType
	tv.Value = typeconverter.String(t)
	return
}
*/

type pgInterpretedString struct {
	typeconverter.StringType
}

func NewPgInterpretedString(s string) (ip *pgInterpretedString) {
	ip = &pgInterpretedString{}
	ip.StringType = typeconverter.String(s)
	return
}

func (ø *pgInterpretedString) Ints() (i []int) {
	str := ø.StringType.String()
	inner := str[1 : len(str)-1]
	a := strings.Split(inner, ",")
	for _, s := range a {
		ii, ſ := strconv.Atoi(strings.Trim(s, `"`))
		if ſ == nil {
			i = append(i, ii)
		}
	}
	return
	// return stringToInts(ø.StringType.String())
}

func (ø *pgInterpretedString) Strings() (ses []string) {
	str := ø.StringType.String()
	inner := str[1 : len(str)-1]
	a := strings.Split(inner, ",")
	for _, s := range a {
		s_tr := strings.Trim(s, `"`)
		ses = append(ses, s_tr)
	}
	return
}

//typeconverter.String

func NewTypeConverter() (ø *typeconverter.BasicConverter) {
	ø = typeconverter.New()

	inSwitch := func(from interface{}, to interface{}) (err error) {
		switch t := from.(type) {
		case int:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int(t)})
		case int32:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int(int(t))})
		case int64:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int64(t)})
		case float64:
			err = ø.Output.Dispatch(to, &TypedValue{FloatType, typeconverter.Float(t)})
		case float32:
			err = ø.Output.Dispatch(to, &TypedValue{FloatType, typeconverter.Float32(t)})
		case string:
			// err = ø.Output.Dispatch(to, typedValForString(t))
			err = ø.Output.Dispatch(to, &TypedValue{TextType, NewPgInterpretedString(t)})
			//err = ø.Output.Dispatch(to, &TypedValue{FloatType, typeconverter.String(t)})
		case bool:
			err = ø.Output.Dispatch(to, &TypedValue{BoolType, typeconverter.Bool(t)})
		case time.Time:
			err = ø.Output.Dispatch(to, &TypedValue{TimeStampTZType, typeconverter.Time(t)})
		case []int:
			err = ø.Output.Dispatch(to, &TypedValue{IntsType, intsStringer(t)})
		case []string:
			err = ø.Output.Dispatch(to, &TypedValue{StringsType, stringsStringer(t)})
		case *int:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int(*t)})
		case *int32:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int(int(*t))})
		case *int64:
			err = ø.Output.Dispatch(to, &TypedValue{IntType, typeconverter.Int64(*t)})
		case *float64:
			err = ø.Output.Dispatch(to, &TypedValue{FloatType, typeconverter.Float(*t)})
		case *float32:
			err = ø.Output.Dispatch(to, &TypedValue{FloatType, typeconverter.Float32(*t)})
		case *string:
			err = ø.Output.Dispatch(to, &TypedValue{TextType, NewPgInterpretedString(*t)})
			// err = ø.Output.Dispatch(to, &TypedValue{TextType, typeconverter.String(*t)})
			// err = ø.Output.Dispatch(to, typedValForString(*t))
		case *bool:
			err = ø.Output.Dispatch(to, &TypedValue{BoolType, typeconverter.Bool(*t)})
		case *time.Time:
			err = ø.Output.Dispatch(to, &TypedValue{TimeStampTZType, typeconverter.Time(*t)})
		case *TypedValue:
			err = ø.Output.Dispatch(to, t)
		case TypedValue:
			err = ø.Output.Dispatch(to, &t)
		case SqlType:
			err = ø.Output.Dispatch(to, t)
		case Type:
			err = ø.Output.Dispatch(to, t)
		default:
			err = ø.Output.Dispatch(to, &TypedValue{TextType, from.(typeconverter.Stringer)})
		}
		return
	}

	ø.Input.SetHandler(intInstance, inSwitch)
	ø.Input.SetHandler(int32Instance, inSwitch)
	ø.Input.SetHandler(int64Instance, inSwitch)
	ø.Input.SetHandler(float64Instance, inSwitch)
	ø.Input.SetHandler(float32Instance, inSwitch)
	ø.Input.SetHandler(stringInstance, inSwitch)
	ø.Input.SetHandler(boolInstance, inSwitch)
	ø.Input.SetHandler(timeInstance, inSwitch)
	ø.Input.SetHandler(&intInstance, inSwitch)
	ø.Input.SetHandler(&int32Instance, inSwitch)
	ø.Input.SetHandler(&int64Instance, inSwitch)
	ø.Input.SetHandler(&float64Instance, inSwitch)
	ø.Input.SetHandler(&float32Instance, inSwitch)
	ø.Input.SetHandler(&stringInstance, inSwitch)
	ø.Input.SetHandler(&boolInstance, inSwitch)
	ø.Input.SetHandler(&timeInstance, inSwitch)
	ø.Input.SetHandler(jsonInstance, inSwitch)
	ø.Input.SetHandler(typedValueInstance, inSwitch)
	ø.Input.SetHandler(&typedValueInstance, inSwitch)
	ø.Input.SetHandler(typeInstance, inSwitch)
	ø.Input.SetHandler(intsInstance, inSwitch)
	ø.Input.SetHandler(stringsInstance, inSwitch)

	outSwitch := func(out interface{}, in interface{}) (err error) {
		switch t := out.(type) {
		case *TypedValue:
			iTyped := in.(Valuer).TypedValue()
			oTyped := out.(*TypedValue)
			if int(oTyped.Type()) == 0 {
				*oTyped = *iTyped
				return
			}
			if iTyped.Type() == oTyped.Type() {
				*oTyped = *iTyped
			} else {
				if oTyped.Type().IsCompatible(iTyped.Type()) {
					oTyped.Value = iTyped.Value
				} else {
					return fmt.Errorf("value %s type %s is incompatible with type %s\n%s", iTyped.Value.String(), iTyped.Type().String(), oTyped.Type().String(), strings.Join(backtrace(), "\n"))
				}
			}
		case *bool:
			*out.(*bool) = in.(*TypedValue).Value.(typeconverter.Booler).Bool()
		case *int:
			*out.(*int) = in.(*TypedValue).Value.(typeconverter.Inter).Int()
		case *int64:
			*out.(*int64) = int64(in.(*TypedValue).Value.(typeconverter.Inter).Int())
		case *string:
			*out.(*string) = in.(typeconverter.Stringer).String()
		case *float64:
			*out.(*float64) = in.(*TypedValue).Value.(typeconverter.Floater).Float()
		case *float32:
			*out.(*float32) = float32(in.(*TypedValue).Value.(typeconverter.Floater).Float())
		case *time.Time:
			*out.(*time.Time) = in.(*TypedValue).Value.(typeconverter.Timer).Time()
		case *[]int:
			*out.(*[]int) = in.(*TypedValue).Value.(Intser).Ints()
		case *[]string:
			*out.(*[]string) = in.(*TypedValue).Value.(Stringser).Strings()
		case *SqlType:
			*out.(*SqlType) = in.(Sqler).Sql()
		case *Type:
			*out.(*Type) = in.(Typer).Type()
		default:
			return fmt.Errorf("can't convert to %#v: no converter found", t)
		}
		return
	}

	ø.Output.SetHandler(&intInstance, outSwitch)
	ø.Output.SetHandler(&int32Instance, outSwitch)
	ø.Output.SetHandler(&int64Instance, outSwitch)
	ø.Output.SetHandler(&float64Instance, outSwitch)
	ø.Output.SetHandler(&float32Instance, outSwitch)
	ø.Output.SetHandler(&stringInstance, outSwitch)
	ø.Output.SetHandler(&boolInstance, outSwitch)
	ø.Output.SetHandler(&timeInstance, outSwitch)
	ø.Output.SetHandler(&jsonInstance, outSwitch)
	ø.Output.SetHandler(&typedValueInstance, outSwitch)
	ø.Output.SetHandler(&typeInstance, outSwitch)
	ø.Output.SetHandler(&sqlInstance, outSwitch)
	ø.Output.SetHandler(&intsInstance, outSwitch)
	ø.Output.SetHandler(&stringsInstance, outSwitch)
	return
}

type Typer interface {
	Type() Type
}

type TypedValue struct {
	PgType Type
	Value  typeconverter.Stringer
}

type Valuer interface {
	TypedValue() *TypedValue
}

func (ø *TypedValue) TypedValue() *TypedValue {
	return ø
}

func (ø *TypedValue) IsNil() bool {
	if ø == nil || ø.Value == nil {
		return true
	}
	return false
}

func (ø *TypedValue) Sql() SqlType {
	if ø.IsNil() {
		return Sql("Null")
	}
	return Sql(fmt.Sprintf("'%s'::%s", ø.Value.String(), ø.PgType.String()))
}

func (ø *TypedValue) String() string { return ø.Value.String() }
func (ø *TypedValue) Type() Type     { return ø.PgType }

func VarChar(i int) Type {
	if i > 255 {
		panic("varchar may not be larger than 255")
	}
	if i < 1 {
		panic("varchar may not be smaller than 1")
	}
	t := Type(i)
	TypeNames[t] = fmt.Sprintf("varchar(%v)", i)
	return t
}

// is the type a varchar
func IsVarChar(t Type) bool {
	if i := int(t); i < 256 && i > 0 {
		return true
	}
	return false
}
