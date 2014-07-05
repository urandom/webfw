package webfw

import (
	"fmt"
	"testing"
)

func TestConfigRead(t *testing.T) {
	c, err := ReadConfig("testdata/server.conf")

	if err != nil {
		t.Fatal(err)
	}

	testconf(t, c)
}

func TestConfigParse(t *testing.T) {
	c, err := ParseConfig(conf)

	if err != nil {
		t.Fatal(err)
	}

	testconf(t, c)
}

func testconf(t *testing.T, c Config) {
	if c.Server.Host != "localhost" {
		t.Fatalf("Expected Server.Host 'localhost', got %s\n", c.Server.Host)
	}

	if c.Server.Port != 10000 {
		t.Fatalf("Expected Server.Port 10000, got %d\n", c.Server.Port)
	}

	if c.Server.Devel {
		t.Fatalf("Expected Server.Devel to be false")
	}

	if c.Renderer.Dir != "tempdir" {
		t.Fatalf("Expected Renderer.Dir 'tempdir', got %s\n", c.Renderer.Dir)
	}

	fmt.Printf("%v\n", c.Dispatcher.Middleware)

	if len(c.Dispatcher.Middleware) != 3 {
		t.Fatalf("Expected Dispatcher.Middleware to have 3 entries\n")
	}

	if c.Dispatcher.Middleware[0] != "Session" {
		t.Fatalf("Expected Dispatcher.Middleware[0] to be 'Session'\n")
	}

	if c.Dispatcher.Middleware[1] != "Static" {
		t.Fatalf("Expected Dispatcher.Middleware[1] to be 'Static'\n")
	}

	if c.Dispatcher.Middleware[2] != "Logger" {
		t.Fatalf("Expected Dispatcher.Middleware[2] to be 'Logger'\n")
	}

	if c.Dispatcher.Middleware[2] != "Logger" {
		t.Fatalf("Expected Dispatcher.Middleware[2] to be 'Logger'\n")
	}

	if c.I18n.Dir != "langdir" {
		t.Fatalf("Expected I18n.Dir to be 'langdir', got %s\n", c.I18n.Dir)
	}

	if len(c.I18n.Languages) != 2 {
		t.Fatalf("Expected I18n.Languages to have 2 entries\n")
	}

	if c.I18n.Languages[0] != "fr" {
		t.Fatalf("Expected I18n.Languages[0] to be 'fr'\n")
	}

	if c.I18n.Languages[1] != "de" {
		t.Fatalf("Expected I18n.Languages[1] to be 'de'\n")
	}

	// Default values
	if c.Static.Dir != "static" {
		t.Fatalf("Expected Static.Dir default value, got '%s'\n", c.Static.Dir)
	}

	if c.Static.Expires != "5m" {
		t.Fatalf("Expected Static.Expires default value, got '%s'\n", c.Static.Expires)
	}

	if c.Session.Dir != "session" {
		t.Fatalf("Expected Session.Dir default value, got '%s'\n", c.Session.Dir)
	}

	if c.Session.MaxAge != "360h" {
		t.Fatalf("Expected Session.MaxAge default value, got %s\n", c.Session.MaxAge)
	}

}

var conf = `
[server]
    host = localhost
    port = 10000
    devel = false

[renderer]
    dir = tempdir

[dispatcher]
    middleware = Session
    middleware = Static
    middleware = Logger

[i18n]
    dir = langdir
    language = fr
    language = de
`
