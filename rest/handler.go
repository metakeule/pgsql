package rest

import (
	"encoding/json"
	"fmt"
	"github.com/go-on/rack/router"
	"github.com/go-on/rack/wrapper"
	. "github.com/metakeule/pgsql"
	"io/ioutil"
	"net/http"
	"path"
)

func (r *REST) HandleCreate(db DB, rt *router.Router, mountPoint string) *router.Route {
	fn := func(vr *router.Vars, wr http.ResponseWriter, rq *http.Request) {
		json_, err := ioutil.ReadAll(rq.Body)
		var id string
		if err == nil {
			id, err = r.Create(db, json_)
		}

		setJsonContentType(wr)
		if err != nil {
			resp := jsonResp{}
			resp.Success = false
			resp.Error = err.Error()
			j, e := json.MarshalIndent(resp, "", "  ")
			if e != nil {
				panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
			}
			wr.WriteHeader(500)
			wr.Write(j)
			return
		}

		resp := map[string]string{"created": id}
		j, e := json.MarshalIndent(resp, "", "  ")
		if e != nil {
			panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
		}
		wr.Write(j)
	}
	return rt.POST("/"+mountPoint, wrapper.HandlerMethod(fn))
}

func (r *REST) HandleList(db DB, rt *router.Router, mountPoint string, limit int) *router.Route {
	fn := func(vr *router.Vars, wr http.ResponseWriter, rq *http.Request) {
		objs, err := r.List(db, limit)
		setJsonContentType(wr)
		if err != nil {
			resp := jsonResp{}
			resp.Success = false
			resp.Error = err.Error()
			j, e := json.MarshalIndent(resp, "", "  ")
			if e != nil {
				panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
			}
			wr.WriteHeader(500)
			wr.Write(j)
			return
		}
		j, e := json.MarshalIndent(objs, "", "  ")
		if e != nil {
			panic(fmt.Sprintf("can't json serialize %#v: %s", objs, e.Error()))
		}
		wr.Write(j)
	}
	return rt.GET("/"+mountPoint, wrapper.HandlerMethod(fn))
}

func (r *REST) HandleUpdate(db DB, rt *router.Router, mountPoint string) *router.Route {
	ph := mountPoint + "_id"
	fn := func(vr *router.Vars, wr http.ResponseWriter, rq *http.Request) {
		id := vr.Get(ph)
		json_, err := ioutil.ReadAll(rq.Body)
		if err == nil {
			err = r.Update(db, id, json_)
		}
		setJsonContentType(wr)
		if err != nil {
			resp := jsonResp{}
			resp.Success = false
			resp.Error = err.Error()
			j, e := json.MarshalIndent(resp, "", "  ")
			if e != nil {
				panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
			}
			wr.WriteHeader(500)
			wr.Write(j)
			return
		}
		resp := jsonResp{}
		resp.Success = true
		j, e := json.MarshalIndent(resp, "", "  ")
		if e != nil {
			panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
		}
		wr.Write(j)
	}
	return rt.PUT("/"+path.Join(mountPoint, ":"+ph), wrapper.HandlerMethod(fn))
}

func (r *REST) HandleRead(db DB, rt *router.Router, mountPoint string) *router.Route {
	ph := mountPoint + "_id"
	fn := func(vr *router.Vars, wr http.ResponseWriter, rq *http.Request) {
		id := vr.Get(ph)
		obj, err := r.Read(db, id)
		setJsonContentType(wr)
		if obj == nil {
			ErrNotFound.ServeHTTP(wr, rq)
			return
		}
		if err != nil {
			resp := jsonResp{}
			resp.Success = false
			resp.Error = err.Error()
			j, e := json.MarshalIndent(resp, "", "  ")
			if e != nil {
				panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
			}
			wr.WriteHeader(500)
			wr.Write(j)
			return
		}
		j, e := json.MarshalIndent(obj, "", "  ")
		if e != nil {
			panic(fmt.Sprintf("can't json serialize %#v: %s", obj, e.Error()))
		}
		wr.Write(j)
	}
	return rt.GET("/"+path.Join(mountPoint, ":"+ph), wrapper.HandlerMethod(fn))
}

func (r *REST) HandleDelete(db DB, rt *router.Router, mountPoint string) *router.Route {
	ph := mountPoint + "_id"
	fn := func(vr *router.Vars, wr http.ResponseWriter, rq *http.Request) {
		id := vr.Get(ph)
		err := r.Delete(db, id)
		setJsonContentType(wr)
		if err != nil {
			resp := jsonResp{}
			resp.Success = false
			resp.Error = err.Error()
			j, e := json.MarshalIndent(resp, "", "  ")
			if e != nil {
				panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
			}
			wr.WriteHeader(500)
			wr.Write(j)
			return
		}
		resp := jsonResp{}
		resp.Success = true
		j, e := json.MarshalIndent(resp, "", "  ")
		if e != nil {
			panic(fmt.Sprintf("can't json serialize %#v: %s", resp, e.Error()))
		}
		wr.Write(j)
	}
	return rt.DELETE("/"+path.Join(mountPoint, ":"+ph), wrapper.HandlerMethod(fn))
}
