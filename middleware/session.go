package middleware

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"
)

/*
The Session middleware is responsible for initializing the persistent user
session for each request. Most of its configuration is passed to the
underlying session object.

The middleware may be configured using the server configuration. The "dir",
"secret", "max-age" and "cleanup-max-age" are passed to the session object
itself. "max-age", "cleanup-interval" and "cleanup-max-age" use the
time.Duration string format. The "cleanup-interval" setting specifies a
time.Ticker duration. On each tick, any file system session data will be
removed, if its older than "cleanup-max-age". If the later setting is empty,
all session data will be deleted.
*/
type Session struct {
	Path            string
	Secret          []byte
	MaxAge          string
	CleanupInterval string
	CleanupMaxAge   string
}

func (smw Session) Handler(ph http.Handler, c context.Context, l *log.Logger) http.Handler {
	var abspath string
	var maxAge, cleanupInterval, cleanupMaxAge time.Duration

	if filepath.IsAbs(smw.Path) {
		abspath = smw.Path
	} else {
		var err error
		abspath, err = filepath.Abs(path.Join(filepath.Dir(os.Args[0]), smw.Path))

		if err != nil {
			panic(err)
		}
	}

	if smw.MaxAge != "" {
		var err error
		maxAge, err = time.ParseDuration(smw.MaxAge)

		if err != nil {
			panic(err)
		}
	}

	if smw.CleanupInterval != "" {
		var err error
		cleanupInterval, err = time.ParseDuration(smw.CleanupInterval)

		if err != nil {
			panic(err)
		}

		cleanupMaxAge, err = time.ParseDuration(smw.CleanupMaxAge)

		if err != nil {
			panic(err)
		}

		go func() {
			for _ = range time.Tick(cleanupInterval) {
				l.Print("Cleaning up old sessions")

				if err := context.CleanupSessions(abspath, cleanupMaxAge); err != nil {
					l.Printf("Failed to clean up sessions: %v", err)
				}
			}
		}()
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		firstTimer := false
		sess := context.NewSession(smw.Secret, abspath)
		sess.SetMaxAge(maxAge)

		err := sess.Read(r, c)

		if err != nil && err != context.ErrExpired && err != context.ErrNotExist {
			sess.SetName(util.UUID())
			firstTimer = true

			if err != context.ErrCookieNotExist {
				l.Printf("Error reading session: %v", err)
			}
		}

		c.Set(r, context.BaseCtxKey("session"), sess)
		c.Set(r, context.BaseCtxKey("firstTimer"), firstTimer)

		rec := httptest.NewRecorder()

		ph.ServeHTTP(rec, r)

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		if sess != nil {
			if err := sess.Write(w); err != nil {
				l.Printf("Unable to write session: %v", err)
			}
		}

		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	}

	return http.HandlerFunc(handler)
}
