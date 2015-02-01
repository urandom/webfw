package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"
)

/*
The Logger middleware generates an access log entry for each request.
The format is similar to the one used by nginx. It may receive a
webfw.Logger object, which by default is a Stdout logger.
*/
type Logger struct {
	AccessLogger webfw.Logger
}

const dateFormat = "Jan 2, 2006 at 3:04pm (MST)"

func (lmw Logger) Handler(ph http.Handler, c context.Context) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := util.NewRecorderHijacker(w)

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
		w.WriteHeader(rec.GetCode())
		w.Write(rec.GetBody().Bytes())

		timestamp := time.Now().Format(dateFormat)
		code := rec.GetCode()
		length := rec.GetBody().Len()

		lmw.AccessLogger.Print(fmt.Sprintf("%s - %s [%s] \"%s %s\" %d %d \"%s\" %s",
			remoteAddr, remoteUser, timestamp, method, uri, code, length, referer, userAgent))
	}

	return http.HandlerFunc(handler)
}
