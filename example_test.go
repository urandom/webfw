package webfw_test

import (
	"net/http"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/controller"
	"github.com/urandom/webfw/types"
)

type Hello struct {
	controller.Base
}

func NewHello(pattern string) Hello {
	return Hello{controller.New(pattern, types.MethodAll, "")}
}

func (con Hello) Handler(c *context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := webfw.GetParams(c, r)
		d := types.RenderData{"name": params["name"]}

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
