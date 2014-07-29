package webfw

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/renderer"
	"github.com/urandom/webfw/util"

	lng "github.com/nicksnyder/go-i18n/i18n/language"
)

// GetDispatcher returns the request dispatcher.
func GetDispatcher(c context.Context) Dispatcher {
	if val, ok := c.GetGlobal(context.BaseCtxKey("dispatcher")); ok {
		return val.(Dispatcher)
	}
	return Dispatcher{}
}

// GetConfig is a helper function for getting the current config
// from the request context.
func GetConfig(c context.Context) Config {
	if val, ok := c.GetGlobal(context.BaseCtxKey("config")); ok {
		return val.(Config)
	}

	return Config{}
}

// GetRenderer returns the current raw renderer from the context.
func GetRenderer(c context.Context) renderer.Renderer {
	if val, ok := c.GetGlobal(context.BaseCtxKey("renderer")); ok {
		return val.(renderer.Renderer)
	}

	return renderer.NewRenderer("template", "base.tmpl")
}

// GetLogger returns the error logger, to be used if an error occurs during
// a request.
func GetLogger(c context.Context) *log.Logger {
	if val, ok := c.GetGlobal(context.BaseCtxKey("logger")); ok {
		return val.(*log.Logger)
	}

	return log.New(os.Stderr, "", 0)
}

// GetParams returns the current request path parameters from the context.
func GetParams(c context.Context, r *http.Request) RouteParams {
	if val, ok := c.Get(r, context.BaseCtxKey("params")); ok {
		return val.(RouteParams)
	}

	return RouteParams{}
}

// GetSession returns the current session from the context,
// if the Session middleware is in use.
func GetSession(c context.Context, r *http.Request) context.Session {
	if val, ok := c.Get(r, context.BaseCtxKey("session")); ok {
		return val.(context.Session)
	}

	conf := GetConfig(c)
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

	sess := context.NewSession([]byte(conf.Session.Secret), []byte(conf.Session.Cipher), abspath)
	sess.SetName(util.UUID())
	return sess
}

// GetLanguage returns the current request language, such as "en", or "bg-BG"
// from the context, if the I18N middleware is in use.
func GetLanguage(c context.Context, r *http.Request) string {
	if val, ok := c.Get(r, context.BaseCtxKey("lang")); ok {
		return val.(string)
	}

	return GetFallbackLanguage(c, r)
}

var localeRegexp = regexp.MustCompile(`\.[\w\-]+$`)

// GetFallbackLanguage tries to obtain a requested language via the session,
// or the Accept-Language request header, or the LANG or LC_MESSAGES
// environment variables
func GetFallbackLanguage(c context.Context, r *http.Request) string {
	if val, ok := c.Get(r, context.BaseCtxKey("session")); ok {
		sess := val.(context.Session)

		if language, ok := sess.Get("language"); ok {
			return language.(string)
		}
	}

	langs := lng.Parse(r.Header.Get("Accept-Language"))

	if len(langs) > 0 {
		return langs[0].String()
	}

	language := os.Getenv("LANG")

	if language == "" {
		language = os.Getenv("LC_MESSAGES")
		language = localeRegexp.ReplaceAllLiteralString(language, "")
	}

	if language == "" {
		language = "en"
	} else {
		langs := lng.Parse(language)
		if len(langs) > 0 {
			return langs[0].String()
		}
	}

	return language
}

// GetRenderCtx returns a RenderCtx wrapper around the current raw renderer
// The wrapper automatically adds the current request ContextData to the
// renderer's Render method call.
func GetRenderCtx(c context.Context, r *http.Request) renderer.RenderCtx {
	rnd := GetRenderer(c)

	return renderer.RenderCtx(func(w io.Writer, data renderer.RenderData, names ...string) error {
		return rnd.Render(w, data, c.GetAll(r), names...)
	})
}

// GetForwards returns a set forward path as a string, or the empty string.
func GetForward(c context.Context, r *http.Request) string {
	if val, ok := c.Get(r, context.BaseCtxKey("forward")); ok {
		return val.(string)
	}
	return ""
}

// GetNamedForward returns a name, used by the dispatcher to lookup a route to forward to.
func GetNamedForward(c context.Context, r *http.Request) string {
	if val, ok := c.Get(r, context.BaseCtxKey("named-forward")); ok {
		return val.(string)
	}
	return ""
}
