package middleware

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/renderer"
)

/*
The Url middleware provides helper template functions, registered on the base
renderer template. The following functions are currently provided:

 * "url" - expectes a series of string arguments, which will be used to generate
   a url. A trailing request object must be provided. The first string argument
   will be used as the url path, and the subsequent ones will be joined as a
   query string. If the path argument does not begin with a '/', it will be
   treated as a relative path, and the current request path will be prepended.
   The same goes if the the path argument is the empty string. In both cases,
   if the current request has query params, those will be added to the generated
   url. The string arguments may also contain one or more ':here:' strings, which
   will be replaces with the current RequestURI. Finally, if a previous middleware
   has set the language in the context, it will be added to the url.
   Examples for a request /example/request and language "en":
    - {{ url "" .base.r }} -> /en/example/request
    - {{ url "test" .base.r }} -> /en/example/request/test
    - {{ url "/test" .base.r }} -> /en/test
    - {{ url "/test/:here:" .base.r }} -> /en/test/example/request
    - {{ url "/test" "backto=:here:" .base.r }} ->
         /en/test?backto=/example/request
 * "localizedUrl" - very similar to the "url" function. The only difference is that
   the last string argument will be treated as the language to be used when building
   the url. An example for the same request:
    - {{ localizedUrl "/foo" "de" .base.r }} -> /de/foo
*/
type Url struct {
	Renderer renderer.Renderer
	Pattern  string
}

func (umw Url) Handler(ph http.Handler, c context.Context, l *log.Logger) http.Handler {
	err := umw.Renderer.Funcs(template.FuncMap{
		"url": func(data ...interface{}) (string, error) {
			r, parts, err := handleParts(data)
			if err != nil {
				return "", err
			}

			return URL(c, r, umw.Pattern, parts)
		},
		"localizedUrl": func(data ...interface{}) (string, error) {
			var lang string
			r, parts, err := handleParts(data)
			if err != nil {
				return "", err
			}

			lang, parts = parts[len(parts)-1], parts[:len(parts)-1]

			return LocalizedURL(c, r, umw.Pattern, lang, parts)
		},
	})

	if err != nil {
		panic(err)
	}

	return ph
}

// The URL function provides the functionality of the url template functions
// for use outside of the template context. The dispatcherPattern is the
// pattern used by the dispatcher responsible for handling the resulting url.
// In most cases it will probably be "/".
func URL(c context.Context, r *http.Request, dispatcherPattern string, parts []string) (string, error) {
	base, err := unlocalizedUrl(r, parts)

	if err != nil {
		return "", err
	}

	if lang, ok := c.Get(r, context.BaseCtxKey("lang")); ok && len(lang.(string)) > 0 {
		base = "/" + lang.(string) + base
	}
	if len(dispatcherPattern) > 1 {
		base = dispatcherPattern[:len(dispatcherPattern)-1] + base
	}
	return base, nil
}

// The LocalizedURL function is equivalent to the 'localizedUrl' template
// function.
func LocalizedURL(c context.Context, r *http.Request, dispatcherPattern, language string, parts []string) (string, error) {
	base, err := unlocalizedUrl(r, parts)

	if err != nil {
		return "", err
	}

	base = language + base
	if len(dispatcherPattern) > 1 {
		base = dispatcherPattern + base
	} else {
		base = "/" + base
	}
	return base, nil
}

func handleParts(data []interface{}) (*http.Request, []string, error) {
	if len(data) < 2 {
		return nil, nil, errors.New("Insufficient arguments")
	}

	if r, ok := data[len(data)-1].(*http.Request); ok {
		data = data[:len(data)-1]

		parts := make([]string, len(data))

		for i, d := range data {
			if p, ok := d.(string); ok {
				parts[i] = p
			} else {
				return nil, nil, errors.New("The leading arguments must be strings")
			}
		}

		return r, parts, nil
	} else {
		return nil, nil, errors.New("The last argument must be a request object")
	}
}

func unlocalizedUrl(r *http.Request, parts []string) (string, error) {
	if len(parts) == 0 {
		return "", errors.New("No base url given")
	}

	for i, part := range parts {
		if r.URL.Path[0] == '/' {
			part = strings.Replace(part, "/:here:", r.URL.RequestURI(), -1)
		}
		part = strings.Replace(part, ":here:", r.URL.RequestURI(), -1)

		parts[i] = part
	}

	base := parts[0]
	query := strings.Join(parts[1:], "&")

	if r.URL.RawQuery != "" && (base == "" || base[0] != '/') {
		query = r.URL.RawQuery + "&" + query
		if query[len(query)-1] == '&' {
			query = query[:len(query)-1]
		}
	}

	if base == "" {
		base = r.URL.Path
	} else if base[0] != '/' {
		if r.URL.Path[len(r.URL.Path)-1] == '/' {
			base = r.URL.Path + base
		} else {
			base = r.URL.Path + "/" + base
		}
	}

	if query != "" {
		if strings.Contains(base, "?") {
			base += "&" + query
		} else {
			base += "?" + query
		}
	}

	return base, nil
}
