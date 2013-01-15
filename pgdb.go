package pgdb

import ()

type Stringer interface {
	String() string
}

type Sql string

func (ø Sql) String() string {
	return string(ø)
}

func String(s string) Sql {
	sql, _ := stringToSql(s)
	return sql
}

func (ø Sql) Sql() Sql {
	return ø
}

type Sqler interface {
	Sql() Sql
}

type Database struct {
	Name          string
	SchemaStructs []*SchemaStruct
}
