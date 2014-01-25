package rest

import (
	"encoding/json"
	"fmt"
	"github.com/go-on/fat"
	. "github.com/metakeule/pgsql"
	"reflect"
	"strconv"
	"strings"
)

func (r *REST) Create(db DB, json_ []byte) (id string, err error) {
	m := map[string]interface{}{}
	err = json.Unmarshal(json_, &m)
	if err != nil {
		return
	}

	_, fields := r.queryParams("C")
	fatstruc := r.newObject()

	err = r.setMapToStruct(m, fields, fatstruc)
	if err != nil {
		fmt.Println("set map failed")
		return
	}

	row := NewRow(db, r.table)
	fs := reflect.ValueOf(fatstruc).Elem()

	for _, fld := range fields {

		if fs.FieldByName(fld).IsNil() {
			row.Set(r.field(fld), nil)
			continue
		}

		ff := fs.FieldByName(fld).Interface().(*fat.Field)

		v := ff.Get()

		switch v.(type) {
		case []fat.Type:
			vl := ff.String()
			vl = strings.Replace(vl, "[", "{", -1)
			vl = strings.Replace(vl, "]", "}", -1)
			err = row.Set(r.field(fld), vl)
		case map[string]fat.Type:
			err = row.Set(r.field(fld), ff.String())
		default:
			err = row.Set(r.field(fld), v)
		}
		if err != nil {
			fmt.Printf("err setting %s: %s\n", fld, err.Error())
			return
		}
	}
	err = row.Validate()
	if err != nil {
		fmt.Printf("error in validation: %s\n", err)
		return
	}
	err = row.Save()
	id = row.GetString(r.primaryKey)
	return
}

func (r *REST) addWhereId(query []interface{}, id string) (q []interface{}, err error) {
	var w *WhereStruct
	if r.pKeyIsString {
		w = Where(
			Equals(
				r.primaryKey,
				r.primaryKey.MustValue(id),
			),
		)
	} else {
		var i int
		i, err = strconv.Atoi(id)
		if err != nil {
			return
		}
		w = Where(
			Equals(
				r.primaryKey,
				r.primaryKey.MustValue(i),
			),
		)
	}
	q = append(query, w)
	return
}

func (r *REST) _Read(db DB, id string) (rr *Row, fields []string, err error) {
	row := NewRow(db, r.table)
	var query []interface{}
	query, fields = r.queryParams("R")
	query, err = r.addWhereId(query, id)
	if err != nil {
		fmt.Printf("addWhereId error: %s\n", err.Error())
		return
	}
	rr, err = row.Any(query...)
	return
}

func (r *REST) Read(db DB, id string) (m map[string]interface{}, err error) {
	row := NewRow(db, r.table)
	query, fields := r.queryParams("R")
	query, err = r.addWhereId(query, id)
	if err != nil {
		fmt.Printf("addWhereId error: %s\n", err.Error())
		return
	}
	var rr *Rows
	rr, err = row.Find(query...)
	if err != nil {
		fmt.Printf("error in sql find query: %s\n", err)
		return
	}
	if !rr.Next() {
		return nil, fmt.Errorf("not found: %v", id)
	}

	ro, e := rr.ScanRow()
	if e != nil {
		err = e
		fmt.Printf("error in scanrow: %s\n", err)
		return
	}
	fatstruc := r.newObject()
	m = map[string]interface{}{}
	err = r.scanToStructAndMap(ro, fields, fatstruc, m)
	if err != nil {
		fmt.Printf("error in scanmap: %s\n", err)
	}
	return
}

func (r *REST) Update(db DB, id string, json_ []byte) (err error) {
	m := map[string]interface{}{}
	err = json.Unmarshal(json_, &m)
	if err != nil {
		return
	}

	row := NewRow(db, r.table)
	query, fields := r.queryParams("U")
	query, err = r.addWhereId(query, id)
	if err != nil {
		fmt.Printf("addWhereId error: %s\n", err.Error())
		return
	}
	// add the primary key to get it back!
	query = append(query, r.primaryKey)

	var rr *Row
	rr, err = row.Any(query...)
	if err != nil {
		fmt.Printf("select error: %s\n", err.Error())
		return
	}
	fatstruc := r.newObject()
	err = r.scanToStruct(rr, fields, fatstruc)
	if err != nil {
		fmt.Printf("scan error: %s\n", err)
		return err
	}
	err = r.setMapToStruct(m, fields, fatstruc)
	if err != nil {
		fmt.Printf("set error: %s\n", err)
		return err
	}
	fs := reflect.ValueOf(fatstruc).Elem()
	for _, fld := range fields {
		if fs.FieldByName(fld).IsNil() {
			row.Set(r.field(fld), nil)
			continue
		}
		ff := fs.FieldByName(fld).Interface().(*fat.Field)
		v := ff.Get()
		var err error
		switch v.(type) {
		case []fat.Type:
			vl := ff.String()
			vl = strings.Replace(vl, "[", "{", -1)
			vl = strings.Replace(vl, "]", "}", -1)
			err = row.Set(r.field(fld), vl)
		case map[string]fat.Type:
			err = row.Set(r.field(fld), ff.String())
		default:
			err = row.Set(r.field(fld), ff.Get())
		}
		if err != nil {
			return err
		}
	}

	err = row.Validate()
	if err != nil {
		fmt.Printf("validation error: %s\n", err)
		return
	}
	return row.Save()
}

func (r *REST) Delete(db DB, id string) error {
	row := NewRow(db, r.table)
	err := row.SetId(id)
	if err != nil {
		return ValidationError(map[string]string{r.primaryKey.Name: err.Error()})
	}
	err = row.Delete()
	if err != nil {
		return ErrServer
	}
	return nil
}

func (r *REST) List(db DB, limit int, queryParams ...interface{}) (ms []map[string]interface{}, err error) {
	ms = []map[string]interface{}{}
	row := NewRow(db, r.table)
	query, fields := r.queryParams("L")
	query = append(query, Limit(limit))
	if len(queryParams) > 0 {
		query = append(query, queryParams...)
	}

	var rr *Rows
	rr, err = row.Find(query...)
	if err != nil {
		return
	}

	for rr.Next() {
		ro, e := rr.ScanRow()
		if e != nil {
			err = e
			return
		}
		fatstruc := r.newObject()
		m := map[string]interface{}{}
		err = r.scanToStructAndMap(ro, fields, fatstruc, m)
		if err != nil {
			return
		}
		ms = append(ms, m)
	}
	return
}
