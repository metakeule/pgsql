package pgsql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PreValidate func(*Row) error
type PostValidate func(*Row) error
type PreGet func(*Row) error
type PostGet func(*Row) error
type PreInsert func(*Row) error
type PostInsert func(*Row) error
type PreUpdate func(*Row) error
type PostUpdate func(*Row) error
type PreDelete func(*Row) error
type PostDelete func(*Row) error

type Row struct {
	*Table
	DB           DB
	Tx           *sql.Tx
	values       map[*Field]*TypedValue
	aliasValues  map[*AsStruct]*TypedValue
	setErrors    []error
	Debug        bool
	LastSql      string
	PreValidate  []PreValidate
	PostValidate []PostValidate
	PreGet       []PreGet
	PostGet      []PostGet
	PreInsert    []PreInsert
	PostInsert   []PostInsert
	PreUpdate    []PreUpdate
	PostUpdate   []PostUpdate
	PreDelete    []PreDelete
	PostDelete   []PostDelete
}

func NewRow(db RowDB, table *Table, hooks ...interface{}) (ø *Row) {
	ø = &Row{
		Table:        table,
		setErrors:    []error{},
		PreValidate:  []PreValidate{},
		PostValidate: []PostValidate{},
		PreGet:       []PreGet{},
		PostGet:      []PostGet{},
		PreInsert:    []PreInsert{},
		PostInsert:   []PostInsert{},
		PreUpdate:    []PreUpdate{},
		PostUpdate:   []PostUpdate{},
		PreDelete:    []PreDelete{},
		PostDelete:   []PostDelete{},
	}

	tx, ok := db.(*sql.Tx)
	if ok {
		ø.SetTransaction(tx)
	} else {
		ø.SetDB(db.(DB))
	}

	for _, h := range hooks {
		switch hook := h.(type) {
		case PreValidate:
			ø.PreValidate = append(ø.PreValidate, hook)
		case PostValidate:
			ø.PostValidate = append(ø.PostValidate, hook)
		case PreGet:
			ø.PreGet = append(ø.PreGet, hook)
		case PostGet:
			ø.PostGet = append(ø.PostGet, hook)
		case PreInsert:
			ø.PreInsert = append(ø.PreInsert, hook)
		case PostInsert:
			ø.PostInsert = append(ø.PostInsert, hook)
		case PreUpdate:
			ø.PreUpdate = append(ø.PreUpdate, hook)
		case PostUpdate:
			ø.PostUpdate = append(ø.PostUpdate, hook)
		case PreDelete:
			ø.PreDelete = append(ø.PreDelete, hook)
		case PostDelete:
			ø.PostDelete = append(ø.PostDelete, hook)
		default:
			panic("unknown hook type: " + fmt.Sprintf("%#v", hook))
		}
	}
	ø.clearValues()
	return
}

// the parameters should be given in pairs of
// *Field and *value
func (ø *Row) Get(o ...interface{}) {
	for i := 0; i < len(o); i = i + 2 {
		switch field := o[i].(type) {
		case *Field:
			//field := o[i].(*Field)
			res := o[i+1]
			if ø.values[field] == nil {
				continue
			}
			err := Convert(ø.values[field], res)
			if err != nil {
				panic(
					"can't convert " +
						field.Name +
						" of table " +
						field.Table.Name +
						" to " +
						fmt.Sprintf("%#v", res) +
						": " +
						err.Error())
			}
		case *AsStruct:
			//as := o[i].(*AsStruct)
			res := o[i+1]
			if ø.aliasValues[field] == nil {
				continue
			}
			err := Convert(ø.aliasValues[field], res)
			if err != nil {
				panic(
					"can't convert " +
						field.As +
						" of alias field " +
						field.Sql() +
						" to " +
						fmt.Sprintf("%#v", res) +
						": " +
						err.Error())
			}
		default:
			panic(fmt.Sprintf("unsupported type %#v\n", o[i]))
		}
	}
}

func (ø *Row) GetString(field interface{}) (s string) {
	ø.Get(field, &s)
	return
}

