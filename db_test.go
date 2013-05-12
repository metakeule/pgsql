package pgsql

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"os"
	"testing"
)

var db *sql.DB

func Connect(str string) {
	cs, ſ := pq.ParseURL(str)
	if ſ != nil {
		panic(ſ)
	}

	db, ſ = sql.Open("postgres", cs)
	if ſ != nil {
		panic(ſ)
	}
}

func init() {
	connStr := os.Getenv("PG_TEST")
	if connStr != "" {
		Connect(connStr)
	} else {
		fmt.Println("PG_TEST not set, skipping db tests")
	}
}

func TestSimpleTable(t *testing.T) {
	if db == nil {
		return
	}
	table := NewTable("person")
	id := table.NewField("id", PrimaryKey, IntType, Serial)
	firstname := table.NewField("firstname", VarChar(255), NullAllowed)
	lastname := table.NewField("lastname", VarChar(255))

	_, _ = db.Exec(table.Drop().Sql().String())

	_, ſ := db.Exec(table.Create().Sql().String())

	if ſ != nil {
		t.Errorf("Can't create table %v:\nSql:\n%v\n", ſ.Error(), table.Create().Sql())
		return
	}

	row := NewRow(db, table)
	row.Set(firstname, "Donald", lastname, "Duck")
	ſ = row.Save()

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	var i int
	row.Get(id, &i)

	if i != 1 {
		t.Errorf("wrong id: %v\n", i)
	}

	row = NewRow(db, table)
	ſ = row.Load(fmt.Sprintf("%v", i))

	if ſ != nil {
		t.Errorf("Can't load: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v := row.AsStrings()

	if v["firstname"] != "Donald" {
		t.Errorf("Wrong firstname: %v\n", ſ.Error())
	}

	if v["lastname"] != "Duck" {
		t.Errorf("Wrong lastname: %v\n", ſ.Error())
	}

	row.Set(lastname, "Mouse", firstname, "Mickey")

	ſ = row.Save()

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = row.AsStrings()

	if v["firstname"] != "Mickey" {
		t.Errorf("Wrong changed firstname: %v\n", ſ.Error())
	}

	if v["lastname"] != "Mouse" {
		t.Errorf("Wrong changed lastname: %v\n", ſ.Error())
	}

	r, ſ := row.Any(Where(Equals(lastname, "Mouse")))

	if ſ != nil {
		t.Errorf("Can't find based on lastname: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = r.AsStrings()

	if v["firstname"] != "Mickey" {
		t.Errorf("Wrong found firstname: %v\n", ſ.Error())
	}
}

func TestMultiplePkeys(t *testing.T) {
	if db == nil {
		return
	}
	table := NewTable("person")
	//firstname := table.NewField("firstname", VarChar(255), PrimaryKey)
	lastname := table.NewField("lastname", VarChar(255), PrimaryKey)
	age := table.NewField("age", IntType, PrimaryKey)

	_, _ = db.Exec(table.Drop().Sql().String())

	_, ſ := db.Exec(table.Create().Sql().String())

	if ſ != nil {
		t.Errorf("Can't create table %v:\nSql:\n%v\n", ſ.Error(), table.Create().Sql())
		return
	}

	row := NewRow(db, table)
	row.Set(age, 340, lastname, "Duck")
	ſ = row.Insert()

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v := row.AsStrings()

	if v["age"] != "340" {
		t.Errorf("Wrong age: %v\n", ſ.Error())
	}

	if v["lastname"] != "Duck" {
		t.Errorf("Wrong lastname: %v\n", ſ.Error())
	}
	row.Set(lastname, "Mouse", age, 1200)
	row.Debug = true

	tv := &TypedValue{PgType: age.Type, Value: NewPgInterpretedString("340")}
	ſ = row.Update("Duck", tv)

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = row.AsStrings()

	if v["age"] != "1200" {
		t.Errorf("Wrong changed age: %v\n", ſ.Error())
	}

	if v["lastname"] != "Mouse" {
		t.Errorf("Wrong changed lastname: %v\n", ſ.Error())
	}

	r, ſ := row.Any(Where(Equals(lastname, "Mouse")))

	if ſ != nil {
		t.Errorf("Can't find based on lastname: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = r.AsStrings()

	if v["age"] != "1200" {
		t.Errorf("Wrong found age: %v\n", ſ.Error())
	}
}

func TestUuidTable(t *testing.T) {
	if db == nil {
		return
	}
	// _, _ = db.Exec(`CREATE EXTENSION "uuid-ossp";`)
	table := NewTable("person")
	id := table.NewField("id", PrimaryKey, UuidType, UuidGenerate)
	//table.NewField("id", PrimaryKey, UuidType, UuidGenerate)
	firstname := table.NewField("firstname", VarChar(255), NullAllowed)
	lastname := table.NewField("lastname", VarChar(255))

	_, _ = db.Exec(table.Drop().Sql().String())

	_, ſ := db.Exec(table.Create().Sql().String())

	if ſ != nil {
		t.Errorf("Can't create table %v:\nSql:\n%v\n", ſ.Error(), table.Create().Sql())
		return
	}

	row := NewRow(db, table)
	row.Set(firstname, "Donald", lastname, "Duck")
	// row.Debug = true
	ſ = row.Save()

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	i := row.GetString(id)
	//fmt.Printf("%v\n", i)

	row = NewRow(db, table)
	//row.Debug = true
	ſ = row.Load(i)

	if ſ != nil {
		t.Errorf("Can't load: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v := row.AsStrings()

	if v["firstname"] != "Donald" {
		t.Errorf("Wrong firstname: %v\n", ſ.Error())
	}

	if v["lastname"] != "Duck" {
		t.Errorf("Wrong lastname: %v\n", ſ.Error())
	}

	row.Set(lastname, "Mouse", firstname, "Mickey")

	ſ = row.Save()

	if ſ != nil {
		t.Errorf("Can't save: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = row.AsStrings()

	if v["firstname"] != "Mickey" {
		t.Errorf("Wrong changed firstname: %v\n", ſ.Error())
	}

	if v["lastname"] != "Mouse" {
		t.Errorf("Wrong changed lastname: %v\n", ſ.Error())
	}

	r, ſ := row.Any(Where(Equals(lastname, "Mouse")))

	if ſ != nil {
		t.Errorf("Can't find based on lastname: %v \nSql:\n%v\n", ſ.Error(), row.LastSql)
	}

	v = r.AsStrings()

	if v["firstname"] != "Mickey" {
		t.Errorf("Wrong found firstname: %v\n", ſ.Error())
	}

}
