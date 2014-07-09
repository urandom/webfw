package middleware

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/renderer"
)

func TestI18NHandler(t *testing.T) {
	c := context.NewContext()
	l := log.New(os.Stderr, "", 0)
	ren := renderer.NewRenderer("testdata", "test.tmpl")

	mw := I18N{
		Languages:       []string{"en", "bg"},
		Pattern:         "/",
		Renderer:        ren,
		Dir:             "testdata",
		IgnoreURLPrefix: []string{"/css", "/js"},
	}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := ren.Render(w, nil, c.GetAll(r), "test_i18n.tmpl"); err != nil {
			t.Fatal(err)
		}
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code %d, got %d\n", http.StatusFound, rec.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/bg/", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code %d, got %d\n", http.StatusOK, rec.Code)
	}

	expected := "test data bg"
	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf("Expected body '%s' to contain '%s'\n", rec.Body.String(), expected)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/en/", nil)
	rec = httptest.NewRecorder()

	s := context.NewSession([]byte(""), nil, os.TempDir())
	s.SetName("test1")
	c.Set(r, context.BaseCtxKey("session"), s)
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code %d, got %d\n", http.StatusOK, rec.Code)
	}

	expected = "test data en"
	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf("Expected body '%s' to contain '%s'\n", rec.Body.String(), expected)
	}

	if s, ok := s.Get("language"); ok {
		if s.(string) != "en" {
			t.Fatalf("Expected session.language to be '%s', got '%s'\n", "en", s.(string))
		}
	} else {
		t.Fatalf("Expected the session to have a language key\n")
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/en", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code %d, got %d\n", http.StatusFound, rec.Code)
	}

	expected = "/en/"
	if rec.Header().Get("Location") != expected {
		t.Fatalf("Expected a redirect to '%s', got '%s'\n", expected, rec.Header().Get("Location"))
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/foo/bar/baz", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code %d, got %d\n", http.StatusFound, rec.Code)
	}

	expected = "/en/foo/bar/baz"
	if rec.Header().Get("Location") != expected {
		t.Fatalf("Expected a redirect to '%s', got '%s'\n", expected, rec.Header().Get("Location"))
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/foo/bar/baz?alpha=beta&gamma=delta", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code %d, got %d\n", http.StatusFound, rec.Code)
	}

	expected = "/en/foo/bar/baz?alpha=beta&gamma=delta"
	if rec.Header().Get("Location") != expected {
		t.Fatalf("Expected a redirect to '%s', got '%s'\n", expected, rec.Header().Get("Location"))
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/foo/bar/baz?alpha=beta&gamma=delta#test", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code %d, got %d\n", http.StatusFound, rec.Code)
	}

	expected = "/en/foo/bar/baz?alpha=beta&gamma=delta#test"
	if rec.Header().Get("Location") != expected {
		t.Fatalf("Expected a redirect to '%s', got '%s'\n", expected, rec.Header().Get("Location"))
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/css/foo", nil)
	rec = httptest.NewRecorder()

	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}), c, l)

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code %d, got %d\n", http.StatusOK, rec.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/js/foo", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code %d, got %d\n", http.StatusOK, rec.Code)
	}
}
