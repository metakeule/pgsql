package pgsql

import (
	"database/sql"
	"fmt"
	"strings"
)

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type Row struct {
	*Table
	DB        DB
	values    map[*Field]*TypedValue
	setErrors []error
}

func NewRow(db DB, table *Table) (ø *Row) {
	ø = &Row{Table: table, setErrors: []error{}}
	ø.SetDB(db)
	ø.clearValues()
	return
}

// the parameters should be given in pairs of
// *Field and *value
func (ø *Row) Get(o ...interface{}) {
	for i := 0; i < len(o); i = i + 2 {
		field := o[i].(*Field)
		res := o[i+1]
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
	err = ø.Validate()
	if err != nil {
		return err
	}
	return
}

func (ø *Row) set(o ...interface{}) (err error) {
	for i := 0; i < len(o); i = i + 2 {
		field := o[i].(*Field)
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

func (ø *Row) Validate() (err error) {
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
	err := ø.Validate()
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

func (ø *Row) Load(id int) (err error) {
	ø.clearValues()
	f := ø.Table.Fields
	err = ø.SetId(id)
	if err != nil {
		return
	}
	row, err := ø.Select(f, Where(Equals(ø.PrimaryKey, ø.Id())))
	if !row.Next() {
		return fmt.Errorf("no row for %v", id)
	}
	scanF := []interface{}{}
	for _, field := range f {
		// make default values and append them
		switch field.Type {
		case IntType:
			in := int(0)
			scanF = append(scanF, &in)
		case FloatType:
			fl := float32(0.0)
			scanF = append(scanF, &fl)
		default:
			s := ""
			scanF = append(scanF, &s)
		}
	}
	err = row.Scan(scanF...)
	if err != nil {
		return
	}
	for i, v := range scanF {
		tv := TypedValue{PgType: f[i].Type}
		e := Convert(v, &tv)
		if e != nil {
			err = e
			return
		}
	}
	return
}

func (ø *Row) Update() (err error) {
	vals := ø.typedValues()
	delete(vals, ø.PrimaryKey)
	u := Update(
		ø.Table,
		vals,
		Where(Equals(ø.PrimaryKey, ø.Id())))

	_, err = ø.Query(u)
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
	for k, v := range ø.values {
		if !v.IsNil() {
			vals[k] = v
		}
	}
	return
}

func (ø *Row) Insert() (err error) {
	u := InsertMap(ø.Table, ø.typedValues())
	_, err = ø.Query(u)
	return
}

func (ø *Row) Delete() (err error) {
	u := Delete(ø.Table, Where(Equals(ø.PrimaryKey, ø.Id())))
	_, err = ø.Query(u)
	return
}

func (ø *Row) Select(objects ...interface{}) (r *sql.Rows, err error) {
	snew := make([]interface{}, len(objects)+1)
	snew[0] = ø.Table
	for i, o := range objects {
		snew[i+1] = o
	}
	s := Select(snew...)
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

func (ø *Row) Exec(query Query, args ...interface{}) (r sql.Result, err error) {
	r, err = ø.DB.Exec(query.String(), args...)
	return
}

func (ø *Row) Prepare(query Query) (r *sql.Stmt, err error) {
	s := query.Sql()
	r, err = ø.DB.Prepare(s.String())
	return
}

func (ø *Row) Query(query Query, args ...interface{}) (r *sql.Rows, err error) {
	s := query.Sql()
	r, err = ø.DB.Query(s.String(), args...)
	return
}

func (ø *Row) QueryRow(query Query, args ...interface{}) (r *sql.Row) {
	s := query.Sql()
	r = ø.DB.QueryRow(s.String(), args...)
	return
}

func (ø *Row) SetDB(db DB) {
	ø.DB = db
}
