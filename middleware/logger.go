package middleware

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
	"webfw/context"

	"strings"
)

/*
The Logger middleware generates an access log entry for each request.
The format is similar to the one used by nginx. It may receive a
*log.Logger object, which by default is a Stdout logger.
*/
type Logger struct {
	AccessLogger *log.Logger
}

func (lmw Logger) Handler(ph http.Handler, c *context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()

		uri := r.URL.RequestURI()
		remoteAddr := remoteAddr(r)
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

func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func remoteAddr(r *http.Request) string {
	hdr := r.Header
	hdrRealIp := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIp == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts[0]
	}
	return hdrRealIp
}
