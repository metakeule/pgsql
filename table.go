package pgsql

import (
	"fmt"
	"strings"
)

type Table struct {
	Name          string
	Schema        *Schema
	Fields        []*Field
	PrimaryKeySeq Sqler
	PrimaryKey    *Field
	Validations   []RowValidator
}

func NewTable(name string, options ...interface{}) *Table {
	t := &Table{
		Name:   name,
		Fields: []*Field{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *Schema:
			t.Schema = v
		case *Field:
			t.AddField(v)
		default:
			if val, ok := v.(RowValidator); ok {
				t.Validations = append(t.Validations, val)
			}
		}
	}
	return t
}

func (ø *Table) NewField(name string, options ...interface{}) (field *Field) {
	field = NewField(name, options...)
	ø.AddField(field)
	return
}

func (ø *Table) AddValidator(v ...RowValidator) {
	for _, val := range v {
		ø.Validations = append(ø.Validations, val)
	}
}

func (ø *Table) Validate(values map[*Field]interface{}) (errs map[Sqler]error) {
	errs = map[Sqler]error{}
	pkey := ø.PrimaryKey
	for _, f := range ø.Fields {
		// don't validate empty ids
		if f == pkey && values[f] == nil {
			continue
		}

		err := f.Validate(values[f])
		if err != nil {
			errs[f] = err
		}
	}
	for _, val := range ø.Validations {
		err := val.ValidateRow(values)
		if err != nil {
			errs[ø] = err
		}
	}
	return
}

func (ø *Table) createField(field *Field) string {
	if field.Is(PrimaryKey) {
		if field.Is(Serial) {
			return field.Name + " SERIAL PRIMARY KEY"
		}
		return field.Name + " PRIMARY KEY"
	}

	s := field.Name + " " + field.Type.String()
	if field.Default != nil {
		s += " DEFAULT " + string(field.Default.Sql())
	}
	if !field.Is(NullAllowed) {
		s += " NOT NULL "
	}

	if field.ForeignKey != nil {
		s += " REFERENCES " + string(field.ForeignKey.Table.Sql()) + `("` + field.ForeignKey.Name + `")`
		if field.Is(OnDeleteCascade) {
			s += " ON DELETE CASCADE"
		} else {
			s += " ON DELETE RESTRICT"
		}
	}
	return s
}

func (ø *Table) Create() SqlType {
	fs := []string{}
	for _, f := range ø.Fields {
		fs = append(fs, ø.createField(f))
	}
	str := fmt.Sprintf(
		"CREATE TABLE %s \n(%s) ", ø.Sql(), strings.Join(fs, ",\n"))
	return Sql(str)
}

//func (ø *Table) Alter() Sql {
//}

func (ø *Table) Drop() SqlType {
	return Sql("DROP TABLE " + string(ø.Sql()))
}

func (ø *Table) AddField(fields ...*Field) {
	for _, f := range fields {
		ø.Fields = append(ø.Fields, f)
		if f.Is(PrimaryKey) {
			ø.PrimaryKey = f
		}
		f.Table = ø
	}
}

func (ø *Table) Field(name string) (f *Field) {
	for _, ff := range ø.Fields {
		if ff.Name == name {
			return ff
		}
	}
	return
}

func (ø *Table) Sql() SqlType {
	if ø.Schema == nil {
		return Sql(fmt.Sprintf(`"%s"`, ø.Name))
	}
	return Sql(fmt.Sprintf(`%s."%s"`, ø.Schema.Sql(), ø.Name))
}
