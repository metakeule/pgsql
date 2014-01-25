package rest

import (
	"fmt"
	"github.com/go-on/fat"
	// . "github.com/metakeule/pgsql"
	. "github.com/metakeule/pgsql/fat"
	"testing"
)

type IntsTest struct {
	Id       *fat.Field `type:"string uuid" db:"id UUIDGEN PKEY" rest:" R "`
	Ints     *fat.Field `type:"[]int"       db:"ints"            rest:"CRU"`
	IntsNull *fat.Field `type:"[]int"       db:"intsnull NULL"   rest:" RU"`
}

var INTS_TEST = fat.Proto(&IntsTest{}).(*IntsTest)
var RESTIntsTest *REST

func init() {
	MustRegisterTable("intstest", INTS_TEST)

	db.Exec("DROP TABLE intstest")

	intsTestTable := TableOf(INTS_TEST)
	_, e := db.Exec(intsTestTable.Create().String())
	if e != nil {
		panic(fmt.Sprintf("Can't create table intstest: \nError: %s\nSql: %s\n", e.Error(),
			intsTestTable.Create()))
	}

	RESTIntsTest = NewREST(INTS_TEST)
}

func TestIntsCreate(t *testing.T) {
	id, err := RESTIntsTest.Create(db, b(`
	{
		"Ints": [2,3]
	}
 	`))

	if err != nil {
		t.Errorf("can't create IntsTest: %s", err)
		return
	}

	if id == "" {
		t.Errorf("got empty id")
		return
	}

	var x map[string]interface{}

	x, err = RESTIntsTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created intstest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Ints"]) != "[2,3]" {
		t.Errorf("intstest Ints is not [2,3], but %#v", x["Ints"])
	}

	if x["IntsNull"] != nil {
		t.Errorf("intstest IntsNull is %#v, but should be nil", x["IntsNull"])
	}

}

func TestIntsUpdate(t *testing.T) {
	id, _ := RESTIntsTest.Create(db, b(`
	{
		"Ints": [1,2]
	}
	`))

	var x map[string]interface{}
	err := RESTIntsTest.Update(db, id, b(`
	{
		"Ints": [4,5],
		"IntsNull": [3,5]
	}
	`))

	if err != nil {
		t.Errorf("can't update intstest with id %s: %s", id, err)
		return
	}

	x, err = RESTIntsTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created intstest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Ints"]) != "[4,5]" {
		t.Errorf("intstest Ints is not [4,5], but %#v", x["Ints"])
	}

	if jsonify(x["IntsNull"]) != "[3,5]" {
		t.Errorf("intstest IntsNull is not [3,5], but %#v", x["IntsNull"])
	}
}
