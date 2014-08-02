package middleware

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"
)

type Sitemap struct {
	Pattern          string
	Prefix           string
	RelativeLocation string
	Controllers      []webfw.SitemapController
}

var sitemapTmpl *template.Template

func init() {
	sitemapTmpl = template.Must(template.New("sitemap").Parse(xmlTemplate))
}

func (mw Sitemap) Handler(ph http.Handler, c context.Context, l *log.Logger) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		loc := mw.RelativeLocation
		if loc == "" {
			loc = "sitemap.xml"
		}

		uriParts := strings.SplitN(r.RequestURI, "?", 2)
		if uriParts[0] == "" {
			uriParts[0] = r.URL.Path
		}
		if uriParts[0] == mw.Pattern+loc {
			prefix := mw.Prefix
			if lang, ok := c.Get(r, context.BaseCtxKey("lang")); ok {
				if l, ok := lang.(string); ok && l != "" {
					prefix = prefix + l
				}
			}

			if strings.HasSuffix(prefix, "/") {
				prefix = prefix[:len(prefix)-1]
			}

			urls := []map[string]string{}

			for _, con := range mw.Controllers {
				sm := con.Sitemap(c)
				for _, s := range sm {
					m := map[string]string{"loc": prefix + s.Loc}

					if s.LastMod != webfw.SitemapNoLastMod {
						m["lastmod"] = s.LastMod.Format("2006-01-02")
					}

					if s.ChangeFreq != webfw.SitemapNoFrequency {
						m["changefreq"] = string(s.ChangeFreq)
					}

					if s.Priority != webfw.SitemapNoPriority {
						m["priority"] = strconv.FormatFloat(s.Priority, 'g', 2, 64)
					}

					urls = append(urls, m)
				}
			}

			buf := util.BufferPool.GetBuffer()
			defer util.BufferPool.Put(buf)

			if err := sitemapTmpl.Execute(buf, urls); err == nil {
				if _, err := buf.WriteTo(w); err == nil {
					return
				} else {
					l.Printf("Error serving sitemap template: %v\n", err)
				}
			} else {
				l.Printf("Error executing sitemap template: %v\n", err)
			}
		}
		ph.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}

const xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	{{ range . }}
		<url>
			<loc>{{ .loc }}</loc>
			{{ if .lastmod }}
			<lastmod>{{ .lastmod }}</lastmod>
			{{ end }}
			{{ if .changefreq }}
			<changefreq>{{ .changefreq }}</changefreq>
			{{ end }}
			{{ if .priority }}
			<priority>{{ .priority }}</priority>
			{{ end }}
		</url>
   {{ end }}
</urlset>
`
