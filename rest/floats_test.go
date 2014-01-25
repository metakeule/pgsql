package rest

import (
	"fmt"
	"github.com/go-on/fat"
	// . "github.com/metakeule/pgsql"
	. "github.com/metakeule/pgsql/fat"
	"testing"
)

type FloatsTest struct {
	Id         *fat.Field `type:"string uuid" db:"id UUIDGEN PKEY" rest:" R "`
	Floats     *fat.Field `type:"[]float"     db:"floats"          rest:"CRU"`
	FloatsNull *fat.Field `type:"[]float"     db:"floatsnull NULL" rest:" RU"`
}

var FLOATS_TEST = fat.Proto(&FloatsTest{}).(*FloatsTest)
var RESTFloatsTest *REST

func init() {
	MustRegisterTable("floatstest", FLOATS_TEST)

	db.Exec("DROP TABLE floatstest")

	floatsTestTable := TableOf(FLOATS_TEST)
	_, e := db.Exec(floatsTestTable.Create().String())
	if e != nil {
		panic(fmt.Sprintf("Can't create table floatstest: \nError: %s\nSql: %s\n", e.Error(),
			floatsTestTable.Create()))
	}

	RESTFloatsTest = NewREST(FLOATS_TEST)
}

func TestFloatsCreate(t *testing.T) {
	id, err := RESTFloatsTest.Create(db, b(`
	{
		"Floats": [2.5,6]
	}
 	`))

	if err != nil {
		t.Errorf("can't create FloatsTest: %s", err)
		return
	}

	if id == "" {
		t.Errorf("got empty id")
		return
	}

	var x map[string]interface{}

	x, err = RESTFloatsTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created floatstest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Floats"]) != "[2.5,6]" {
		t.Errorf("floatstest Floats is not [2.5,6], but %#v", x["Floats"])
	}

	if x["FloatsNull"] != nil {
		t.Errorf("floatstest FloatsNull is %#v, but should be nil", x["FloatsNull"])
	}

}

func TestFloatsUpdate(t *testing.T) {
	id, _ := RESTFloatsTest.Create(db, b(`
	{
		"Floats": [2.2,2.5]
	}
	`))

	var x map[string]interface{}
	err := RESTFloatsTest.Update(db, id, b(`
	{
		"Floats": [2.5,6],
		"FloatsNull": [2.5,2.2]
	}
	`))

	if err != nil {
		t.Errorf("can't update floatstest with id %s: %s", id, err)
		return
	}

	x, err = RESTFloatsTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created floatstest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Floats"]) != "[2.5,6]" {
		t.Errorf("floatstest Floats is not [2.5,6], but %#v", x["Floats"])
	}

	if jsonify(x["FloatsNull"]) != "[2.5,2.2]" {
		t.Errorf("floatstest FloatsNull is not [2.5,2.2], but %#v", x["FloatsNull"])
	}
}
