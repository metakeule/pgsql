package rest

import (
	"fmt"
	"github.com/go-on/fat"
	// . "github.com/metakeule/pgsql"
	. "github.com/metakeule/pgsql/fat"
	"testing"
)

type FloatTest struct {
	Id        *fat.Field `type:"string uuid" db:"id UUIDGEN PKEY" rest:" R "`
	Float     *fat.Field `type:"float"       db:"float"           rest:"CRU"`
	FloatNull *fat.Field `type:"float"       db:"floatnull NULL"  rest:" RU" `
}

var FLOAT_TEST = fat.Proto(&FloatTest{}).(*FloatTest)
var RESTFloatTest *REST

func init() {
	MustRegisterTable("floattest", FLOAT_TEST)

	db.Exec("DROP TABLE floattest")

	floatTestTable := TableOf(FLOAT_TEST)
	_, e := db.Exec(floatTestTable.Create().String())
	if e != nil {
		panic(fmt.Sprintf("Can't create table floattest: \nError: %s\nSql: %s\n", e.Error(),
			floatTestTable.Create()))
	}

	RESTFloatTest = NewREST(FLOAT_TEST)
}

func TestFloatCreate(t *testing.T) {
	id, err := RESTFloatTest.Create(db, b(`
	{
		"Float": 2.5
	}
 	`))

	if err != nil {
		t.Errorf("can't create FloatTest: %s", err)
		return
	}

	if id == "" {
		t.Errorf("got empty id")
		return
	}

	var x map[string]interface{}

	x, err = RESTFloatTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created floattest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Float"]) != "2.5" {
		t.Errorf("floattest Float is not 2.5, but %#v", x["Float"])
	}

	if x["FloatNull"] != nil {
		t.Errorf("floattest FloatNull is %#v, but should be nil", x["FloatNull"])
	}

}

func TestFloatUpdate(t *testing.T) {
	id, _ := RESTFloatTest.Create(db, b(`
	{
		"Float": 2.2
	}
	`))

	var x map[string]interface{}
	err := RESTFloatTest.Update(db, id, b(`
	{
		"Float": 6,
		"FloatNull": 2.5
	}
	`))

	if err != nil {
		t.Errorf("can't update floattest with id %s: %s", id, err)
		return
	}

	x, err = RESTFloatTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created floattest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Float"]) != "6" {
		t.Errorf("floattest Float is not 6, but %#v", x["Float"])
	}

	if jsonify(x["FloatNull"]) != "2.5" {
		t.Errorf("floattest FloatNull is not 2.5, but %#v", x["FloatNull"])
	}
}
