package middleware

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"

	"github.com/urandom/webfw"
)

// InitializeDefault creates all default middleware objects by the order
// of the dispatcher configuration, and registers them to the later.
func InitializeDefault(d *webfw.Dispatcher) {
	for _, m := range d.Config.Dispatcher.Middleware {
		switch m {
		case "Error":
			d.RegisterMiddleware(Error{ShowStack: d.Config.Server.Devel})
		case "Context":
			d.RegisterMiddleware(Context{})
		case "Logger":
			d.RegisterMiddleware(Logger{AccessLogger: webfw.NewStandardLogger(os.Stdout, "", 0)})
		case "Gzip":
			d.RegisterMiddleware(Gzip{})
		case "Static":
			d.RegisterMiddleware(Static{
				FileList: d.Config.Static.FileList || d.Config.Server.Devel,
				Path:     d.Config.Static.Dir,
				Expires:  d.Config.Static.Expires,
				Prefix:   d.Config.Static.Prefix,
				Index:    d.Config.Static.Index,
			})
		case "Session":
			var cipher []byte
			if d.Config.Session.Cipher != "" {
				var err error
				if cipher, err = base64.StdEncoding.DecodeString(d.Config.Session.Cipher); err != nil {
					panic(err)
				}
			}
			d.RegisterMiddleware(Session{
				Path:            d.Config.Session.Dir,
				Secret:          []byte(d.Config.Session.Secret),
				Cipher:          cipher,
				MaxAge:          d.Config.Session.MaxAge,
				CleanupInterval: d.Config.Session.CleanupInterval,
				CleanupMaxAge:   d.Config.Session.CleanupMaxAge,
				Pattern:         d.Pattern,
				IgnoreURLPrefix: d.Config.Session.IgnoreURLPrefix,
			})
		case "I18N":
			d.RegisterMiddleware(I18N{
				Dir:             d.Config.I18n.Dir,
				Pattern:         d.Pattern,
				Languages:       d.Config.I18n.Languages,
				IgnoreURLPrefix: d.Config.I18n.IgnoreURLPrefix,
			})
		case "Url":
			d.RegisterMiddleware(Url{
				Pattern: d.Pattern,
			})
		case "Sitemap":
			if u, err := url.Parse(d.Config.Sitemap.LocPrefix); err != nil || !u.IsAbs() {
				break
			}

			d.RegisterMiddleware(Sitemap{
				Pattern:          d.Pattern,
				Prefix:           fmt.Sprintf("%s%s", d.Config.Sitemap.LocPrefix, d.Pattern),
				RelativeLocation: d.Config.Sitemap.RelativeLocation,
				Controllers:      d.Controllers,
			})
		}
	}
}
