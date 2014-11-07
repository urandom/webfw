package middleware

import (
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	ttemplate "text/template"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"

	"github.com/nicksnyder/go-i18n/i18n"
)

/*
The I18N middleware is used to provide a translation interface for message
within templates. It registers a "__" function in the 'base' renderer
template, and stores the current language and all configured languages
inside the context, so they can be used within the templates using the
corresponding ".base.lang" and ".base.langs" dot pipelines. It also stores
the current language within the session, if the session middleware is
registered before it.

A Language is set for a request via different means. First, if the relative
request path begins with '/' and the language code, that language will be
used. The dispatcher pattern will not be included in this relative path.
For example, if a dispatcher has the pattern "/", the request path may look
like this:
    - "/en/example/path"
however, if it has a path "/test/", the request will be:
    - "/test/en/example/path"
If the relative request path doesn't contain the language, the language stored
in the session will be used, if one is present. Otherwise, it will fall back
to the first one in the Accept-Language header, then to the LANG environment
variable, tben to the LC_MESSAGES one, and finally to "en". If the request
path doesn't contain a language, the client will be redirected to the same
path, including whatever fallback language is selected. Finally, the request
path is modified so that subsequent middleware and the final handler do not
see the actual language.

Internally, it uses the "github.com/nicksnyder/go-i18n/i18n" package for
the actual translation.

The middleware may be configured via the server configuration. The "dir"
setting specifies the directory that contains  *.all.json files, one per
configured languages. The supported languages may be configured via the
"languages" slice. If a language file doesn't exist, the middleware will
panic early on. In order to ignore requests for a certain prefix, the
"ignore-url-prefix" slice may be defined in the settings.

The template function "__" receives the message id as its first argument,
the language as the second, and any trailing arguments will be interpretted
as key-value tuples to be used for the message. The current language may be
obtained using the ".base.lang" pipeline.
*/
type I18N struct {
	Languages       []string
	Pattern         string
	Dir             string
	IgnoreURLPrefix []string
}

func (imw I18N) Handler(ph http.Handler, c context.Context, l webfw.Logger) http.Handler {
	for _, l := range imw.Languages {
		i18n.MustLoadTranslationFile(filepath.Join(imw.Dir, l+".all.json"))
	}

	renderer := webfw.GetRenderer(c)
	renderer.Funcs(imw.TemplateFuncMap())

	handler := func(w http.ResponseWriter, r *http.Request) {
		c.Set(r, context.BaseCtxKey("langs"), imw.Languages)

		if len(imw.Languages) == 0 {
			c.Set(r, context.BaseCtxKey("lang"), "")
			ph.ServeHTTP(w, r)
			return
		}

		found := false

		uriParts := strings.SplitN(r.RequestURI, "?", 2)
		if uriParts[0] == "" {
			uriParts[0] = r.URL.Path
		}
		for _, prefix := range imw.IgnoreURLPrefix {
			if prefix[0] == '/' {
				prefix = prefix[1:]
			}

			if strings.HasPrefix(uriParts[0], imw.Pattern+prefix+"/") {
				found = true
				break
			}

			if uriParts[0] == imw.Pattern+prefix {
				found = true
				break
			}
		}

		if !found {
			for _, language := range imw.Languages {
				if uriParts[0] == imw.Pattern+language {
					url := uriParts[0] + "/"
					if len(uriParts) > 1 && uriParts[1] != "" {
						url += "?" + uriParts[1]
					}

					http.Redirect(w, r, url, http.StatusFound)

					return
				}

				if strings.HasPrefix(uriParts[0], imw.Pattern+language+"/") {
					r.URL.Path = imw.Pattern + r.URL.Path[len(imw.Pattern+language+"/"):]

					uriParts[0] = imw.Pattern + uriParts[0][len(imw.Pattern+language+"/"):]

					r.RequestURI = strings.Join(uriParts, "?")

					c.Set(r, context.BaseCtxKey("lang"), language)
					found = true

					s := webfw.GetSession(c, r)
					s.Set("language", language)

					break
				}
			}
		}

		if !found {
			fallback := webfw.GetFallbackLanguage(c, r)
			index := strings.Index(fallback, "-")
			short := fallback
			if index > -1 {
				short = fallback[:index]
			}
			foundShort := false

			for _, language := range imw.Languages {
				if language == fallback {
					found = true
					break
				}

				if language == short {
					foundShort = true
				}
			}

			if !found && !foundShort {
				c.Set(r, context.BaseCtxKey("lang"), "")
				ph.ServeHTTP(w, r)
				return
			}

			var language string
			if found {
				language = fallback
			} else {
				language = short
			}

			url := imw.Pattern + language + uriParts[0][len(imw.Pattern)-1:]
			if len(uriParts) > 1 && uriParts[1] != "" {
				url += "?" + uriParts[1]
			}

			http.Redirect(w, r, url, http.StatusFound)

			return
		}

		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}

func (imw I18N) TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"__": func(message, lang string, data ...interface{}) (template.HTML, error) {
			if len(imw.Languages) == 0 {
				return template.HTML(message), nil
			}
			return t(message, lang, data...)
		},
	}
}

func t(message, lang string, data ...interface{}) (template.HTML, error) {
	var count interface{}
	hasCount := false

	if len(data)%2 == 1 {
		if !isNumber(data[0]) {
			return "", errors.New("The count argument must be a number")
		}
		count = data[0]
		hasCount = true

		data = data[1:]
	}

	dataMap := map[string]interface{}{}
	for i := 0; i < len(data); i += 2 {
		dataMap[data[i].(string)] = data[i+1]
	}

	T, err := i18n.Tfunc(lang, "en-US")

	if err != nil {
		return "", err
	}

	var translated string
	if hasCount {
		translated = T(message, count, dataMap)
	} else {
		translated = T(message, dataMap)
	}

	if translated == message {
		// Doesn't have a translation mapping, we have to do the template evaluation by hand
		t, err := ttemplate.New("i18n").Parse(message)

		if err != nil {
			return "", err
		}

		buf := util.BufferPool.GetBuffer()
		defer util.BufferPool.Put(buf)

		if err := t.Execute(buf, dataMap); err != nil {
			return "", err
		}

		return template.HTML(buf.String()), nil
	} else {
		return template.HTML(translated), nil
	}
}

func isNumber(n interface{}) bool {
	switch n.(type) {
	case int, int8, int16, int32, int64, string:
		return true
	}
	return false
}
