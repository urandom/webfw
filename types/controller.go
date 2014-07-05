package types

import (
	"net/http"
	"webfw/context"
)

type Controller interface {
	Handler(*context.Context) http.HandlerFunc
	Pattern() string

	Method() Method
	Name() string
}
