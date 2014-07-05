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
	"github.com/urandom/webfw/types"
)

func TestUrlHandler(t *testing.T) {
	c := context.NewContext()
	l := log.New(os.Stderr, "", 0)
	ren := renderer.NewRenderer("testdata", "test.tmpl")

	mw := Url{
		Renderer: ren,
	}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := ren.Render(w, nil, c.GetAll(r), "test_url.tmpl"); err != nil {
			t.Fatal(err)
		}
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080/some/url", nil)
	rec := httptest.NewRecorder()
	c.Set(r, types.BaseCtxKey("r"), r)
	c.Set(r, types.BaseCtxKey("lang"), "en")

	h.ServeHTTP(rec, r)

	content := rec.Body.String()

	expected := "[url1: /en/foo/bar]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url2: /de/foo/bar]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url3: /en/alpha/some/url/beta]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url4: /de/alpha/some/url/beta]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url5: /en/some/url/test/1]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url6: /de/some/url/test/2]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url7: /en/some/url/beta]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url8: /de/some/url/beta]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url9: /en/some/url/beta/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url10: /de/some/url/beta/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url11: /en/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url12: /de/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url13: /en/foo?bar=baz&amp;alpha=/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}

	expected = "[url14: /de/foo?bar=baz&amp;alpha=/some/url]"
	if !strings.Contains(content, expected) {
		t.Fatalf("Expected '%s' in '%s'\n", expected, content)
	}
}
