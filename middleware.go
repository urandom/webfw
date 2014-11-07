package webfw

import (
	"net/http"

	"github.com/urandom/webfw/context"
)

/*
The middleware interface defines a method which receives a parent
http.Handler, the context object, and the error logger, which
is set to a Stderr logger by default. It has to return a regular
http.Handler.
*/
type Middleware interface {
	Handler(http.Handler, context.Context, Logger) http.Handler
}