func (ø *Row) ValidateAll() (errs map[Sqler]error) {
	return ø.Table.Validate(ø.Values())
}

// the parameters should be given in pairs of
// *Field and value
func (ø *Row) Set(o ...interface{}) (err error) {
	err = ø.set(o...)
	if err != nil {
		return err
	}
	err = ø.validate()
	if err != nil {
		return err
	}
	return
}

// set field to null
func (ø *Row) SetNull(field *Field) {
	ø.values[field].Value = nil
}

// unset the given fields
func (ø *Row) Unset(o ...*Field) {
	for _, f := range o {
		delete(ø.values, f)
	}
}

func (ø *Row) set(o ...interface{}) (err error) {
	for i := 0; i < len(o); i = i + 2 {
		field := o[i].(*Field)
		var tv *TypedValue
		tv, err = field.Value(o[i+1])
		/*
			if o[i+1] == nil {
				if field.Is(NullAllowed) {
					ø.values[field] = &TypedValue{PgType: field.Type}
					continue
				} else {
					e := fmt.Errorf("error when setting field %s to value %#v: Null is not allowed for this field\n", field.Sql(), o[i+1])
					ø.setErrors = append(ø.setErrors, e)
					return e
				}

			}
			tv := &TypedValue{PgType: field.Type}
			err = Convert(o[i+1], tv)
		*/
		if err != nil {
			ø.setErrors = append(ø.setErrors, err)
			return
		}
		ø.values[field] = tv
	}
	return
}

var _ = strings.Replace

func (ø *Row) validate() (err error) {
	if len(ø.setErrors) > 0 {
		errString := []string{}
		for _, e := range ø.setErrors {
			errString = append(errString, fmt.Sprintf("\t%s", e.Error()))
		}
		return fmt.Errorf("%s\n", strings.Join(errString, "\n"))
	}

	errs := ø.ValidateAll()
	if len(errs) > 0 {
		errString := []string{}
		for k, e := range errs {
			errString = append(errString, fmt.Sprintf("\tValidation Error in %s: %s\n", k.Sql(), e.Error()))
		}
		return fmt.Errorf("%s\n", strings.Join(errString, "\n"))
	}
	return
}

