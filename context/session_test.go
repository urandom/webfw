package context

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var secret = []byte("test")

func TestSession(t *testing.T) {
	s := NewSession(secret, nil, os.TempDir())
	s.SetName("test1")

	if s.Name() != "test1" {
		t.Fatalf("Expected the session name to be 'test1', got '%s'\n", s.Name())
	}

	if s.MaxAge() != time.Hour {
		t.Fatalf("Expected the session max-age to be a duration of 1h, got '%s'\n", s.MaxAge())
	}

	if s.(*session).Path != os.TempDir() {
		t.Fatalf("Expected the session path to be set to the os temp-dir, got '%s'\n", s.(*session).Path)
	}

	context := NewContext()
	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test2", r, nil)

	s2 := NewSession(secret, nil, os.TempDir())

	err := s2.Read(r, context)
	if err != nil {
		t.Fatal(err)
	}

	temp := NewSession(secret, nil, os.TempDir())

	err = temp.Read(r, context)
	if err != nil {
		t.Fatal(err)
	}

	if temp.Name() != s2.Name() {
		t.Fatal()
	}

	if temp.MaxAge() != s2.MaxAge() {
		t.Fatal()
	}

	if temp.CookieName() != s2.CookieName() {
		t.Fatal()
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test3", r, nil)

	temp = NewSession(secret, nil, os.TempDir())

	if err = temp.Read(r, nil); err != nil && err != ErrExpired && err != ErrNotExist {
		t.Fatal(err)
	}

	s2.Set("foo", "bar")

	if val, ok := s2.Get("foo"); ok {
		if val != "bar" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	rec := httptest.NewRecorder()
	if err := s2.Write(rec); err != nil {
		t.Fatalf("%s", err)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test2", r, nil)

	s2 = NewSession(secret, nil, os.TempDir())
	err = s2.Read(r, context)

	if err != nil {
		t.Fatal(err)
	}

	if val, ok := s2.Get("foo"); ok {
		if val != "bar" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test2", r, nil)

	/* Load from filesystem */
	s2 = NewSession(secret, nil, os.TempDir())
	err = s2.Read(r, nil)

	if err != nil {
		t.Fatal(err)
	}

	if val, ok := s2.Get("foo"); ok {
		if val != "bar" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	s2.Delete("foo")
	if _, ok := s2.Get("foo"); ok {
		t.Fatal()
	}

	s2.Set("foo", "bar")
	s2.Set("alpha", "beta")

	if _, ok := s2.Get("foo"); !ok {
		t.Fatal()
	}

	if _, ok := s2.Get("alpha"); !ok {
		t.Fatal()
	}

	s2.DeleteAll()
	if _, ok := s2.Get("foo"); ok {
		t.Fatal()
	}

	if _, ok := s2.Get("alpha"); ok {
		t.Fatal()
	}

	flash := s2.Flash("foo")

	if flash != nil {
		t.Fatal()
	}

	s2.SetFlash("foo", "bar")

	flash = s2.Flash("foo")

	if flash.(string) != "bar" {
		t.Fatal()
	}

	flash = s2.Flash("foo")

	if flash != nil {
		t.Fatal()
	}

}

func TestSecure(t *testing.T) {
	cipher, err := base64.StdEncoding.DecodeString(`HsPW6w85KMiTNm7q5ZaruE/f3Hl9wlKFYP8AyYF/N7s=`)
	if err != nil {
		t.Fatal(err)
	}

	s := NewSession(secret, cipher, os.TempDir())
	s.SetName("test4")

	s.Set("foo", "bar")

	r, _ := http.NewRequest("GET", "http://localhost:8080/some/url", nil)
	rec := httptest.NewRecorder()

	if err := s.Write(rec); err != nil {
		t.Fatal(err)
	}

	cookie := rec.Header().Get("Set-Cookie")
	r.Header.Set("Cookie", cookie[:strings.Index(cookie, ";")])

	context := NewContext()

	s = NewSession(secret, cipher, os.TempDir())
	if err := s.Read(r, context); err != nil {
		t.Fatal(err)
	}

	if v, ok := s.Get("foo"); !ok || v.(string) != "bar" {
		t.Fatalf("Expected value for `foo` to be 'bar', got '%s'\n", v.(string))
	}
}

func TestCleanup(t *testing.T) {
	root := filepath.Join(os.TempDir(), "/sessions/")

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test1", r, nil)

	s := NewSession(secret, nil, root)

	if err := s.Read(r, nil); err != nil && err != ErrExpired && err != ErrNotExist {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, "test1")); !os.IsNotExist(err) {
		t.Fatal("Session 'test1' already exists")
	}

	s.Set("test", "test2")

	rec := httptest.NewRecorder()
	s.Write(rec)

	if _, err := os.Stat(filepath.Join(root, "test1")); os.IsNotExist(err) {
		t.Fatal(err)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test2", r, nil)

	s = NewSession(secret, nil, root)

	s.Read(r, nil)

	rec = httptest.NewRecorder()
	s.Write(rec)

	if _, err := os.Stat(filepath.Join(root, "test2")); os.IsNotExist(err) {
		t.Fatal(err)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test1", r, nil)

	s = NewSession(secret, nil, root)

	s.Read(r, nil)

	if v, ok := s.Get("test"); !ok || v.(string) != "test2" {
		t.Fatalf("Expected the value for 'test' to be 'test2', got '%s'\n", v)
	}

	if err := CleanupSessions(root, 0); err != nil {
		t.Fatal(err)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	addCookie(t, "test3", r, nil)

	s = NewSession(secret, nil, root)

	s.Read(r, nil)
	s.SetMaxAge(time.Second)
	s.Set("test", "test2")

	rec = httptest.NewRecorder()
	s.Write(rec)

	s = NewSession(secret, nil, root)
	s.SetName("test3")

	if err := s.Read(r, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	s = NewSession(secret, nil, root)
	s.SetName("test3")

	if err := s.Read(r, nil); err != ErrExpired {
		t.Fatalf("Expected an expiration error, got '%s'\n", err)
	}

	if _, ok := s.Get("test"); ok {
		t.Fatalf("Expected the session to not have values\n")
	}

	if _, err := os.Stat(filepath.Join(root, "test3")); os.IsNotExist(err) {
		t.Fatal(err)
	}

	if err := CleanupSessions(root, time.Second); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, "test3")); !os.IsNotExist(err) {
		t.Fatalf("Expected the session 'test3' to not exist anymore")
	}

}

func addCookie(t *testing.T, name string, r *http.Request, cipher []byte) {
	now := time.Now().Unix()
	hm := hmac.New(sha256.New, secret)

	message := []byte(fmt.Sprintf("%s|%s|%d", "session", name, now))

	if _, err := hm.Write(message); err != nil {
		t.Fatal(err)
	}

	mac := hm.Sum(nil)

	sig := []byte(fmt.Sprintf("%s|%d|%s", name, now, mac))

	encoded := base64.URLEncoding.EncodeToString(sig)

	cookie := &http.Cookie{
		Name:  "session",
		Value: string(encoded),
	}

	r.AddCookie(cookie)
}
