package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/urandom/webfw/context"
)

var secret = []byte("test")

func TestSessionHandler(t *testing.T) {
	c := context.NewContext()
	mw := Session{
		Path:            path.Join(os.TempDir(), "session"),
		Secret:          secret,
		MaxAge:          "1s",
		CleanupInterval: "1s",
		CleanupMaxAge:   "1s",
	}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}), c)

	r, _ := http.NewRequest("GET", "http://localhost:8080/some/url", nil)
	rec := httptest.NewRecorder()
	c.Set(r, context.BaseCtxKey("r"), r)
	c.Set(r, context.BaseCtxKey("lang"), "en")

	h.ServeHTTP(rec, r)

	var cookie string
	if s, ok := c.Get(r, context.BaseCtxKey("session")); ok {
		sess := s.(context.Session)
		if sess.MaxAge() != time.Second {
			t.Fatalf("Expected Session.MaxAge to be '%s', got '%s'\n", time.Second, sess.MaxAge())
		}

		if err := sess.Write(rec); err != nil {
			t.Fatal(err)
		}
		sess.Set("foo", "bar")
		cookie = rec.Header().Get("Set-Cookie")
	} else {
		t.Fatalf("Expected a new session")
	}

	if ft, ok := c.Get(r, context.BaseCtxKey("firstTimer")); ok {
		if !ft.(bool) {
			t.Fatalf("Expected a true first-timer flag")
		}
	} else {
		t.Fatalf("Expected a first-timer flag")
	}

	time.Sleep(2 * time.Second)

	r, _ = http.NewRequest("GET", "http://localhost:8080/some/url", nil)
	rec = httptest.NewRecorder()
	r.Header.Set("Cookie", cookie[:strings.Index(cookie, ";")])

	h.ServeHTTP(rec, r)

	if ft, ok := c.Get(r, context.BaseCtxKey("firstTimer")); ok {
		if ft.(bool) {
			t.Fatalf("Expected a false first-timer flag")
		}
	} else {
		t.Fatalf("Expected a first-timer flag")
	}

	sess, _ := c.Get(r, context.BaseCtxKey("session"))
	if _, ok := sess.(context.Session).Get("foo"); ok {
		t.Fatalf("Expected the session to be empty")
	}

}
