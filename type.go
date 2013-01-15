package pgdb

import (
	"fmt"
	"strings"
)

type Type int

const (
	Int Type = iota + 256 // starting from 256 to allow VarChar to have numbers from 1 to 255
	Float
	Text
	Null
	EMailAddress
)

func VarChar(i int) Type {
	if i > 255 {
		panic("varchar may not be larger than 255")
	}
	if i < 1 {
		panic("varchar may not be smaller than 1")
	}
	t := Type(i)
	typeNames[t] = fmt.Sprintf("varchar(%v)", i)
	return t
}

func intToSql(i interface{}) (s Sql, err error) {
	in := i.(int)
	s = Sql(string(in))
	return
}

func floatToSql(i interface{}) (s Sql, err error) {
	fl := i.(float64)
	s = Sql(fmt.Sprintf("%v", fl))
	return
}

func stringToSql(i interface{}) (s Sql, err error) {
	var str string
	switch v := i.(type) {
	case Sql:
		s = v
		return
	case string:
		str = v
	case int:
		str = string(v)
	case int64:
		str = string(v)
	case float64:
		str = fmt.Sprintf("%v", v)
	case float32:
		str = fmt.Sprintf("%v", v)
	default:
		if ss, ok := v.(Stringer); ok {
			str = ss.String()
			return
		}
		err = fmt.Errorf("not convertable to a string %#v\n", i)
	}
	if err != nil {
		return
	}
	if strings.Contains(string(s), "$quote$") {
		err = fmt.Errorf("string contains $quote$, is considered a hacking attempt: %#v\n", s)
	}
	s = Sql("$quote$" + str + "$quote$")
	return
}

var EscapeFuncs = map[Type]func(interface{}) (Sql, error){
	Int:   intToSql,
	Float: floatToSql,
	Null: func(i interface{}) (s Sql, err error) {
		s = "null"
		return
	},
}

var typeNames = map[Type]string{
	Int:   "integer",
	Float: "float",
	Text:  "text",
}

func (ø Type) String() string {
	return typeNames[ø]
}

func (ø Type) Escape(i interface{}) (s Sql, err error) {
	if EscapeFuncs[ø] != nil {
		s, err = EscapeFuncs[ø](i)
	} else {
		s, err = stringToSql(i)
	}
	return
}
