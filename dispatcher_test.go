package webfw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/urandom/webfw/context"
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

	d.Initialize()

	order := []string{"MyCustomMW", "MyCustomMW2"}
	for i, m := range order {
		if m != d.middlewareOrder[i] {
			t.Fatalf("Expected middleware '%s' at position %d, got '%s'\n", m, i, d.middlewareOrder[i])
		}
	}

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, r)

	if v, ok := d.Context.Get(r, "foo"); !ok || v.(string) != "/test" {
		t.Fatalf("Expected MyCustomMW to be called be last\n")
	}

	c = Config{}
	c.Dispatcher.Middleware = []string{"Static", "MyCustomMW", "Error"}

	d = NewDispatcher("/", c)

	mw = MyCustomMW{to: "/test"}
	mw2 = MyCustomMW2{to: "/another-test"}
	d.RegisterMiddleware(mw)
	d.RegisterMiddleware(mw2)

	d.Initialize()

	order = []string{"MyCustomMW2", "MyCustomMW"}
	for i, m := range order {
		if m != d.middlewareOrder[i] {
			t.Fatalf("Expected middleware '%s' at position %d, got '%s'\n", m, i, d.middlewareOrder[i])
		}
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)

	if v, ok := d.Context.Get(r, "foo"); !ok || v.(string) != "/another-test" {
		t.Fatalf("Expected MyCustomMW2 to be called be last\n")
	}

}

