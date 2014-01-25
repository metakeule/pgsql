package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ValidationError map[string]string

func (e ValidationError) Error() string {
	return "validation error"
}

func (e ValidationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error            string
		ValidationErrors map[string]string
	}{"validation error", e})
}

func (e ValidationError) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	setJsonContentType(wr)
	b, _ := json.MarshalIndent(e, "", "  ")
	wr.WriteHeader(422)
	fmt.Fprintf(wr, "%s", b)
}

type _err struct {
	err    string
	status int
}

func (e _err) Error() string {
	return e.err
}

func (e _err) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{ Error string }{e.err})
}

func (e _err) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	setJsonContentType(wr)
	b, _ := json.MarshalIndent(e, "", "  ")
	wr.WriteHeader(e.status)
	fmt.Fprintf(wr, "%s", b)
}

var ErrNotFound = _err{"Not found", http.StatusNotFound}
var ErrServer = _err{"Internal Server Error", http.StatusInternalServerError}
var ErrConflict = _err{"Conflict", http.StatusConflict}

type jsonResp struct {
	Success    bool
	Error      string
	Validation map[string]string
}
