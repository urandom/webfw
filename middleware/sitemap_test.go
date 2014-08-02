package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
)

func TestSitemapHandler(t *testing.T) {
	c := context.NewContext()
	l := log.New(os.Stderr, "", 0)
	mw := Sitemap{
		Pattern:          "/",
		Prefix:           "http://example.com/",
		RelativeLocation: "sitemap.xml",
		Controllers:      []webfw.SitemapController{},
	}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}), c, l)

	r, _ := http.NewRequest("GET", "http://example.com/en.all.json", nil)
	r.RequestURI = "/en.all.json"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected := []byte("test")
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	r, _ = http.NewRequest("GET", "http://example.com/sitemap.xml", nil)
	r.RequestURI = "/sitemap.xml"
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	
</urlset>
`)

	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	mw = Sitemap{
		Pattern:          "/",
		Prefix:           "http://example.com/",
		RelativeLocation: "sitemap2.xml",
		Controllers: []webfw.SitemapController{
			sc{[]webfw.SitemapItem{webfw.SitemapItem{
				Loc:        "/foo",
				LastMod:    webfw.SitemapNoLastMod,
				ChangeFreq: webfw.SitemapFrequencyDaily,
				Priority:   0.5,
			}}},
		},
	}

	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}), c, l)

	r, _ = http.NewRequest("GET", "http://example.com/sitemap2.xml", nil)
	r.RequestURI = "/sitemap2.xml"
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	
		<url>
			<loc>http://example.com/foo</loc>
			
			
			<changefreq>daily</changefreq>
			
			
			<priority>0.5</priority>
			
		</url>
   
</urlset>
`)

	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	mod := time.Now()
	mw = Sitemap{
		Pattern:          "/",
		Prefix:           "http://example.com/",
		RelativeLocation: "sitemap2.xml",
		Controllers: []webfw.SitemapController{
			sc{[]webfw.SitemapItem{
				webfw.SitemapItem{
					Loc:        "/hello/john",
					LastMod:    webfw.SitemapNoLastMod,
					ChangeFreq: webfw.SitemapFrequencyDaily,
					Priority:   0.5,
				},
				webfw.SitemapItem{
					Loc:        "/hello/smith",
					LastMod:    webfw.SitemapNoLastMod,
					ChangeFreq: webfw.SitemapFrequencyMonthly,
					Priority:   0.9,
				},
			}},
			sc{[]webfw.SitemapItem{webfw.SitemapItem{
				Loc:        "/foo",
				LastMod:    mod,
				ChangeFreq: webfw.SitemapNoFrequency,
				Priority:   webfw.SitemapNoPriority,
			}}},
		},
	}

	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}), c, l)

	r, _ = http.NewRequest("GET", "http://example.com/sitemap2.xml", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	
		<url>
			<loc>http://example.com/hello/john</loc>
			
			
			<changefreq>daily</changefreq>
			
			
			<priority>0.5</priority>
			
		</url>
   
		<url>
			<loc>http://example.com/hello/smith</loc>
			
			
			<changefreq>monthly</changefreq>
			
			
			<priority>0.9</priority>
			
		</url>
   
		<url>
			<loc>http://example.com/foo</loc>
			
			<lastmod>2014-07-30</lastmod>
			
			
			
		</url>
   
</urlset>
`)

	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

}

type sc struct {
	items []webfw.SitemapItem
}

func (c sc) Sitemap(cont context.Context) []webfw.SitemapItem {
	return c.items
}
