package pgdb

import (
	"fmt"
)

type SchemaStruct struct {
	Name         string
	Database     *Database
	TableStructs []*TableStruct
}

func Schema(name string, options ...interface{}) *SchemaStruct {
	s := &SchemaStruct{
		Name:         name,
		TableStructs: []*TableStruct{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *Database:
			s.Database = v
		case *TableStruct:
			s.AddTable(v)
		}
	}
	return s
}

func (ø *SchemaStruct) AddTable(tables ...*TableStruct) {
	for _, f := range tables {
		ø.TableStructs = append(ø.TableStructs, f)
		f.SchemaStruct = ø
	}
}

func (ø *SchemaStruct) Sql() Sql {
	return Sql(fmt.Sprintf("\"%s\"", ø.Name))
}
