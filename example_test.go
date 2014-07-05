package webfw_test

import (
	"net/http"
	"webfw"
	"webfw/context"
	"webfw/controller"
	"webfw/types"
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
