package middleware

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
)

/*
The Logger middleware generates an access log entry for each request.
The format is similar to the one used by nginx. It may receive a
*log.Logger object, which by default is a Stdout logger.
*/
type Logger struct {
	AccessLogger *log.Logger
}

const dateFormat = "Jan 2, 2006 at 3:04pm (MST)"

func (lmw Logger) Handler(ph http.Handler, c context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()

		uri := r.URL.RequestURI()
		remoteAddr := webfw.RemoteAddr(r)
		remoteUser := ""
		method := r.Method
		referer := r.Header.Get("Referer")
		userAgent := r.Header.Get("User-Agent")

		ph.ServeHTTP(rec, r)

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())

		timestamp := time.Now().Format(dateFormat)
		code := rec.Code
		length := rec.Body.Len()

		lmw.AccessLogger.Print(fmt.Sprintf("%s - %s [%s] \"%s %s\" %d %d \"%s\" %s",
			remoteAddr, remoteUser, timestamp, method, uri, code, length, referer, userAgent))
	}

	return http.HandlerFunc(handler)
}
