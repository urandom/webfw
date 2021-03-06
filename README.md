webfw
=====

A simple collection of things for writing web stuff.

Docs and examples are avaiable at [godoc](http://godoc.org/github.com/urandom/webfw)

A quick example, straight from the docs (the later is always up-to-date):

1. The code
    ```go
    package main
    
    import (
    	"net/http"
    
    	"github.com/urandom/webfw"
    	"github.com/urandom/webfw/context"
    )
    
    type Hello struct {
            webfw.BasePatternController
    }
    
    func NewHello(pattern string) Hello {
            return Hello{webfw.NewBasePatternController(pattern, MethodAll, "")}
    }
    
    func (con Hello) Handler(c *context.Context) http.HandlerFunc {
            return func(w http.ResponseWriter, r *http.Request) {
                    params := webfw.GetParams(c, r)
                    d := renderer.RenderData{"name": params["name"]}

                    err := webfw.GetRenderCtx(c, r)(w, d, "hello.tmpl")
                    if err != nil {
                            webfw.GetLogger(c, r).Print(err)
                    }
            }
    }

    
    func Example() {
    	s := webfw.NewServer()
    
    	dispatcher := s.Dispatcher("/")
    
    	dispatcher.Handle(NewHello("/hello/:name"))
    	if err := s.ListenAndServe(); err != nil {
    		panic(err)
    	}
    }
    
    func main() {
    	Example()
    }
    ```
2. The templates, in a directory "templates"

    2.1 "base.tmpl":
    ```html
    <!doctype html>
    <html>
        <body>{{ template "content" . }}</body>
    </html>
    {{ define "content" }}{{ end }}
    ```
    2.2 "hello.tmpl"
    ```html
    {{ define "content" }}
    <h1>Hello {{ .name }}</h1>
    {{ end }}
    ```
