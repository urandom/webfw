package webfw_test

import (
	"net/http"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/renderer"
)

type Hello struct {
	webfw.BaseController
}

func NewHello(pattern string) Hello {
	return Hello{webfw.NewBaseController(pattern, MethodAll, "")}
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
