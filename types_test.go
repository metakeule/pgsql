package pgsql

import (
	"fmt"
	"github.com/metakeule/typeconverter"
	"testing"
	"time"
)

func err(t *testing.T, msg string, is interface{}, shouldbe interface{}) {
	t.Errorf(msg+": is %#v, should be %#v\n", is, shouldbe)
}

var _ = fmt.Errorf
var ti, _ = time.Parse(time.RFC3339, "2011-01-26T18:53:18+01:00")
var timeString = ti.Format(time.RFC3339)
var timeUnix = ti.Unix()
var timeFloat = float64(1010000000)
var tiFloat = time.Unix(1010000000, 0)
var tiFloatString = tiFloat.Format(time.RFC3339)

var toTypedValueTests = map[interface{}]Type{
	1:            IntType,
	int32(2):     IntType,
	int64(2):     IntType,
	float64(3.0): FloatType,
	float32(3.0): FloatType,
	`3.0`:        TextType,
	true:         BoolType,
	ti:           TimeStampTZType,
	typeconverter.Json(`{"a": 4}`): TextType,
}

func TestToTypedValue(t *testing.T) {
	for in, out := range toTypedValueTests {
		r := TypedValue{}
		Convert(in, &r)
		if r.Type() != out {
			err(t, "Convert to typed value, wrong type", r.Type().String(), out.String())
		}

		inStr := ""
		typeconverter.Convert(in, &inStr)

		if r.Value.String() != inStr {
			err(t, "Convert to typed value changed input", r.Value.String(), inStr)
		}
	}
}

func TestToInt(t *testing.T) {
	var r int

	out := int(1)

	if Convert(out, &r); r != out {
		err(t, "Convert int", r, out)
	}

	if Convert(int32(1), &r); r != out {
		err(t, "Convert int", r, out)
	}

	if Convert(int64(1), &r); r != out {
		err(t, "Convert int", r, out)
	}

	if Convert(float64(1.0), &r); r != out {
		err(t, "Convert int", r, out)
	}

	if Convert(float32(1.0), &r); r != out {
		err(t, "Convert int", r, out)
	}

	if Convert("1", &r); r != out {
		err(t, "Convert int", r, out)
	}

	var tv TypedValue
	Convert(1, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert int", r, out)
	}

	Convert(1.0, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert int", r, out)
	}

	Convert("1", &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert int", r, out)
	}

	var i64 int64
	if Convert(int64(1), &i64); i64 != int64(1) {
		err(t, "Convert int", i64, int64(1))
	}

}

func TestToFloat(t *testing.T) {
	var r float64

	out := float64(1.0)

	if Convert(out, &r); r != out {
		err(t, "Convert Float", r, out)
	}

	if Convert(int32(1), &r); r != out {
		err(t, "Convert Float", r, out)
	}

	if Convert(int64(1), &r); r != out {
		err(t, "Convert Float", r, out)
	}

	if Convert(float64(1.0), &r); r != out {
		err(t, "Convert Float", r, out)
	}

	if Convert(float32(1.0), &r); r != out {
		err(t, "Convert Float", r, out)
	}

	if Convert("1", &r); r != out {
		err(t, "Convert Float", r, out)
	}

	var tv TypedValue
	Convert(1, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Float", r, out)
	}

	Convert(1.0, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Float", r, out)
	}

	Convert("1.0", &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Float", r, out)
	}

	var f32 float32
	if Convert(float32(1), &f32); f32 != float32(1) {
		err(t, "Convert Float", f32, float32(1))
	}
}

func TestToBool(t *testing.T) {
	var r bool

	out := true

	if Convert(out, &r); r != out {
		err(t, "Convert Bool", r, out)
	}

	if Convert("true", &r); r != out {
		err(t, "Convert Bool", r, out)
	}

	if Convert(typeconverter.Json("true"), &r); r != out {
		err(t, "Convert Bool", r, out)
	}

	var tv TypedValue
	Convert(true, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Bool", r, out)
	}

	Convert("true", &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Bool", r, out)
	}

	Convert(typeconverter.Json("true"), &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert Bool", r, out)
	}
}

func TestToString(t *testing.T) {
	var r string

	out := "1"

	if Convert(out, &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert(int32(1), &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert(int64(1), &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert(float64(1.0), &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert(float32(1.0), &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert("1", &r); r != out {
		err(t, "Convert String", r, out)
	}

	if Convert(typeconverter.Json(`1`), &r); r != out {
		err(t, "Convert String", r, out)
	}

	sql := Sql("Select 1")

	if Convert(sql, &r); r != sql.Sql().String() {
		err(t, "Convert String", r, sql.Sql())
	}

	var tv TypedValue
	Convert(1, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert String", r, out)
	}

	Convert(1.0, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert String", r, out)
	}

	Convert("1", &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert String", r, out)
	}

	Convert(typeconverter.Json(`1`), &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert String", r, out)
	}

	tv = TypedValue{}

	Convert(sql, &tv)

	if Convert(&tv, &r); r != sql.Sql().String() {
		err(t, "Convert String", r, sql.Sql())
	}
}

func TestToTime(t *testing.T) {
	var r time.Time

	out := ti

	if Convert(out, &r); r != out {
		err(t, "Convert time", r, out)
	}

	if Convert(int32(timeUnix), &r); r != out {
		err(t, "Convert time", r, out)
	}
	if Convert(int64(timeUnix), &r); r != out {
		err(t, "Convert time", r, out)
	}

	/*
		Does not work: check!
		if Convert(timeFloat, &r); r.Format(time.RFC3339) != tiFloatString {
			err(t, "Convert time", r.Format(time.RFC3339), tiFloatString)
		}
	*/

	if Convert(timeString, &r); r != out {
		err(t, "Convert time", r, out)
	}

	if Convert(typeconverter.Json(timeString), &r); r != out {
		err(t, "Convert time", r, out)
	}

	var tv TypedValue
	Convert(timeUnix, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert time", r, out)
	}

	Convert(int32(timeUnix), &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert time", r, out)
	}

	Convert(timeString, &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert time", r, out)
	}

	Convert(typeconverter.Json(timeString), &tv)

	if Convert(&tv, &r); r != out {
		err(t, "Convert time", r, out)
	}
}

func TestToType(t *testing.T) {
	for in, out := range toTypedValueTests {
		r := Type(0)
		Convert(in, &r)
		if r.String() != out.String() {
			err(t, "Convert to type wrong type", r.String(), out.String())

		}
	}
}

type MyInt int

func (ø MyInt) Sql() SqlType { return Sql(fmt.Sprintf("select %v", ø)) }

var toSqlTests = map[interface{}]string{
	1:            `'1'::int`,
	int32(2):     `'2'::int`,
	int64(2):     `'2'::int`,
	float64(3.5): `'3.5'::float`,
	float32(3.5): `'3.5'::float`,
	`3.0`:        `'3.0'::text`,
	true:         `'true'::bool`,
	ti:           `'` + timeString + `'::timestamptz`,
	typeconverter.Json(`{"a":4}`): `'{"a":4}'::text`,
	Sql("select * from person"):   `select * from person`,
	MyInt(4):                      `select 4`,
}

func TestToSql(t *testing.T) {
	for in, out := range toSqlTests {
		r := Sql("")
		Convert(in, &r)
		if r.Sql().String() != out {
			err(t, "Convert to Sql", r, out)
		}
	}
}
