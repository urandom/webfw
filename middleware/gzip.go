package middleware

import (
	"compress/gzip"
	"log"
	"net/http"
	"net/http/httptest"
	"webfw/context"
	"webfw/util"

	"strconv"
	"strings"
)

// The Gzip middleware will compress the response using the gzip format.
// If placed in the middleware chain, it will be triggered whenever the
// client states it may accept gzip via the Accept-Encoding header.
type Gzip struct{}

func (gmw Gzip) Handler(ph http.Handler, c *context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		useGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		ph.ServeHTTP(rec, r)

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Header().Set("Vary", "Accept-Encoding")

		if useGzip {
			w.Header().Set("Content-Encoding", "gzip")

			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", http.DetectContentType(rec.Body.Bytes()))
			}
		}

		if useGzip {
			buf := util.BufferPool.GetBuffer()
			defer util.BufferPool.Put(buf)

			gz := gzip.NewWriter(buf)

			if _, err := gz.Write(rec.Body.Bytes()); err != nil {
				panic(err)
			}
			gz.Close()

			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

			w.WriteHeader(rec.Code)

			buf.WriteTo(w)
		} else {
			w.WriteHeader(rec.Code)

			w.Write(rec.Body.Bytes())
		}
	}

	return http.HandlerFunc(handler)
}