func (ø *Row) Validate() (err error) {
	for _, hook := range ø.PreValidate {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	err = ø.validate()
	if err != nil {
		return
	}
	for _, hook := range ø.PostValidate {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	return nil
}

func (ø *Row) Save() (err error) {
	if len(ø.PrimaryKey) != 1 {
		panic("save should only be called for single primary keys, try update or insert directly")
	}
	err = ø.Validate()
	if err != nil {
		return fmt.Errorf("Can't save row of %s:\n%s\n", ø.Table.Sql(), err.Error())
	}
	ø.setErrors = []error{}
	if ø.HasId() {
		err = ø.Update()
	} else {
		err = ø.Insert()
	}
	return
}

func (ø *Row) HasId() bool {
	for _, pkey := range ø.PrimaryKey {
		if ø.values[pkey].IsNil() {
			return false
		}
	}
	return true
}

func (ø *Row) Fill(m map[string]interface{}) error {
	ø.values = map[*Field]*TypedValue{}
	for k, v := range m {
		e := ø.set(ø.Table.Field(k), v)

		if e != nil {
			fmt.Printf("error while filling %s with %v: %s\n", k, v, e.Error())
			return e
		}
	}
	err := ø.validate()
	if err != nil {
		return err
	}
	return nil
}

func (ø *Row) Properties() (m map[string]interface{}) {
	m = map[string]interface{}{}
	for field, val := range ø.values {
		m[field.Name] = val.Value
	}
	for as, val := range ø.aliasValues {
		m[as.As] = val.Value
	}
	return
}

func (ø *Row) AsStrings() (m map[string]string) {
	m = map[string]string{}
	// fmt.Printf("values: %#v\n", ø.values)
	for field, val := range ø.values {
		// fmt.Printf("key: %#v val: %#v\n", field.Sql(), val)
		var s string
		err := Convert(val, &s)
		if err != nil {
			panic("can't convert to string: " + err.Error())
		}
		m[field.Name] = s
	}
	for as, val := range ø.aliasValues {
		var s string
		err := Convert(val, &s)
		if err != nil {
			panic("can't convert to string: " + err.Error())
		}
		m[as.As] = s
	}
	return
}

func (ø *Row) Reset() {
	ø.clearValues()
	ø.setErrors = []error{}
}

func (ø *Row) clearValues() {
	ø.values = map[*Field]*TypedValue{}
	if len(ø.PrimaryKey) > 0 {
		for _, pkey := range ø.PrimaryKey {
			ø.values[pkey] = &TypedValue{PgType: pkey.Type}
		}
	}
	ø.aliasValues = map[*AsStruct]*TypedValue{}
}

// vals must be in the order of ø.PrimaryKey
func (ø *Row) SetId(vals ...string) (err error) {
	for i, val := range vals {
		ø.values[ø.PrimaryKey[i]] = &TypedValue{ø.PrimaryKey[i].Type, NewPgInterpretedString(val)}
		/*
			err = Convert(val, ø.values[ø.PrimaryKey[i]])
			if err != nil {
				return
			}
		*/
	}
	return
}

func (ø *Row) Id() (vals []SqlType) {
	//var idVal SqlType
	vals = []SqlType{}
	for _, pkey := range ø.PrimaryKey {
		var val SqlType
		err := Convert(ø.values[pkey], &val)
		if err != nil {
			panic("can't get id for table " + ø.Table.Name)
		}
		vals = append(vals, val)
	}
	return
}

type Rows struct {
	*sql.Rows // inherits from *sql.Rows and is fully compatible
	row       *Row
	Fields    []interface{}
}

// scan the result into a row
func (ø *Rows) ScanRow() (row *Row, ſ error) {
	ſ = ø.row.Scan(ø.Rows, ø.Fields...)
	if ſ == nil {
		row = ø.row
	}
	return
}

// call fn on each row
func (ø *Row) Each(fn func(*Row) error, options ...interface{}) (err error) {
	var rows *Rows
	defer rows.Close()
	rows, err = ø.Find(options...)
	if err != nil {
		return
	}
	for rows.Next() {
		var r *Row
		r, err = rows.ScanRow()
		if err != nil {
			return
		}
		err = fn(r)
		if err != nil {
			return
		}
	}
	return
}

// return the first result
func (ø *Row) Any(options ...interface{}) (r *Row, err error) {
	var rows *Rows
	rows, err = ø.Find(options...)
	defer rows.Close()
	if err != nil {
		return
	}
	rows.Next()
	r, err = rows.ScanRow()
	return
}

func (ø *Row) Find(options ...interface{}) (rows *Rows, err error) {
	sel := ø.selectquery(options...)
	r, err := ø.Query(sel)

	if err != nil {
		return
	}

	rows = &Rows{
		Rows:   r,
		row:    ø,
		Fields: []interface{}{},
	}

	for _, f := range sel.Fields {
		rows.Fields = append(rows.Fields, f)
	}

	for _, aliasF := range sel.FieldsWithAlias {
		//rows.Fields = append(rows.Fields, NewField(aliasF.As, aliasF.Type))
		rows.Fields = append(rows.Fields, aliasF)
	}

	return
}

func (ø *Row) Scan(row *sql.Rows, fields ...interface{}) (err error) {
	//ø.clearValues()
	ø.values = map[*Field]*TypedValue{}
	ø.aliasValues = map[*AsStruct]*TypedValue{}
	for _, hook := range ø.PreGet {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	scanF := []interface{}{}
	for _, field := range fields {
		switch f := field.(type) {
		case *Field:
			// make default values and append them
			switch f.Type {
			case IntType:
				if f.Is(NullAllowed) {
					var inNull sql.NullInt64
					scanF = append(scanF, &inNull)
				} else {
					var in int
					scanF = append(scanF, &in)
				}
			case FloatType:
				if f.Is(NullAllowed) {
					var flNull sql.NullFloat64
					scanF = append(scanF, &flNull)
				} else {
					var fl float32
					scanF = append(scanF, &fl)
				}
			case BoolType:
				if f.Is(NullAllowed) {
					var blNull sql.NullBool
					scanF = append(scanF, &blNull)
				} else {
					var bl bool
					scanF = append(scanF, &bl)
				}

			case TimeStampTZType, TimeStampType, DateType, TimeType:
				var t time.Time
				scanF = append(scanF, &t)
			default:
				if f.Is(NullAllowed) {
					var sNull sql.NullString
					scanF = append(scanF, &sNull)
				} else {
					var s string
					scanF = append(scanF, &s)
				}
			}
		case *AsStruct:
			// make default values and append them
			switch f.Type {
			case IntType:
				var in int
				scanF = append(scanF, &in)
			case FloatType:
				var fl float32
				scanF = append(scanF, &fl)
			case BoolType:
				var bl bool
				scanF = append(scanF, &bl)
			case TimeStampTZType, TimeStampType, DateType, TimeType:
				var t time.Time
				scanF = append(scanF, &t)
			default:
				var s string
				scanF = append(scanF, &s)
			}
		}

	}
	err = row.Scan(scanF...)
	if err != nil {
		return
	}
	for i, res := range scanF {
		fi := fields[i]
		switch f := fi.(type) {
		case *Field:
			tv := &TypedValue{PgType: f.Type}
			switch v := res.(type) {
			case *sql.NullInt64:
				if (*v).Valid {
					e := Convert((*v).Int64, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullFloat64:
				if (*v).Valid {
					e := Convert((*v).Float64, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullBool:
				if (*v).Valid {
					e := Convert((*v).Bool, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullString:
				if (*v).Valid {
					e := Convert((*v).String, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			default:
				e := Convert(v, tv)
				if e != nil {
					err = e
					return
				}
			}
			ø.values[f] = tv
		case *AsStruct:
			tv := &TypedValue{PgType: f.Type}
			switch v := res.(type) {
			case *sql.NullInt64:
				if (*v).Valid {
					e := Convert((*v).Int64, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullFloat64:
				if (*v).Valid {
					e := Convert((*v).Float64, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullBool:
				if (*v).Valid {
					e := Convert((*v).Bool, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			case *sql.NullString:
				if (*v).Valid {
					e := Convert((*v).String, tv)
					if e != nil {
						err = e
						return
					}
				} else {
					continue
				}
			default:
				e := Convert(v, tv)
				if e != nil {
					err = e
					return
				}
			}
			ø.aliasValues[f] = tv
		}
	}

	for _, hook := range ø.PostGet {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	return
}

func (ø *Row) Reload() (err error) {
	if !ø.HasId() {
		return fmt.Errorf("can't reload, have no id")
	}
	var ids []string

	for _, pk := range ø.PrimaryKey {
		var id string
		ø.Get(pk, &id)
		ids = append(ids, id)
	}
	fmt.Printf("loaded ids to: %#v", ids)
	return ø.Load(ids...)
}

func (ø *Row) Load(ids ...string) (err error) {
	f := ø.Table.Fields
	err = ø.SetId(ids...)
	if err != nil {
		return
	}

	//ø.Debug = true

	var conds []Sqler
	is := ø.Id()

	for i, pk := range ø.PrimaryKey {
		conds = append(conds, Equals(pk, is[i]))
	}

	row, err := ø.Select(f, Where(And(conds...)))
	if err != nil {
		return
	}
	if !row.Next() {
		row.Close()
		return fmt.Errorf("no row for %v", ids)
	}

	fs := []interface{}{}
	for _, ff := range f {
		fs = append(fs, ff)
	}

	err = ø.Scan(row, fs...)
	row.Close()
	return
}

// has to be invoked directly
func (ø *Row) Update(pkVals ...interface{}) (err error) {
	err = ø.Validate()
	if err != nil {
		return fmt.Errorf("Can't update row of %s:\n%s\n", ø.Table.Sql(), err.Error())
	}
	ø.setErrors = []error{}

	for _, hook := range ø.PreUpdate {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	vals := ø.typedValues()
	// fmt.Println(vals)
	var cond []Sqler
	if len(ø.PrimaryKey) == 1 {
		ids := ø.Id()
		delete(vals, ø.PrimaryKey[0])
		cond = append(cond, Equals(ø.PrimaryKey[0], ids[0]))
	} else {
		if len(pkVals) != len(ø.PrimaryKey) {
			n := []string{}
			for _, pk := range ø.PrimaryKey {
				n = append(n, pk.Name)
			}
			panic(fmt.Sprintf("number of vals for multiple primary keys does not match: given: %v, needed: %v (%s)\n", len(pkVals), len(ø.PrimaryKey), strings.Join(n, ", ")))
		}
		for i, pk := range ø.PrimaryKey {
			cond = append(cond, Equals(pk, pkVals[i]))
		}
	}
	/*
		else {
			ids := ø.Id()
			for i, pkey := range ø.PrimaryKey {
				//delete(vals, pkey)
				cond = append(cond, Equals(pkey, ids[i]))
			}
		}
	*/

	//w := []*Condition{}
	//conditions = append(conditions, And(cond...))
	w := And(cond...)

	u := Update(
		ø.Table,
		vals,
		//Where(Equals(ø.PrimaryKey, ø.Id())))
		Where(w))
	// fmt.Println(u.Sql())
	_, err = ø.Exec(u)
	for _, hook := range ø.PostUpdate {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	return
}

func (ø *Row) Values() (vals map[*Field]interface{}) {
	vals = map[*Field]interface{}{}
	for k, v := range ø.values {
		if !v.IsNil() {
			vals[k] = v.Value
		}
	}
	return
}

func (ø *Row) AliasValues() (vals map[*AsStruct]interface{}) {
	vals = map[*AsStruct]interface{}{}
	for k, v := range ø.aliasValues {
		if !v.IsNil() {
			vals[k] = v.Value
		}
	}
	return
}

func (ø *Row) typedValues() (vals map[*Field]interface{}) {
	vals = map[*Field]interface{}{}
	//pkey := ø.PrimaryKey
	for k, v := range ø.values {
		if ø.IsPrimaryKey(k) && v.IsNil() {
			continue
		}
		vals[k] = v
	}
	return
}

func (ø *Row) Insert() (err error) {
	err = ø.Validate()
	if err != nil {
		return fmt.Errorf("Can't insert row of %s:\n%s\n", ø.Table.Sql(), err.Error())
	}
	ø.setErrors = []error{}
	for _, hook := range ø.PreInsert {
		err := hook(ø)
		if err != nil {
			return err
		}
	}
	u := InsertMap(ø.Table, ø.typedValues())
	//_, err = ø.Exec(u)

	/*
		r, err := ø.DB.Exec(u.String())
		if err != nil {
			return err
		}
	*/

	if len(ø.PrimaryKey) == 1 {
		// t := ø.PrimaryKey[0].Type
		var i string
		// ø.setSearchPath()
		//r, err := ø.DB.Query(u.String())
		r, err := ø.Query(u)
		if err != nil || r == nil {
			return err
		}
		r.Next()
		//err := ø.DB.QueryRow(u.String()).Scan(&i)
		err = r.Scan(&i)
		r.Close()
		//i := 0
		//i, err := r.LastInsertId()
		//fmt.Println(ø.LastSql)
		//fmt.Printf("last id %#v\n", i)
		//fmt.Printf("type %#v\n", ø.PrimaryKey[0].Type)
		if err != nil {
			fmt.Printf("Error while inserting: %v\n", err.Error())
			return err
		}
		//r.Next()
		//r.Scan(&i)
		tv := &TypedValue{ø.PrimaryKey[0].Type, NewPgInterpretedString(i)}
		// Convert(i, tv)
		//fmt.Printf("converted %#v\n", tv)
		ø.values[ø.PrimaryKey[0]] = tv
	} else {
		_, err := ø.Exec(u)
		if err != nil {
			return err
		}
	}
	for _, hook := range ø.PostInsert {
		err := hook(ø)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ø *Row) Delete() (err error) {
	for _, hook := range ø.PreDelete {
		err = hook(ø)
		if err != nil {
			return
		}
	}

	cond := []Sqler{}

	ids := ø.Id()
	for i, pk := range ø.PrimaryKey {
		cond = append(cond, Equals(pk, ids[i]))
	}

	w := And(cond...)

	u := Delete(ø.Table, Where(w))
	_, err = ø.Exec(u)
	for _, hook := range ø.PostDelete {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	return
}

func (ø *Row) selectquery(objects ...interface{}) (s *SelectQuery) {
	snew := make([]interface{}, len(objects)+1)
	snew[0] = ø.Table
	for i, o := range objects {
		snew[i+1] = o
	}
	s = Select(snew...).(*SelectQuery)
	if len(s.Fields) == 0 && len(s.FieldsWithAlias) == 0 {
		s.Fields = ø.Fields
	}
	return
}

func (ø *Row) Select(objects ...interface{}) (r *sql.Rows, err error) {
	s := ø.selectquery(objects...)
	r, err = ø.Query(s)
	return
}

func (ø *Row) IsValid(f string, value interface{}) bool {
	field := ø.Table.Field(f)
	tv := &TypedValue{PgType: field.Type}
	err := Convert(value, tv)
	if err != nil {
		return false
	}
	err = field.Validate(tv)
	if err == nil {
		return true
	}
	return false
}

func (ø *Row) Begin() (tx *sql.Tx, ſ error) {
	if ø.isTransaction() {
		tx = ø.Tx
	} else {
		tx, ſ = ø.DB.Begin()
		ø.SetTransaction(tx)
	}
	return
}

func (ø *Row) Rollback() (ſ error) {
	return ø.Tx.Rollback()
}

func (ø *Row) Commit() (ſ error) {
	return ø.Tx.Commit()
}

func (ø *Row) isTransaction() (is bool) {
	return ø.Tx != nil
}

func (ø *Row) setSearchPath() {
	if !ø.isTransaction() {
		if ø.Table.Schema != nil {
			schemaName := ø.Table.Schema.Name
			sql := `SET search_path = "` + schemaName + `"`
			if ø.Debug {
				fmt.Println(sql)
			}
			_, _ = ø.DB.Exec(sql)
		}
	}
}

func (ø *Row) Exec(query Query, args ...interface{}) (r sql.Result, err error) {
	ø.setSearchPath()
	//ø.Debug = true
	if ø.Debug {
		fmt.Println(query.String())
	}
	ø.LastSql = query.String()
	if ø.isTransaction() {
		r, err = ø.Tx.Exec(query.String(), args...)
	} else {
		r, err = ø.DB.Exec(query.String(), args...)
	}
	return
}

func (ø *Row) Prepare(query Query) (r *sql.Stmt, err error) {
	s := query.Sql()
	// ø.Debug = true
	if ø.Debug {
		fmt.Println(s.String())
	}
	ø.LastSql = s.String()
	ø.setSearchPath()
	if ø.isTransaction() {
		r, err = ø.Tx.Prepare(s.String())
	} else {
		r, err = ø.DB.Prepare(s.String())
	}
	return
}

func (ø *Row) Query(query Query, args ...interface{}) (r *sql.Rows, err error) {
	s := query.Sql()
	ø.LastSql = s.String()
	ø.setSearchPath()
	//ø.Debug = true
	if ø.Debug {
		fmt.Println(s.String())
	}
	if ø.isTransaction() {
		r, err = ø.Tx.Query(s.String(), args...)
	} else {
		r, err = ø.DB.Query(s.String(), args...)
	}
	return
}

func (ø *Row) QueryRow(query Query, args ...interface{}) (r *sql.Row) {
	//ø.Debug = true
	if ø.Debug {
		fmt.Println(query.String())
	}
	s := query.Sql()
	ø.LastSql = s.String()
	ø.setSearchPath()
	if ø.isTransaction() {
		r = ø.Tx.QueryRow(s.String(), args...)
	} else {
		r = ø.DB.QueryRow(s.String(), args...)
	}
	return
}

func (ø *Row) SetDB(db DB) {
	ø.DB = db
}

func (ø *Row) SetTransaction(tx *sql.Tx) {
	ø.Tx = tx
}
