package middleware

import (
	"compress/gzip"
	"net/http"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"

	"strconv"
	"strings"
)

// The Gzip middleware will compress the response using the gzip format.
// If placed in the middleware chain, it will be triggered whenever the
// client states it may accept gzip via the Accept-Encoding header.
type Gzip struct{}

func (gmw Gzip) Handler(ph http.Handler, c context.Context) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := util.NewRecorderHijacker(w)
		useGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		ph.ServeHTTP(rec, r)

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Header().Set("Vary", "Accept-Encoding")

		if useGzip {
			w.Header().Set("Content-Encoding", "gzip")

			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", http.DetectContentType(rec.GetBody().Bytes()))
			}
		}

		if useGzip {
			buf := util.BufferPool.GetBuffer()
			defer util.BufferPool.Put(buf)

			gz := gzip.NewWriter(buf)

			if _, err := gz.Write(rec.GetBody().Bytes()); err != nil {
				panic(err)
			}
			gz.Close()

			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

			w.WriteHeader(rec.GetCode())

			buf.WriteTo(w)
		} else {
			w.WriteHeader(rec.GetCode())

			w.Write(rec.GetBody().Bytes())
		}
	}

	return http.HandlerFunc(handler)
}
