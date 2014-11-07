package webfw

import (
	"net/http"

	"github.com/urandom/webfw/context"
)

/*
The middleware interface defines a method which receives a parent
http.Handler and the context object. It has to return a regular
http.Handler.
*/
type Middleware interface {
	Handler(http.Handler, context.Context) http.Handler
}
