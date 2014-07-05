package middleware

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"webfw/context"
)

func TestContext(t *testing.T) {
	c := context.NewContext()

	l := log.New(os.Stderr, "", 0)

	mw := Context{}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Set(r, "alpha", "beta")
		w.Write([]byte("Test"))
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if _, ok := c.Get(r, "foo"); ok {
		t.Fatalf("Expected the context to get deleted for this request\n")
	}

	if _, ok := c.Get(r, "alpha"); ok {
		t.Fatalf("Expected the context to get deleted for this request\n")
	}
}
