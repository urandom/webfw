package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"
	"webfw/context"
)

func TestStaticHandler(t *testing.T) {
	c := context.NewContext()
	l := log.New(os.Stderr, "", 0)

	mw := Static{
		Path: "testdata",
	}

	h := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}), c, l)

	r, _ := http.NewRequest("GET", "http://localhost:8080/en.all.json", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected := []byte("test")
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}), c, l)

	r, _ = http.NewRequest("GET", "http://localhost:8080/en.all.json", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	expected, _ = ioutil.ReadFile(path.Join("testdata", "en.all.json"))
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	f, _ := os.Open(path.Join("testdata", "en.all.json"))
	defer f.Close()

	stat, _ := f.Stat()
	lm := rec.Header().Get("Last-Modified")
	if lm != stat.ModTime().UTC().Format(http.TimeFormat) {
		t.Fatalf("Expected the Last-Modified header to be set to the file modtime of '%s', got '%s'\n", stat.ModTime().UTC().Format(http.TimeFormat), lm)
	}

	etag := rec.Header().Get("ETag")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", path.Join("/", "en.all.json"), stat.ModTime().Unix())))
	expectedStr := base64.URLEncoding.EncodeToString(hash[:])
	if etag != expectedStr {
		t.Fatalf("Expected ETag to be '%s', got '%s'\n", expectedStr, etag)
	}

	cc := rec.Header().Get("Cache-Control")
	if cc != "" {
		t.Fatalf("Expected an empty Cache-Control header, got '%s'\n", cc)
	}

	exp := rec.Header().Get("Expires")
	if exp != "" {
		t.Fatalf("Expected an empty Expires header, got '%s'\n", exp)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/dummy", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusFound, rec.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/dummy/", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusNotFound, rec.Code)
	}

	mw = Static{
		Path:  "testdata",
		Index: "dummy",
	}
	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}), c, l)

	r, _ = http.NewRequest("GET", "http://localhost:8080/dummy/", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusOK, rec.Code)
	}

	expected, _ = ioutil.ReadFile(path.Join("testdata", "dummy", "dummy"))
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	mw = Static{
		Path:   "testdata",
		Prefix: "/test",
	}
	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}), c, l)

	r, _ = http.NewRequest("GET", "http://localhost:8080/en.all.json", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusNotFound, rec.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/test/en.all.json", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusOK, rec.Code)
	}

	expected, _ = ioutil.ReadFile(path.Join("testdata", "en.all.json"))
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	mw = Static{
		Path:    "testdata",
		Expires: "1s",
	}
	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}), c, l)

	r, _ = http.NewRequest("GET", "http://localhost:8080/en.all.json", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusOK, rec.Code)
	}

	cc = rec.Header().Get("Cache-Control")
	if cc != "max-age=1" {
		t.Fatalf("Expected Cache-Control to be 'max-age=1' header, got '%s'\n", cc)
	}

	exp = rec.Header().Get("Expires")
	d, _ := time.ParseDuration("1s")
	expectedStr = time.Now().Add(d).Format(http.TimeFormat)
	if exp != expectedStr {
		t.Fatalf("Expected Expires to be '%s', got '%s'\n", expectedStr, exp)
	}

	mw = Static{
		Path:     "testdata",
		Index:    "dummy",
		FileList: true,
	}
	h = mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}), c, l)

	r, _ = http.NewRequest("GET", "http://localhost:8080/dummy/", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusOK, rec.Code)
	}

	expected, _ = ioutil.ReadFile(path.Join("testdata", "dummy", "dummy"))
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("Expected '%s', got '%s'\n", expected, rec.Body.Bytes())
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	rec = httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected code to be '%d', got '%d'\n", http.StatusOK, rec.Code)
	}

	expectedStr = `<a href="../">../</a>`
	if !strings.Contains(rec.Body.String(), expectedStr) {
		t.Fatalf("Expected file list to contain '%s', got '%s'\n", expectedStr, rec.Body.String())
	}

	expectedStr = `<a href="bg.all.json">bg.all.json</a>`
	if !strings.Contains(rec.Body.String(), expectedStr) {
		t.Fatalf("Expected file list to contain '%s', got '%s'\n", expectedStr, rec.Body.String())
	}

	expectedStr = `<a href="en.all.json">en.all.json</a>`
	if !strings.Contains(rec.Body.String(), expectedStr) {
		t.Fatalf("Expected file list to contain '%s', got '%s'\n", expectedStr, rec.Body.String())
	}

	expectedStr = `<a href="dummy/">dummy/</a>`
	if !strings.Contains(rec.Body.String(), expectedStr) {
		t.Fatalf("Expected file list to contain '%s', got '%s'\n", expectedStr, rec.Body.String())
	}
}
