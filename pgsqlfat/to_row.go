package pgsqlfat

import (
	"fmt"
	"reflect"
	"strings"
	"github.com/go-on/meta"

	"github.com/go-on/fat"
	. "github.com/metakeule/pgsql"
)

/*
	only the fields that are *fat.Field and not nil are chosen to set
	the row. øptrToFatStruct must be registered with RegisterTable
	before using this function
*/
func ToRow(øptrToFatStruct interface{}, row *Row) (err error) {
	var stru *meta.Struct
	stru, err = meta.StructByValue(reflect.ValueOf(øptrToFatStruct))

	if err != nil {
		return
	}

	t := TableOf(øptrToFatStruct)

	if t == nil {
		err = fmt.Errorf("%T is not registered, use RegisterTable", øptrToFatStruct)
		return
	}

	if row.Table != t {
		err = fmt.Errorf("table of the given fatstruct (%s) is not the same as table of the given row (%s)",
			t.Sql().String(),
			row.Table.Sql().String(),
		)
	}

	if err != nil {
		return
	}

	fn := func(field *meta.Field) {
		// stop on first error
		if err != nil {
			return
		}

		if field.Value.IsNil() {
			return
		}

		ff, isFat := field.Value.Interface().(*fat.Field)

		if !isFat {
			return
		}

		rowField := FieldOf(ff)
		v := ff.Get()

		switch v.(type) {
		case []fat.Type:
			vl := ff.String()
			vl = strings.Replace(vl, "[", "{", -1)
			vl = strings.Replace(vl, "]", "}", -1)
			err = row.Set(rowField, vl)
		case map[string]fat.Type:
			err = row.Set(rowField, ff.String())
		default:
			err = row.Set(rowField, v)
		}
	}

	stru.Each(fn)
	return
}
