package webfw

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/renderer"
)

// Dispatchers are responsible for calling controller handlers for a
// particular request path, passing it through middleware chain for
// each request. The middleware order can be configured via the server
// configuration. Multiple dispatchers may be created, and each one is
// defined by a ServerMux root pattern. The pattern must end in a '/'.
type Dispatcher struct {
	Pattern     string
	Host        string
	Context     context.Context
	Config      Config
	Logger      Logger
	Renderer    renderer.Renderer
	Controllers []Controller

	trie            *Trie
	handler         http.Handler
	middleware      map[string]Middleware
	middlewareOrder []string
}

// NewDispatcher creates a dispatcher for the given base pattern and config.
func NewDispatcher(pattern string, c Config) Dispatcher {
	if !strings.HasSuffix(pattern, "/") {
		panic("The dispatcher pattern has to end with a '/'")
	}

	d := Dispatcher{
		Pattern: pattern,
		Context: context.NewContext(),
		Config:  c,
		Logger:  log.New(os.Stderr, "", 0),

		trie:       NewTrie(),
		middleware: make(map[string]Middleware),
	}

	return d
}

// RegisterMiddleware registers the given middleware.
// If its name, derived from the type itself, is present within the
// dispatcher's middleware configuration, it will be used in that position,
// otherwise it will be added to the end of the chain, closest to the
// controller handler. Middleware, supplied by webfw may also be registered
// in this manner, if a more fine-grained configuration is desired.
func (d *Dispatcher) RegisterMiddleware(mw Middleware) {
	name := reflect.TypeOf(mw).Name()

	if _, ok := d.middleware[name]; !ok {
		d.middleware[name] = mw
		d.middlewareOrder = append(d.middlewareOrder, name)
	}
}

// Middleware returns a registered middleware based on the given name.
func (d Dispatcher) Middleware(name string) (Middleware, bool) {
	mw, ok := d.middleware[name]

	return mw, ok
}

// ServeHTTP fulfills the net/http's Handler interface.
func (d Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.handler.ServeHTTP(w, r)
}

// Handle registers the provided pattern controller.
func (d *Dispatcher) Handle(c PatternController) {
	d.Controllers = append(d.Controllers, c)

	var routes []Route

	r := Route{
		c.Pattern(), c.Method(), c.Handler(d.Context), c.Name(), c,
	}

	routes = append(routes, r)

	for _, r := range routes {
		if err := d.trie.AddRoute(r); err != nil {
			panic(err)
		}
	}
}

// Handle registers the provided multi-pattern controller.
func (d *Dispatcher) HandleMultiPattern(c MultiPatternController) {
	d.Controllers = append(d.Controllers, c)

	var routes []Route

	for pattern, tuple := range c.Patterns() {
		r := Route{
			pattern, tuple.Method, c.Handler(d.Context), "", c,
		}

		routes = append(routes, r)
	}

	for _, r := range routes {
		if err := d.trie.AddRoute(r); err != nil {
			panic(err)
		}
	}
}

// NameToPath returns a url path, mapped to the given route name. A method
// should be specified to further narrow down the search. If more than one
// methods are matched, the first path is returned. Finally, an optional
// RouteParams object may be given, to replace any path parameters with their
// values in the given map. The empty string is returned if no route is found
// for the given name. The root dispatcher pattern is always prepended to any
// found path.
func (d Dispatcher) NameToPath(name string, method Method, params ...RouteParams) string {
	if match, ok := d.trie.LookupNamed(name, method, params...); ok {
		for _, v := range match.ReverseURL {
			if d.Pattern != "/" {
				return d.Pattern[:len(d.Pattern)-1] + v
			}
			return v
		}

	}

	return ""
}

// RequestRoute returns the route object and params associated with the
// supplied request.
func (d Dispatcher) RequestRoute(r *http.Request) (Route, RouteParams, bool) {
	path := strings.SplitN(r.RequestURI, "?", 2)[0]
	if path == "" {
		path = r.URL.RequestURI()
	}
	if d.Pattern != "/" {
		path = path[len(d.Pattern)-1:]
	}
	method := ReverseMethodNames[r.Method]
	match, matchFound := d.trie.Lookup(path, method)

	if matchFound {
		r, ok := match.RouteMap[method]

		return r, match.Params, ok
	} else {
		return Route{}, RouteParams{}, matchFound
	}
}

