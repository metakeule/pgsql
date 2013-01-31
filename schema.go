package pgdb

import (
	"fmt"
)

type Schema struct {
	Name     string
	Database *Database
	Tables   []*Table
}

func NewSchema(name string, options ...interface{}) *Schema {
	s := &Schema{
		Name:   name,
		Tables: []*Table{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *Database:
			s.Database = v
		case *Table:
			s.AddTable(v)
		}
	}
	return s
}

func (ø *Schema) AddTable(tables ...*Table) {
	for _, f := range tables {
		ø.Tables = append(ø.Tables, f)
		f.Schema = ø
	}
}

func (ø *Schema) Sql() SqlType {
	return Sql(fmt.Sprintf("\"%s\"", ø.Name))
}
