package webfw

import (
	"net/http"

	"github.com/urandom/webfw/types"
)

type Route struct {
	Pattern string
	Method  types.Method
	Handler http.HandlerFunc
	Name    string
}
