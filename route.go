package webfw

import (
	"net/http"
	"webfw/types"
)

type Route struct {
	Pattern string
	Method  types.Method
	Handler http.HandlerFunc
	Name    string
}
