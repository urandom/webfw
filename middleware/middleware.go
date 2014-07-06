package middleware

import (
	"log"
	"net/http"

	"github.com/urandom/webfw/types"
)

const dateFormat = "Jan 2, 2006 at 3:04pm (MST)"

/*
The middleware interface defines a method which receives a parent
http.Handler, the context object, and the error logger, which
is set to a Stderr logger by default. It has to return a regular
http.Handler.
*/
type Middleware interface {
	Handler(parentHandler http.Handler, context types.Context, logger *log.Logger) http.Handler
}
