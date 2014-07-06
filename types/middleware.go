package types

import (
	"log"
	"net/http"
)

/*
The middleware interface defines a method which receives a parent
http.Handler, the context object, and the error logger, which
is set to a Stderr logger by default. It has to return a regular
http.Handler.
*/
type Middleware interface {
	Handler(parentHandler http.Handler, context Context, logger *log.Logger) http.Handler
}
