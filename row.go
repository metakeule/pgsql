package pgsql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

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
	values       map[*Field]*TypedValue
	setErrors    []error
	Debug        bool
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

func NewRow(db DB, table *Table, hooks ...interface{}) (ø *Row) {
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
	ø.SetDB(db)
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
		field := o[i].(*Field)
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
	}
}

func (ø *Row) GetString(field *Field) (s string) {
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
		if err != nil {
			e := fmt.Errorf("error when setting field %s to value %#v: %s\n", field.Sql(), o[i+1], err.Error())
			ø.setErrors = append(ø.setErrors, e)
			return e
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
	pkey := ø.PrimaryKey
	if ø.values[pkey].IsNil() {
		return false
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
	return
}

func (ø *Row) AsStrings() (m map[string]string) {
	m = map[string]string{}
	for field, val := range ø.values {
		var s string
		err := Convert(val, &s)
		if err != nil {
			panic("can't convert to string: " + err.Error())
		}
		m[field.Name] = s
	}
	return
}

func (ø *Row) Reset() {
	ø.clearValues()
	ø.setErrors = []error{}
}

func (ø *Row) clearValues() {
	pkey := ø.PrimaryKey
	ø.values = map[*Field]*TypedValue{pkey: &TypedValue{PgType: pkey.Type}}
}

func (ø *Row) SetId(id int) (err error) {
	err = Convert(id, ø.values[ø.PrimaryKey])
	if err != nil {
		return
	}
	return
}

func (ø *Row) Id() SqlType {
	var idVal SqlType
	err := Convert(ø.values[ø.PrimaryKey], &idVal)
	if err != nil {
		panic("can't get id for table " + ø.Table.Name)
	}
	return idVal
}

type RowIterator struct {
	*Row
	Fields  []*Field
	sqlRows *sql.Rows
}

func (ø *RowIterator) Next() (r *Row, err error) {
	if ø.sqlRows.Next() {
		err = ø.Row.Scan(ø.sqlRows, ø.Fields...)
		r = ø.Row
		if err != nil {
			return
		}
		return
	}
	r = nil
	return
}

func (ø *Row) Find(options ...interface{}) (iter *RowIterator, err error) {
	sel := ø.selectquery(options...)
	r, err := ø.Query(sel)

	if err != nil {
		return
	}

	iter = &RowIterator{
		Row:     ø,
		Fields:  sel.Fields,
		sqlRows: r,
	}
	for _, aliasF := range sel.FieldsWithAlias {
		iter.Fields = append(iter.Fields, NewField(aliasF.As, aliasF.Type))
	}

	return
}

func (ø *Row) Scan(row *sql.Rows, fields ...*Field) (err error) {
	ø.clearValues()
	for _, hook := range ø.PreGet {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	scanF := []interface{}{}
	for _, field := range fields {
		// make default values and append them
		switch field.Type {
		case IntType:
			if field.Is(NullAllowed) {
				var inNull sql.NullInt64
				scanF = append(scanF, &inNull)
			} else {
				var in int
				scanF = append(scanF, &in)
			}
		case FloatType:
			if field.Is(NullAllowed) {
				var flNull sql.NullFloat64
				scanF = append(scanF, &flNull)
			} else {
				var fl float32
				scanF = append(scanF, &fl)
			}
		case BoolType:
			if field.Is(NullAllowed) {
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
			if field.Is(NullAllowed) {
				var sNull sql.NullString
				scanF = append(scanF, &sNull)
			} else {
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
		f := fields[i]
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
	}

	for _, hook := range ø.PostGet {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	return
}

func (ø *Row) Load(id int) (err error) {
	f := ø.Table.Fields
	err = ø.SetId(id)
	if err != nil {
		return
	}
	row, err := ø.Select(f, Where(Equals(ø.PrimaryKey, ø.Id())))
	if !row.Next() {
		return fmt.Errorf("no row for %v", id)
	}

	return ø.Scan(row, f...)
}

func (ø *Row) Update() (err error) {
	for _, hook := range ø.PreUpdate {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	vals := ø.typedValues()
	//fmt.Println(vals)
	delete(vals, ø.PrimaryKey)
	u := Update(
		ø.Table,
		vals,
		Where(Equals(ø.PrimaryKey, ø.Id())))
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

func (ø *Row) typedValues() (vals map[*Field]interface{}) {
	vals = map[*Field]interface{}{}
	pkey := ø.PrimaryKey
	for k, v := range ø.values {
		if k == pkey && v.IsNil() {
			continue
		}
		vals[k] = v
	}
	return
}

func (ø *Row) Insert() error {
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
	i := 0
	if ø.Debug {
		fmt.Println(u.String())
	}
	// ø.setSearchPath()
	//r, err := ø.DB.Query(u.String())
	r, err := ø.Query(u)
	if err != nil || r == nil {
		return err
	}
	r.Next()
	//err := ø.DB.QueryRow(u.String()).Scan(&i)
	err = r.Scan(&i)
	//i := 0
	//i, err := r.LastInsertId()
	if err != nil {
		return err
	}
	//r.Next()
	//r.Scan(&i)
	tv := &TypedValue{PgType: ø.PrimaryKey.Type}
	Convert(i, tv)
	ø.values[ø.PrimaryKey] = tv
	for _, hook := range ø.PostInsert {
		err = hook(ø)
		if err != nil {
			return err
		}
	}
	return err
}

func (ø *Row) Delete() (err error) {
	for _, hook := range ø.PreDelete {
		err = hook(ø)
		if err != nil {
			return
		}
	}
	u := Delete(ø.Table, Where(Equals(ø.PrimaryKey, ø.Id())))
	_, err = ø.Query(u)
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
	return Select(snew...).(*SelectQuery)
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

func (ø *Row) setSearchPath() {
	schemaName := ø.Table.Schema.Name
	_, _ = ø.DB.Exec(`SET search_path = "` + schemaName + `"`)
}

func (ø *Row) Exec(query Query, args ...interface{}) (r sql.Result, err error) {
	if ø.Debug {
		fmt.Println(query.String())
	}
	ø.setSearchPath()
	r, err = ø.DB.Exec(query.String(), args...)
	return
}

func (ø *Row) Prepare(query Query) (r *sql.Stmt, err error) {
	s := query.Sql()
	if ø.Debug {
		fmt.Println(s.String())
	}
	ø.setSearchPath()
	r, err = ø.DB.Prepare(s.String())
	return
}

func (ø *Row) Query(query Query, args ...interface{}) (r *sql.Rows, err error) {
	s := query.Sql()
	if ø.Debug {
		fmt.Println(s.String())
	}
	ø.setSearchPath()
	r, err = ø.DB.Query(s.String(), args...)
	return
}

func (ø *Row) QueryRow(query Query, args ...interface{}) (r *sql.Row) {
	if ø.Debug {
		fmt.Println(query.String())
	}
	s := query.Sql()
	ø.setSearchPath()
	r = ø.DB.QueryRow(s.String(), args...)
	return
}

func (ø *Row) SetDB(db DB) {
	ø.DB = db
}
