package pgdb

import (
	"fmt"
)

type Stringer interface {
	String() string
}

//type Sql string
/*
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
*/

type Database struct {
	Name    string
	Schemas []*Schema
}

func ToString(i interface{}) string {
	return fmt.Sprintf("%v", i)
}
