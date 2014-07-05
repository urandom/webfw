package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"webfw/context"
)

// The Error middleware provides basic panic recovery for a request. For
// this reason, it should be at the botton of the middleware chain, to
// catch any raised panics along the way. If such occurs, the response
// writer will contain the 'Internal Server Error' message, and the
// stack trace will be written to the error log. It also has a ShowStack
// option, which will cause the stack trace to be written to the response
// writer if true. It is set to true if the global configuration is set to
// "devel".
type Error struct {
	ShowStack bool
}

func (emw Error) Handler(ph http.Handler, c *context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				message := fmt.Sprintf("%s\n%s\n", rec, stack)

				l.Print(message)
				w.WriteHeader(http.StatusInternalServerError)

				if !emw.ShowStack {
					message = "Internal Server Error"
				}
				w.Write([]byte(message))
				c.DeleteAll(r)
			}
		}()

		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
