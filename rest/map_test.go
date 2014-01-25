package rest

import (
	"fmt"
	"github.com/go-on/fat"
	// . "github.com/metakeule/pgsql"
	. "github.com/metakeule/pgsql/fat"
	"testing"
)

type MapTest struct {
	Id      *fat.Field `type:"int"            db:"id SERIAL PKEY" rest:" R "`
	Map     *fat.Field `type:"[string]string" db:"map"            rest:"CRU"`
	MapNull *fat.Field `type:"[string]int"    db:"mapnull NULL"   rest:" RU"`
}

var MAP_TEST = fat.Proto(&MapTest{}).(*MapTest)
var RESTMapTest *REST

func init() {
	MustRegisterTable("maptest", MAP_TEST)

	db.Exec("DROP TABLE maptest")

	mapTestTable := TableOf(MAP_TEST)
	_, e := db.Exec(mapTestTable.Create().String())
	if e != nil {
		panic(fmt.Sprintf("Can't create table maptest: \nError: %s\nSql: %s\n", e.Error(),
			mapTestTable.Create()))
	}

	RESTMapTest = NewREST(MAP_TEST)
}

func TestMapCreate(t *testing.T) {
	id, err := RESTMapTest.Create(db, b(`
	{
		"Map": {"a":"b"}
	}
 	`))

	if err != nil {
		t.Errorf("can't create MapTest: %s", err)
		return
	}

	if id == "" {
		t.Errorf("got empty id")
		return
	}

	var x map[string]interface{}

	x, err = RESTMapTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created maptest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Map"]) != `{"a":"b"}` {
		t.Errorf("maptest Map is not {\"a\":\"b\"}, but %#v", x["Map"])
	}

	if x["MapNull"] != nil {
		t.Errorf("maptest MapNull is %#v, but should be nil", x["MapNull"])
	}

}

func TestMapUpdate(t *testing.T) {
	id, _ := RESTMapTest.Create(db, b(`
	{
		"Map": {"b":"c"}
	}
	`))

	var x map[string]interface{}
	err := RESTMapTest.Update(db, id, b(`
	{
		"Map": {"d":"e"},
		"MapNull": {"f":5}
	}
	`))

	if err != nil {
		t.Errorf("can't update maptest with id %s: %s", id, err)
		return
	}

	x, err = RESTMapTest.Read(db, id)

	if err != nil {
		t.Errorf("can't get created maptest with id %s: %s", id, err)
		return
	}

	if jsonify(x["Map"]) != `{"d":"e"}` {
		t.Errorf("maptest Map is not {\"d\":\"e\"}, but %#v", x["Map"])
	}

	if jsonify(x["MapNull"]) != `{"f":5}` {
		t.Errorf("maptest MapNull is not {\"f\":5}, but %#v", x["MapNull"])
	}
}
