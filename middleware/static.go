package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"time"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"

	"sort"
	"strings"
)

/*
The Static middleware serves static files, and optionally provides directory
listings. Any parent handler which sets the response code to
http.StatusNotFound will cause the middleware to try and handle the request
as a static file. Furthermore, only "HEAD" and "GET" methods will be handled.

The following server configuration variables may be set:
    - "dir" specifies the directory from where to serve static files
    - "prefix" is a request path prefix, which may be used to limit the
      static handling for only those requests
    - "index" specifies the filename to try and serve if the request
      is for a directory
    - "expires" specifies a time.Duration string format, which will be
      used to set the Expires and Cache-Control header fields. If left
      empty, those fields will not be set.
    - "file-list" is a boolean flag, which will cause the middleware to
      show the directory listing if the request is for a directory, and
      it doesn't contain an index file.
*/
type Static struct {
	Path     string
	Prefix   string
	Index    string
	Expires  string
	FileList bool
}

var staticTmpl *template.Template

type FileStats []os.FileInfo

type fileList struct {
	CurDir string
	Stats  FileStats
}

func (fs FileStats) Len() int           { return len(fs) }
func (fs FileStats) Swap(i, j int)      { fs[i], fs[j] = fs[j], fs[i] }
func (fs FileStats) Less(i, j int) bool { return fs[i].Name() < fs[j].Name() }

func init() {
	staticTmpl = template.Must(template.New("filelist").Funcs(template.FuncMap{
		"formatdate": func(t time.Time) string {
			return t.Format(dateFormat)
		},
	}).Parse(fileListTemplate))
}

func (smw Static) Handler(ph http.Handler, c context.Context) http.Handler {
	var expires time.Duration

	root := http.Dir(smw.Path)

	if smw.Expires != "" {
		var err error
		expires, err = time.ParseDuration(smw.Expires)

		if err != nil {
			panic(err)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()

		ph.ServeHTTP(rec, r)

		if rec.Code != http.StatusNotFound {
			copyRecorder(rec, w)

			return
		}

		uriParts := strings.SplitN(r.RequestURI, "?", 2)
		if uriParts[0] == "" {
			uriParts[0] = r.URL.Path
		}
		for {
			if r.Method != "GET" && r.Method != "HEAD" {
				break
			}
			rpath := uriParts[0]

			if smw.Prefix != "" {
				if !strings.HasPrefix(rpath, smw.Prefix) {
					break
				}

				rpath = rpath[len(smw.Prefix):]
				if rpath != "" && rpath[0] != '/' {
					break
				}
			}

			file, err := root.Open(rpath)
			if err != nil {
				break
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				break
			}

			if stat.IsDir() {
				if !strings.HasSuffix(uriParts[0], "/") {
					http.Redirect(w, r, uriParts[0]+"/", http.StatusFound)
					return
				}

				index := "index.html"
				if smw.Index != "" {
					index = smw.Index
				}

				ipath := path.Join(rpath, index)

				file, err = root.Open(ipath)
				if err == nil {
					defer file.Close()
					stat, err = file.Stat()
				}

				if err != nil || stat.IsDir() {
					if smw.FileList {
						file, err = root.Open(rpath)
						if err != nil {
							break
						}

						stats, err := file.Readdir(1000)
						if err != nil {
							break
						}

						sort.Sort(FileStats(stats))
						fileList := &fileList{CurDir: path.Base(rpath), Stats: stats}

						buf := util.BufferPool.GetBuffer()
						defer util.BufferPool.Put(buf)

						if err := staticTmpl.Execute(buf, fileList); err != nil {
							break
						}

						if _, err := buf.WriteTo(w); err != nil {
							break
						}

						return
					} else {
						break
					}
				} else {
					rpath = ipath
				}
			}

			etag := generateEtag(rpath, stat)

			w.Header().Set("ETag", etag)

			if expires != 0 {
				w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%.0f", expires.Seconds()))
				w.Header().Set("Expires", time.Now().Add(expires).Format(http.TimeFormat))
			}

			http.ServeContent(w, r, rpath, stat.ModTime(), file)
			return
		}

		copyRecorder(rec, w)
	}

	return http.HandlerFunc(handler)
}

func generateEtag(rpath string, stat os.FileInfo) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", rpath, stat.ModTime().Unix())))

	return base64.URLEncoding.EncodeToString(hash[:])
}

func copyRecorder(rec *httptest.ResponseRecorder, w http.ResponseWriter) {
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(rec.Code)
	w.Write(rec.Body.Bytes())
}

const fileListTemplate = `
<!doctype html>
<html>
	<head>
		<title>{{ .CurDir }}</title>
	</head>
	<body>
		<table>
			<tbody>
				<tr>
					<td><a href="../">../</a></td>
					<td colspan="2"></td>
				</tr>
				{{ range .Stats }}
					<tr>
						<td>
							{{ if .IsDir }}
								<a href="{{ .Name }}/">{{ .Name }}/</a>
							{{ else }}
								<a href="{{ .Name }}">{{ .Name }}</a>
							{{ end }}
						</td>
						<td>
							{{ .ModTime | formatdate }}
						</td>
						<td>
							{{ .Size }}
						</td>
					</tr>
				{{ end }}
	</body>
</html>
`
