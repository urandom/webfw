package webfw

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/middleware"
	"github.com/urandom/webfw/renderer"
)

// Dispatchers are responsible for calling controller handlers for a
// particular request path, passing it through middleware chain for
// each request. The middleware order can be configured via the server
// configuration. Multiple dispatchers may be created, and each one is
// defined by a ServerMux root pattern. The pattern must end in a '/'.
type Dispatcher struct {
	trie            *Trie
	pattern         string
	handler         http.Handler
	context         context.Context
	logger          *log.Logger
	config          Config
	renderer        *renderer.Renderer
	middleware      map[string]Middleware
	middlewareOrder []string
}

// NewDispatcher creates a dispatcher for the given base pattern and config.
func NewDispatcher(pattern string, c Config) *Dispatcher {
	if !strings.HasSuffix(pattern, "/") {
		panic("The dispatcher pattern has to end with a '/'")
	}

	d := &Dispatcher{
		trie:       NewTrie(),
		config:     c,
		pattern:    pattern,
		context:    context.NewContext(),
		logger:     log.New(os.Stderr, "", 0),
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

	d.middleware[name] = mw
	d.middlewareOrder = append(d.middlewareOrder, name)
}

// SetContext allows for setting a different context implementation. This
// should ideally be called before registering any controller, so that any
// controller handler method is called with the new context.
func (d *Dispatcher) SetContext(context context.Context) {
	d.context = context
}

// ServeHTTP fulfills the net/http's Handler interface.
func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.handler.ServeHTTP(w, r)
}

// Handle registers the provided controller.
func (d *Dispatcher) Handle(c Controller) {
	r := &Route{
		c.Pattern(), c.Method(), c.Handler(d.context), c.Name(),
	}

	if err := d.trie.AddRoute(r); err != nil {
		panic(err)
	}
}

func (d *Dispatcher) handlerFunc() http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.RequestURI()
		method := ReverseMethodNames[r.Method]

		if d.pattern != "/" {
			path = path[len(d.pattern)-1:]
		}

		d.context.Set(r, context.BaseCtxKey("r"), r)
		d.context.Set(r, context.BaseCtxKey("renderer"), d.renderer)
		d.context.Set(r, context.BaseCtxKey("logger"), d.logger)

		if match, ok := d.trie.Lookup(path, method); ok {
			d.context.Set(r, context.BaseCtxKey("params"), match.params)

			route := match.routes[method]
			route.Handler(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)

			err := GetRenderCtx(d.context, r)(w, nil, "404.tmpl")
			if err != nil {
				fmt.Print(err)
			}
		}
	}

	return http.HandlerFunc(handler)
}

func (d *Dispatcher) init() {
	d.renderer = renderer.NewRenderer(d.config.Renderer.Dir, d.config.Renderer.Base)

	var mw []Middleware
	order := []string{}
	middlewareInserted := make(map[string]bool)

	for _, m := range d.config.Dispatcher.Middleware {
		if custom, ok := d.middleware[m]; ok {
			mw = append(mw, custom)
			order = append(order, m)

			middlewareInserted[m] = true
		} else {
			switch m {
			case "Error":
				mw = append(mw, middleware.Error{ShowStack: d.config.Server.Devel})
				order = append(order, m)
			case "Context":
				mw = append(mw, middleware.Context{})
				order = append(order, m)
			case "Logger":
				mw = append(mw, middleware.Logger{AccessLogger: log.New(os.Stdout, "", 0)})
				order = append(order, m)
			case "Gzip":
				mw = append(mw, middleware.Gzip{})
				order = append(order, m)
			case "Static":
				mw = append(mw, middleware.Static{
					FileList: d.config.Static.FileList || d.config.Server.Devel,
					Path:     d.config.Static.Dir,
					Expires:  d.config.Static.Expires,
					Prefix:   d.config.Static.Prefix,
					Index:    d.config.Static.Index,
				})
				order = append(order, m)
			case "Session":
				var cipher []byte
				if d.config.Session.Cipher != "" {
					var err error
					if cipher, err = base64.StdEncoding.DecodeString(d.config.Session.Cipher); err != nil {
						panic(err)
					}
				}
				mw = append(mw, middleware.Session{
					Path:            d.config.Session.Dir,
					Secret:          []byte(d.config.Session.Secret),
					Cipher:          cipher,
					MaxAge:          d.config.Session.MaxAge,
					CleanupInterval: d.config.Session.CleanupInterval,
					CleanupMaxAge:   d.config.Session.CleanupMaxAge,
				})
				order = append(order, m)
			case "I18N":
				mw = append(mw, middleware.I18N{
					Dir:             d.config.I18n.Dir,
					Pattern:         d.pattern,
					Languages:       d.config.I18n.Languages,
					Renderer:        d.renderer,
					IgnoreURLPrefix: d.config.I18n.IgnoreURLPrefix,
				})
				order = append(order, m)
			case "Url":
				mw = append(mw, middleware.Url{
					Renderer: d.renderer,
				})
				order = append(order, m)
			}
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
		handler = m.Handler(handler, d.context, d.logger)
	}

	handler = func(ph http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			d.context.Set(r, context.BaseCtxKey("config"), d.config)

			ph.ServeHTTP(w, r)
		}

		return http.HandlerFunc(handler)
	}(handler)

	d.handler = handler
	d.middlewareOrder = order
}
