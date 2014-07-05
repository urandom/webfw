package middleware

import (
	"bytes"
	"compress/gzip"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/urandom/webfw/context"
)

func TestGzipHandler(t *testing.T) {
	c := context.NewContext()
	l := log.New(os.Stderr, "", 0)

	mw := Gzip{}

	testContent := []byte("Test this string")
	buf := new(bytes.Buffer)

	gz := gzip.NewWriter(buf)
	if _, err := gz.Write(testContent); err != nil {
		t.Fatal(err)
	}
	gz.Close()

	gzippedContent := buf.Bytes()

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(testContent)
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if !bytes.Equal(rec.Body.Bytes(), testContent) {
		t.Fatalf("Expected '%s', got '%s'\n", rec.Body.String(), string(testContent[:]))
	}

	r.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if !bytes.Equal(rec.Body.Bytes(), gzippedContent) {
		t.Fatalf("Expected '%s', got '%s'\n", rec.Body.String(), string(gzippedContent[:]))
	}

}
