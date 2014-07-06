package webfw

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/urandom/webfw/types"
)

func TestDispatcherIncorrectInit(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("NewDispatcher was expected to fail\n")
		}
	}()

	NewDispatcher("/foo", Config{})
}

func TestDispatcherMiddlewareRegistration(t *testing.T) {
	c := Config{}
	c.Dispatcher.Middleware = []string{"Static", "Error"}

	d := NewDispatcher("/", c)

	mw := MyCustomMW{to: "/test"}
	mw2 := MyCustomMW2{to: "/another-test"}
	d.RegisterMiddleware(mw)
	d.RegisterMiddleware(mw2)

	if _, ok := d.middleware["MyCustomMW"]; !ok {
		t.Fatalf("Expected middleware to be registered as 'MyCustomMW'\n")
	}

	if mw != d.middleware["MyCustomMW"] {
		t.Fatalf("Expected middleware %v, got %v\n", mw, d.middleware["MyCustomMW"])
	}

	if _, ok := d.middleware["MyCustomMW2"]; !ok {
		t.Fatalf("Expected middleware to be registered as 'MyCustomMW2'\n")
	}

	if mw2 != d.middleware["MyCustomMW2"] {
		t.Fatalf("Expected middleware %v, got %v\n", mw2, d.middleware["MyCustomMW2"])
	}

	if d.middlewareOrder[0] != "MyCustomMW" {
		t.Fatalf("Expected the first custom middleware to be 'MyCustomMW'")
	}

	if d.middlewareOrder[1] != "MyCustomMW2" {
		t.Fatalf("Expected the first custom middleware to be 'MyCustomMW2'")
	}

	d.init()

	order := []string{"MyCustomMW", "MyCustomMW2", "Static", "Error"}
	for i, m := range order {
		if m != d.middlewareOrder[i] {
			t.Fatalf("Expected middleware '%s' at position %d, got '%s'\n", m, i, d.middlewareOrder[i])
		}
	}

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, r)

	if v, ok := d.context.Get(r, "foo"); !ok || v.(string) != "/test" {
		t.Fatalf("Expected MyCustomMW to be called be last\n")
	}

	c = Config{}
	c.Dispatcher.Middleware = []string{"Static", "MyCustomMW", "Error"}

	d = NewDispatcher("/", c)

	mw = MyCustomMW{to: "/test"}
	mw2 = MyCustomMW2{to: "/another-test"}
	d.RegisterMiddleware(mw)
	d.RegisterMiddleware(mw2)

	d.init()

	order = []string{"MyCustomMW2", "Static", "MyCustomMW", "Error"}
	for i, m := range order {
		if m != d.middlewareOrder[i] {
			t.Fatalf("Expected middleware '%s' at position %d, got '%s'\n", m, i, d.middlewareOrder[i])
		}
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)

	if v, ok := d.context.Get(r, "foo"); !ok || v.(string) != "/another-test" {
		t.Fatalf("Expected MyCustomMW2 to be called be last\n")
	}

}

func TestDispatcherHandle(t *testing.T) {
	d := NewDispatcher("/", Config{})
	d.init()

	c1 := controller{
		pattern: "/",
		method:  types.MethodAll,
		handler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "" && r.URL.Path != "/" {
				t.Fatalf("Expected the url path to be '' or '/', got %v\n", r.URL.Path)
			}
		},
	}

	d.Handle(c1)

	c2 := controller{
		pattern: "/hello/:name",
		method:  types.MethodAll,
		handler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/hello/World" {
				t.Fatalf("Expected the url path to be '/hello/World', got %v\n", r.URL.Path)
			}

			if GetParams(d.context, r)["name"] != "World" {
				t.Fatalf("Expected the name parameter to be 'World', got %v\n", GetParams(d.context, r)["name"])
			}
		},
	}

	d.Handle(c2)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/hello/World", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/test", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected StatusNotFound, got %v\n", w.Code)
	}
}

type controller struct {
	handler http.HandlerFunc
	pattern string
	name    string
	method  types.Method
}

func (cntl controller) Handler(c types.Context) http.HandlerFunc {
	return cntl.handler
}

func (cntl controller) Pattern() string {
	return cntl.pattern
}

func (cntl controller) Name() string {
	return cntl.name
}

func (cntl controller) Method() types.Method {
	return cntl.method
}

type MyCustomMW struct {
	to string
}

func (mmw MyCustomMW) Handler(ph http.Handler, c types.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mmw.to
		c.Set(r, "foo", mmw.to)
		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}

type MyCustomMW2 struct {
	to string
}

func (mmw MyCustomMW2) Handler(ph http.Handler, c types.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mmw.to
		c.Set(r, "foo", mmw.to)
		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
