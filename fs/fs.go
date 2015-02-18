package fs

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FS struct {
	sync.RWMutex
	root listing
}

type listing struct {
	files       map[string]listing
	fileListing *FileListing
}

var (
	DefaultFS = NewFS()
)

func NewFS() *FS {
	return &FS{root: listing{files: make(map[string]listing),
		fileListing: NewFileListing("/", 0, os.ModeDir, time.Now(), []byte{})}}
}

func (fs *FS) Add(fl *FileListing) bool {
	fs.Lock()
	defer fs.Unlock()

	if d, ok := fs.mkdirP(fl.dir); ok {
		st, err := fl.Stat()
		if err != nil {
			return false
		}

		d.files[st.Name()] = listing{fileListing: fl}
		fl.fs = fs
		return true
	} else {
		return false
	}
}

func (fs *FS) Open(name string) (f http.File, err error) {
	fs.RLock()
	defer fs.RUnlock()

	name = filepath.ToSlash(name)

	if path.IsAbs(name) {
		name = path.Clean(name)
	} else {
		name = path.Clean("/" + name)[1:]
	}

	if l, ok := fs.getListing(name); ok {
		return l.fileListing.Open(), nil
	} else {
		f, err = os.Open(filepath.FromSlash(name))
	}

	return
}

func (fs *FS) OpenRoot(root, name string) (http.File, error) {
	if root == "" {
		root = "."
	}

	if !filepath.IsAbs(root) {
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
	}

	return fs.Open(strings.Join([]string{root, path.Clean("/" + name)[1:]}, "/"))
}

func (fs *FS) mkdirP(name string) (listing, bool) {
	parts := strings.Split(name, "/")
	if parts[0] == "" {
		parts = parts[1:]
	}
	l := fs.root

	for i, p := range parts {
		st, err := l.fileListing.Stat()
		if err != nil {
			break
		}

		if st.IsDir() {
			if c, ok := l.files[p]; ok {
				if st, err := c.fileListing.Stat(); err != nil || !st.IsDir() {
					break
				}
				l = c
			} else {
				c = listing{
					files: make(map[string]listing),
					fileListing: NewFileListing(
						strings.Join(parts[:i+1], "/"), 0, os.ModeDir, time.Now(), []byte{}),
				}
				l.files[p] = c
				l = c
			}

			if len(parts)-1 == i {
				return l, true
			}
		} else {
			break
		}
	}

	return fs.root, false
}

func (fs *FS) getListing(name string) (listing, bool) {
	parts := strings.Split(name, "/")
	if parts[0] == "" {
		parts = parts[1:]
	}
	l := fs.root

	for i, p := range parts {
		st, err := l.fileListing.Stat()
		if err != nil {
			break
		}

		if st.IsDir() {
			var ok bool
			if l, ok = l.files[p]; ok {
				if len(parts)-1 == i {
					return l, true
				}
			} else {
				break
			}
		} else {
			break
		}
	}

	return fs.root, false
}
