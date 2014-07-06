package types

import (
	"net/http"
)

type Controller interface {
	Handler(Context) http.HandlerFunc
	Pattern() string

	Method() Method
	Name() string
}
