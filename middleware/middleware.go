package middleware

import (
	"log"
	"net/http"
	"webfw/context"
)

const dateFormat = "Jan 2, 2006 at 3:04pm (MST)"

/*
The middleware interface defines a method which receives a parent
http.Handler, the context object, and the error logger, which
is set to a Stderr logger by default. It has to return a regular
http.Handler.
*/
type Middleware interface {
	Handler(parentHandler http.Handler, context *context.Context, logger *log.Logger) http.Handler
}
