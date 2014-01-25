package rest

import (
	"fmt"
	"github.com/go-on/fat"
	// . "github.com/metakeule/pgsql"
	. "github.com/metakeule/pgsql/fat"
	"testing"
)

type BoolTest struct {
	Id       *fat.Field `type:"string uuid" db:"id UUIDGEN PKEY" rest:" R "`
	Bool     *fat.Field `type:"bool"        db:"bool"            rest:"CRU"`
	BoolNull *fat.Field `type:"bool"        db:"boolnull NULL"   rest:" RU"`
}

var BOOL_TEST = fat.Proto(&BoolTest{}).(*BoolTest)
var RESTBoolTest *REST

func init() {
	MustRegisterTable("booltest", BOOL_TEST)

	db.Exec("DROP TABLE booltest")

	boolTestTable := TableOf(BOOL_TEST)
	_, e := db.Exec(boolTestTable.Create().String())
	if e != nil {
		panic(fmt.Sprintf("Can't create table booltest: \nError: %s\nSql: %s\n", e.Error(),
			boolTestTable.Create()))
	}

	RESTBoolTest = NewREST(BOOL_TEST)
}

func TestBoolCreate(t *testing.T) {
	id, err := RESTBoolTest.Create(db, b(`
	{
		"Bool": true
	}
 	`))

	if err != nil {
		t.Errorf("can't create BoolTest: %s", err)
		return
	}

	if id == "" {
		t.Errorf("got empty id")
		return
	}

	var x map[string]interface{}

	x, err = RESTBoolTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created booltest with id %s: %s", id, err)
		return
	}

	if x["Bool"].(bool) != true {
		t.Errorf("booltest Bool is not true, but %#v", x["Bool"])
	}

	if x["BoolNull"] != nil {
		t.Errorf("booltest BoolNull is %#v, but should be nil", x["BoolNull"])
	}

}

func TestBoolUpdate(t *testing.T) {
	id, _ := RESTBoolTest.Create(db, b(`
	{
		"Bool": true
	}
	`))

	var x map[string]interface{}
	err := RESTBoolTest.Update(db, id, b(`
	{
		"Bool": false,
		"BoolNull": true
	}
	`))

	if err != nil {
		t.Errorf("can't update booltest with id %s: %s", id, err)
		return
	}

	x, err = RESTBoolTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created booltest with id %s: %s", id, err)
		return
	}

	if x["Bool"].(bool) != false {
		t.Errorf("booltest Bool is not false, but %#v", x["Bool"])
	}

	if x["BoolNull"].(bool) != true {
		t.Errorf("booltest BoolNull is not true, but %#v", x["BoolNull"].(bool))
	}
}
