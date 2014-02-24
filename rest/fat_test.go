package rest

import (
	"testing"

	"github.com/go-on/fat"
)

// . "github.com/metakeule/pgsql"

/*
ty := reflect.TypeOf(østruct).Elem()
	return "*" + ty.PkgPath() + "." + ty.Name()
*/

type coop struct {
	Id        *fat.Field `type:"string uuid"        db:"id UUIDGEN PKEY" rest:" R DL"`
	Name      *fat.Field `type:"string varchar(66)" db:"name"            rest:"CRU L"`
	FoundedAt *fat.Field `type:"time date"          db:"founded_at NULL" rest:"CRU L"`
}

var COOP = fat.Proto(&coop{}).(*coop)

func init() {
	MustRegisterTable("co_op", COOP)
}

func TestRegistry(t *testing.T) {
	f := FieldOf(COOP.Name)

	if f == nil {
		t.Error("field should not be nil")
	}

	ta := TableOf(COOP)

	if ta == nil {
		t.Error("table should not be nil")
	}
}
