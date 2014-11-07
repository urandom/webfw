package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/urandom/webfw/context"
)

func TestErrorHandler(t *testing.T) {
	c := context.NewContext()
	mw := Error{}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("Test")
	}), c)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected response code to be %d, got %d\n", http.StatusInternalServerError, rec.Code)
	}

	if rec.Body.String() != "Internal Server Error" {
		t.Fatalf("Expected ISE message, got '%s'\n", rec.Body.String())
	}

	mw = Error{ShowStack: true}

	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("Test")
	}), c)

	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected response code to be %d, got %d\n", http.StatusInternalServerError, rec.Code)
	}

	if !strings.HasPrefix(rec.Body.String(), fmt.Sprintf("%s - %s", time.Now().Format(dateFormat), "Test")) {
		t.Fatalf("Expected ISE message, got '%s'\n", rec.Body.String())
	}
}
