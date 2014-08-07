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
	Context     context.Context
	Config      Config
	Logger      *log.Logger
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

// ServeHTTP fulfills the net/http's Handler interface.
func (d Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.handler.ServeHTTP(w, r)
}

// Handle registers the provided controller.
func (d *Dispatcher) Handle(c Controller) {
	r := Route{
		c.Pattern(), c.Method(), c.Handler(d.Context), c.Name(), c,
	}

	d.Controllers = append(d.Controllers, c)
	if err := d.trie.AddRoute(r); err != nil {
		panic(err)
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

// SitemapControllers returns a slice of all SitemapController instances
// currently handled by this dispatcher
func (d Dispatcher) SitemapControllers() []SitemapController {
	var controllers []SitemapController
	for _, c := range d.Controllers {
		if sc, ok := c.(SitemapController); ok {
			controllers = append(controllers, sc)
		}
	}

	return controllers
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
		handler = m.Handler(handler, d.Context, d.Logger)
	}

	d.handler = handler
	d.middlewareOrder = order
}

func (d Dispatcher) handlerFunc() http.Handler {
	var handler func(w http.ResponseWriter, r *http.Request)

	handler = func(w http.ResponseWriter, r *http.Request) {
		var match Match
		matchFound, namedMatch := false, false

		method := ReverseMethodNames[r.Method]
		if GetNamedForward(d.Context, r) != "" {
			namedMatch = true
			match, matchFound = d.trie.LookupNamed(GetNamedForward(d.Context, r), method)
			d.Context.Delete(r, context.BaseCtxKey("named-forward"))
		} else {
			path := GetForward(d.Context, r)

			if path == "" {
				path = strings.SplitN(r.RequestURI, "?", 2)[0]
				if path == "" {
					path = r.URL.RequestURI()
				}
			} else {
				d.Context.Delete(r, context.BaseCtxKey("forward"))
			}

			if d.Pattern != "/" {
				path = path[len(d.Pattern)-1:]
			}
			d.Context.Delete(r, context.BaseCtxKey("params"))
			match, matchFound = d.trie.Lookup(path, method)
		}

		d.Context.Set(r, context.BaseCtxKey("r"), r)

		if matchFound {
			if !namedMatch {
				d.Context.Set(r, context.BaseCtxKey("params"), match.Params)
			}

			route := match.RouteMap[method]

			d.Context.Set(r, context.BaseCtxKey("route-name"), route.Name)
			d.Context.Set(r, context.BaseCtxKey("controller"), route.Controller)
			route.Handler(w, r)

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
