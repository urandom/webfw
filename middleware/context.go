package middleware

import (
	"log"
	"net/http"

	"github.com/urandom/webfw/context"
)

// The Context middleware cleans up the framework context object of any data
// related to the current request, after it has gone through the middleware
// chain.
type Context struct{}

func (cmw Context) Handler(ph http.Handler, c context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			c.DeleteAll(r)
		}()

		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
