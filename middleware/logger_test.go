package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"webfw/context"
)

func TestLogger(t *testing.T) {
	c := context.NewContext()
	buf := new(bytes.Buffer)

	l := log.New(os.Stderr, "", 0)

	mw := Logger{AccessLogger: log.New(buf, "", 0)}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Test"))
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected := `"GET /" 200 4`
	if !strings.Contains(buf.String(), expected) {
		t.Fatalf("Expected '%s' to contain '%s'\n", buf.String(), expected)
	}
}
