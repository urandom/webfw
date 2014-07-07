package webfw

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/middleware"
)

func TestContextUtil(t *testing.T) {
	c := context.NewContext()
	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)

	conf := GetConfig(c, r)
	if fmt.Sprintf("%v", conf) != fmt.Sprintf("%v", Config{}) {
		t.Fatalf("Expected a an empty Config, got %v\n", conf)
	}

	conf.Server.Host = "example.com"
	c.Set(r, context.BaseCtxKey("config"), conf)

	conf = GetConfig(c, r)
	if conf.Server.Host != "example.com" {
		t.Fatalf("Expected Server.Host to be 'example.com', got %s\n", conf.Server.Host)
	}

	params := GetParams(c, r)
	if len(params) != 0 {
		t.Fatalf("Expected empty params, got %v\n", params)
	}

	params["foo"] = "bar"
	c.Set(r, context.BaseCtxKey("params"), params)

	params = GetParams(c, r)
	if len(params) != 1 {
		t.Fatalf("Expected params with 1 entry, got %v\n", params)
	}

	if v := params["foo"]; v != "bar" {
		t.Fatalf("Expected param value for 'foo' to be 'bar', got %s\n", v)
	}

	sess := GetSession(c, r)
	if sess.Name() == "" {
		t.Fatalf("Expected a non-empty session name\n")
	}

	if len(sess.GetAll()) != 0 {
		t.Fatalf("Expected an empty session, got %v\n", sess.GetAll())
	}

	sess.Set("foo", "bar")
	uuid := sess.Name()
	c.Set(r, context.BaseCtxKey("session"), sess)

	sess = GetSession(c, r)
	if sess.Name() != uuid {
		t.Fatalf("Expected Session.Name '%s', got '%s'\n", uuid, sess.Name())
	}

	if v, ok := sess.Get("foo"); ok {
		if v != "bar" {
			t.Fatalf("Expected the value for session key 'foo' to be 'bar', got '%v'\n", v)
		}
	} else {
		t.Fatalf("Expected the session to have a value for key 'foo'\n")
	}

	lang := GetLanguage(c, r)

	if lang != middleware.FallbackLocale(c, r) {
		t.Fatalf("Expected lang to be '%s', got '%s'\n", middleware.FallbackLocale(c, r), lang)
	}

	c.Set(r, context.BaseCtxKey("lang"), "ZZ")

	lang = GetLanguage(c, r)
	if lang != "ZZ" {
		t.Fatalf("Expected lang to be 'ZZ', got '%s'\n", lang)
	}

	ren := GetRenderer(c, r)
	if ren == nil {
		t.Fatalf("Expected a non-nil renderer\n")
	}

	log := GetLogger(c, r)
	if log == nil {
		t.Fatalf("Expected a non-nill logger\n")
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	conf = GetConfig(c, r)
	if conf.Server.Host != "" {
		t.Fatalf("Expected Server.Host to be empty, got %s\n", conf.Server.Host)
	}

	params = GetParams(c, r)
	if len(params) != 0 {
		t.Fatalf("Expected empty params, got %v\n", params)
	}
}
