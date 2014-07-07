package webfw

import "net/http"

type Route struct {
	Pattern string
	Method  Method
	Handler http.HandlerFunc
	Name    string
}

type Method int
type RouteParams map[string]string

const (
	MethodGet Method = 1 << iota
	MethodPost
	MethodPut
	MethodDelete
	MethodPatch
	MethodAll Method = MethodGet | MethodPost | MethodPut | MethodDelete | MethodPatch
)

var MethodNames map[Method]string = map[Method]string{
	MethodGet:    "GET",
	MethodPost:   "POST",
	MethodPut:    "PUT",
	MethodDelete: "DELETE",
	MethodPatch:  "PATCH",
}

var ReverseMethodNames = map[string]Method{
	"GET":    MethodGet,
	"POST":   MethodPost,
	"PUT":    MethodPut,
	"DELETE": MethodDelete,
	"PATCH":  MethodPatch,
}
