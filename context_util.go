package webfw

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/middleware"
	"github.com/urandom/webfw/renderer"
	"github.com/urandom/webfw/types"
	"github.com/urandom/webfw/util"
)

// GetConfig is a helper function for getting the current config
// from the request context.
func GetConfig(c types.Context, r *http.Request) Config {
	if val, ok := c.Get(r, types.BaseCtxKey("config")); ok {
		return val.(Config)
	}

	return Config{}
}

// GetParams returns the current request path parameters from the context.
func GetParams(c types.Context, r *http.Request) types.RouteParams {
	if val, ok := c.Get(r, types.BaseCtxKey("params")); ok {
		return val.(types.RouteParams)
	}

	return types.RouteParams{}
}

// GetSession returns the current session from the context,
// if the Session middleware is in use.
func GetSession(c types.Context, r *http.Request) types.Session {
	if val, ok := c.Get(r, types.BaseCtxKey("session")); ok {
		return val.(types.Session)
	}

	conf := GetConfig(c, r)
	var abspath string

	if filepath.IsAbs(conf.Session.Dir) {
		abspath = conf.Session.Dir
	} else {
		var err error
		abspath, err = filepath.Abs(path.Join(filepath.Dir(os.Args[0]), conf.Session.Dir))

		if err != nil {
			abspath = os.TempDir()
		}
	}

	sess := context.NewSession([]byte(conf.Session.Secret), abspath)
	sess.SetName(util.UUID())
	return sess
}

// GetLanguage returns the current request language, such as "en", or "bg-BG"
// from the context, if the I18N middleware is in use.
func GetLanguage(c types.Context, r *http.Request) string {
	if val, ok := c.Get(r, types.BaseCtxKey("lang")); ok {
		return val.(string)
	}

	return middleware.FallbackLocale(c, r)
}

// GetRenderer returns the current raw renderer from the context.
func GetRenderer(c types.Context, r *http.Request) *renderer.Renderer {
	if val, ok := c.Get(r, types.BaseCtxKey("renderer")); ok {
		return val.(*renderer.Renderer)
	}

	return renderer.NewRenderer("template", "base.tmpl")
}

// GetRenderCtx returns a RenderCtx wrapper around the current raw renderer
// The wrapper automatically adds the current request ContextData to the
// renderer's Render method call.
func GetRenderCtx(c types.Context, r *http.Request) types.RenderCtx {
	rnd := GetRenderer(c, r)

	return types.RenderCtx(func(w io.Writer, data types.RenderData, names ...string) error {
		return rnd.Render(w, data, c.GetAll(r), names...)
	})
}

// GetLogger returns the error logger, to be used if an error occurs during
// a request.
func GetLogger(c types.Context, r *http.Request) *log.Logger {
	if val, ok := c.Get(r, types.BaseCtxKey("logger")); ok {
		return val.(*log.Logger)
	}

	return log.New(os.Stderr, "", 0)
}
