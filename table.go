package pgdb

import (
	"fmt"
	"strings"
)

type TableStruct struct {
	Name          string
	SchemaStruct  *SchemaStruct
	FieldStructs  []*FieldStruct
	PrimaryKeySeq Sqler
	PrimaryKey    *FieldStruct
}

func Table(name string, options ...interface{}) *TableStruct {
	t := &TableStruct{
		Name:         name,
		FieldStructs: []*FieldStruct{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *SchemaStruct:
			t.SchemaStruct = v
		case *FieldStruct:
			t.AddField(v)
		}
	}
	return t
}

func (ø *TableStruct) createField(field *FieldStruct) string {
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
		s += " REFERENCES " + string(field.ForeignKey.Sql())
		if field.Is(OnDeleteCascade) {
			s += " ON DELETE CASCADE"
		} else {
			s += " ON DELETE RESTRICT"
		}
	}
	return s
}

func (ø *TableStruct) Create() Sql {
	fs := []string{}
	for _, f := range ø.FieldStructs {
		fs = append(fs, ø.createField(f))
	}
	str := fmt.Sprintf(
		"CREATE TABLE %s \n(%s) ", ø.Sql(), strings.Join(fs, ",\n"))
	return Sql(str)
}

//func (ø *TableStruct) Alter() Sql {
//}

func (ø *TableStruct) Drop() Sql {
	return Sql("DROP " + string(ø.Sql()))
}

func (ø *TableStruct) AddField(fields ...*FieldStruct) {
	for _, f := range fields {
		ø.FieldStructs = append(ø.FieldStructs, f)
		if f.Is(PrimaryKey) {
			ø.PrimaryKey = f
		}
		f.TableStruct = ø
	}
}

func (ø *TableStruct) Field(name string) (f *FieldStruct) {
	for _, ff := range ø.FieldStructs {
		if ff.Name == name {
			return ff
		}
	}
	return
}

func (ø *TableStruct) Sql() Sql {
	if ø.SchemaStruct == nil {
		return Sql(fmt.Sprintf(`"%s"`, ø.Name))
	}
	return Sql(fmt.Sprintf(`%s."%s"`, ø.SchemaStruct.Sql(), ø.Name))
}