// Initialize creates all configured middleware handlers, producing a chain
// of functions to be called on each request. This function is called
// automatically by the Server object, when its ListenAndServe method is
// called. It should be called directly before calling http.Handle() using
// the dispatcher when the Server object is not used.
func (d *Dispatcher) Initialize() {
	if d.Renderer == nil {
		d.Renderer = renderer.NewRenderer(d.Config.Renderer.Dir, d.Config.Renderer.Base)
	}

	d.Context.SetGlobal(context.BaseCtxKey("renderer"), d.Renderer)
	d.Context.SetGlobal(context.BaseCtxKey("logger"), d.Logger)
	d.Context.SetGlobal(context.BaseCtxKey("dispatcher"), d)
	d.Context.SetGlobal(context.BaseCtxKey("config"), d.Config)

	var mw []Middleware
	order := []string{}
	middlewareInserted := make(map[string]bool)

	for _, m := range d.Config.Dispatcher.Middleware {
		if custom, ok := d.middleware[m]; ok {
			mw = append(mw, custom)
			order = append(order, m)

			middlewareInserted[m] = true
		}
	}

	reverseOrder := d.middlewareOrder[:]
	sort.Sort(sort.Reverse(sort.StringSlice(reverseOrder)))
	for _, name := range reverseOrder {
		if !middlewareInserted[name] {
			if custom, ok := d.middleware[name]; ok {
				mw = append([]Middleware{custom}, mw...)
				order = append([]string{name}, order...)
			}
		}
	}

	handler := d.handlerFunc()

	for _, m := range mw {
		handler = m.Handler(handler, d.Context)
	}

	d.handler = handler
	d.middlewareOrder = order
}

func (d Dispatcher) handlerFunc() http.Handler {
	var handler func(w http.ResponseWriter, r *http.Request)

	handler = func(w http.ResponseWriter, r *http.Request) {
		var route Route
		routeFound := false

		method := ReverseMethodNames[r.Method]
		if GetNamedForward(d.Context, r) != "" {
			match, ok := d.trie.LookupNamed(GetNamedForward(d.Context, r), method)
			d.Context.Delete(r, context.BaseCtxKey("named-forward"))

			if ok {
				route, routeFound = match.RouteMap[method]
			}
		} else if GetForward(d.Context, r) != "" {
			path := GetForward(d.Context, r)

			d.Context.Delete(r, context.BaseCtxKey("forward"))

			if d.Pattern != "/" {
				path = path[len(d.Pattern)-1:]
			}
			d.Context.Delete(r, context.BaseCtxKey("params"))
			if match, ok := d.trie.Lookup(path, method); ok {
				route, routeFound = match.RouteMap[method]
				d.Context.Set(r, context.BaseCtxKey("params"), match.Params)
			}
		} else {
			var params RouteParams
			route, params, routeFound = d.RequestRoute(r)

			if routeFound {
				d.Context.Set(r, context.BaseCtxKey("params"), params)

				switch tc := route.Controller.(type) {
				case MultiPatternController:
					if tuple, ok := tc.Patterns()[route.Pattern]; ok {
						d.Context.Set(r, context.BaseCtxKey("multi-pattern-identifier"), tuple.Identifier)
					}
				}
			}
		}

		d.Context.Set(r, context.BaseCtxKey("r"), r)

		if routeFound {
			d.Context.Set(r, context.BaseCtxKey("route-name"), route.Name)
			route.Handler.ServeHTTP(w, r)

			if GetForward(d.Context, r) != "" || GetNamedForward(d.Context, r) != "" {
				handler(w, r)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)

			err := GetRenderCtx(d.Context, r)(w, nil, "404.tmpl")
			if err != nil {
				fmt.Print(err)
			}
		}
	}

	return http.HandlerFunc(handler)
}