func TestDispatcherHandle(t *testing.T) {
	d := NewDispatcher("/", Config{})
	d.Initialize()

	c1 := controller{
		pattern: "/",
		method:  MethodAll,
		handler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "" && r.URL.Path != "/" {
				t.Fatalf("Expected the url path to be '' or '/', got %v\n", r.URL.Path)
			}
		},
	}

	d.Handle(c1)

	c2 := controller{
		pattern: "/hello/:name",
		method:  MethodAll,
		handler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/hello/World" {
				t.Fatalf("Expected the url path to be '/hello/World', got %v\n", r.URL.Path)
			}

			if GetParams(d.Context, r)["name"] != "World" {
				t.Fatalf("Expected the name parameter to be 'World', got %v\n", GetParams(d.Context, r)["name"])
			}
		},
	}

	d.Handle(c2)

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	r.RequestURI = ""
	w := httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	r.RequestURI = "/"
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/hello/World", nil)
	r.RequestURI = "/hello/World"
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected StatusOk, got %v\n", w.Code)
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/test", nil)
	r.RequestURI = "/test"
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected StatusNotFound, got %v\n", w.Code)
	}

	passedC3, passedC4, passedC5M1, passedC5M2 := false, false, false, false
	c3NameParam, c4NameParam, c5M1Param, c5M2Param := "", "", "", ""
	c3 := controller{
		pattern: "/test/:name",
		method:  MethodAll,
		name:    "named1",
		handler: func(w http.ResponseWriter, r *http.Request) {
			passedC3 = true
			c3NameParam = GetParams(d.Context, r)["name"]

			d.Context.Set(r, context.BaseCtxKey("forward"), "/test2/stuff")
		},
	}

	c4 := controller{
		pattern: "/test2/:foo",
		method:  MethodAll,
		handler: func(w http.ResponseWriter, r *http.Request) {
			passedC4 = true
			c4NameParam = GetParams(d.Context, r)["foo"]
		},
	}

	passedMW3 := new(bool)
	*passedMW3 = false
	mw := MyCustomMW3{passed: passedMW3}

	d = NewDispatcher("/", Config{})

	d.Handle(c3)
	d.Handle(c4)
	d.RegisterMiddleware(mw)

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	w = httptest.NewRecorder()

	d.Initialize()
	d.ServeHTTP(w, r)

	if !*passedMW3 {
		t.Fatalf("Expected to pass through middleware 3\n")
	}

	if !passedC3 {
		t.Fatalf("Expected to pass through controller 3\n")
	}

	if !passedC4 {
		t.Fatalf("Expected to pass through controller 4\n")
	}

	if c3NameParam != "foo" {
		t.Fatalf("Expected controller 3 name parameter to be 'foo', got '%s'\n", c3NameParam)
	}

	if c4NameParam != "stuff" {
		t.Fatalf("Expected controller 4 foo parameter to be 'stuff', got '%s'\n", c4NameParam)
	}

	path := d.NameToPath("named1", MethodAll, RouteParams{"name": "stuff"})
	if path == "" {
		t.Fatalf("The named route 'named1' wasn't found\n")
	}

	if path != "/test/stuff" {
		t.Fatalf("Expected '/test/stuff', got '%s'\n", path)
	}

	c5 := multicontroller{
		controller: controller{
			handler: func(w http.ResponseWriter, r *http.Request) {
				id := GetMultiPatternIdentifier(d.Context, r)
				params := GetParams(d.Context, r)
				if id == "multi1" {
					passedC5M1 = true
					c5M1Param = params["foo"]
				} else if id == "multi2" {
					passedC5M2 = true
					c5M2Param = params["bar"]
				} else {
					t.Fatalf("Unexpected MultiPattern Identifier: '%s'\n", id)
				}
			},
		},
		patterns: map[string]MethodIdentifierTuple{
			"/multi1/:foo": MethodIdentifierTuple{MethodGet, "multi1"},
			"/multi2/:bar": MethodIdentifierTuple{MethodGet | MethodPost, "multi2"},
		},
	}

	r, _ = http.NewRequest("GET", "http://localhost:8080/multi1/test1", nil)
	w = httptest.NewRecorder()

	d = NewDispatcher("/", Config{})
	d.HandleMultiPattern(c5)
	d.Initialize()
	d.ServeHTTP(w, r)

	if !passedC5M1 {
		t.Fatal("Expected to pass through controller 5-multi1")
	}

	if passedC5M2 || c5M2Param != "" {
		t.Fatal("Did not expect to pass through controller 5-multi2")
	}

	expectedStr := "test1"
	if c5M1Param != expectedStr {
		t.Fatalf("Expected c5M1 param to be %s, got '%s'\n", expectedStr, c5M1Param)
	}

	passedC5M1, passedC5M2 = false, false
	c5M1Param, c5M2Param = "", ""

	r, _ = http.NewRequest("GET", "http://localhost:8080/multi2/test2", nil)
	w = httptest.NewRecorder()

	d.ServeHTTP(w, r)

	if !passedC5M2 {
		t.Fatal("Expected to pass through controller 5-multi2")
	}

	if passedC5M1 || c5M1Param != "" {
		t.Fatal("Did not expect to pass through controller 5-multi1")
	}

	expectedStr = "test2"
	if c5M2Param != expectedStr {
		t.Fatalf("Expected c5M1 param to be %s, got '%s'\n", expectedStr, c5M2Param)
	}

	d = NewDispatcher("/prefix/", Config{})

	d.Handle(c3)
	d.Initialize()

	path = d.NameToPath("named1", MethodAll, RouteParams{"name": "stuff"})
	if path == "" {
		t.Fatalf("The named route 'named1' wasn't found\n")
	}

	if path != "/prefix/test/stuff" {
		t.Fatalf("Expected '/prefix/test/stuff', got '%s'\n", path)
	}

}

type controller struct {
	handler http.HandlerFunc
	pattern string
	name    string
	method  Method
}

func (cntl controller) Handler(c context.Context) http.HandlerFunc {
	return cntl.handler
}

func (cntl controller) Pattern() string {
	return cntl.pattern
}

func (cntl controller) Name() string {
	return cntl.name
}

func (cntl controller) Method() Method {
	return cntl.method
}

type multicontroller struct {
	controller
	patterns map[string]MethodIdentifierTuple
}

func (cntl multicontroller) Patterns() map[string]MethodIdentifierTuple {
	return cntl.patterns
}

type MyCustomMW struct {
	to string
}

func (mmw MyCustomMW) Handler(ph http.Handler, c context.Context) http.Handler {
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

func (mmw MyCustomMW2) Handler(ph http.Handler, c context.Context) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mmw.to
		r.RequestURI = mmw.to
		c.Set(r, "foo", mmw.to)
		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}

type MyCustomMW3 struct {
	passed *bool
}

func (mmw MyCustomMW3) Handler(ph http.Handler, c context.Context) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		*mmw.passed = true
		c.Set(r, context.BaseCtxKey("named-forward"), "named1")
		c.Set(r, context.BaseCtxKey("params"), RouteParams{"name": "foo"})
		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
