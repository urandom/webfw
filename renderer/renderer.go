// Copyright 2011 Viktor Kojouharov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Package renderer provides a helper type for rendering html/template files.
// There is always a 'base' template, which acts as an initial template onto
// which functions may be registered and used in the subsequent template chain.
// Each template chain always ends with the 'base' template, and is cached after
// being parsed.
package renderer

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/fs"
	"github.com/urandom/webfw/util"
)

type Renderer interface {
	BaseName() string
	SetBaseName(string)
	Funcs(template.FuncMap)
	Delims(left, right string)
	SkipCache(skip ...bool) bool
	Render(w io.Writer, data RenderData, cdata context.ContextData, names ...string) error
}

type renderer struct {
	mutex      sync.RWMutex
	tmpls      map[string]*template.Template
	path       string
	baseName   string
	leftDelim  string
	rightDelim string
	skipCache  bool
	funcMaps   []template.FuncMap
}

type RenderCtx func(w io.Writer, data RenderData, names ...string) error

type RenderData map[string]interface{}

// NewRenderer creates a new renderer object. The path points to a directory
// containing the template files. The base is the name of the file for the
// 'base' template.
func NewRenderer(path, base string) Renderer {
	return &renderer{
		tmpls:    make(map[string]*template.Template),
		path:     path,
		baseName: base,
	}
}

// BaseName returns the current base template name
func (r *renderer) BaseName() string {
	return r.baseName
}

// SetBaseName sets a new template base name
func (r *renderer) SetBaseName(base string) {
	r.baseName = base
}

// Sets the action delimeters for all future templates
func (r *renderer) Delims(left, right string) {
	r.leftDelim = left
	r.rightDelim = right
}

// Funcs registers a template.FuncMap object all future templates
func (r *renderer) Funcs(funcMap template.FuncMap) {
	r.funcMaps = append(r.funcMaps, funcMap)
}

func (r *renderer) SkipCache(skip ...bool) bool {
	if len(skip) > 0 {
		r.skipCache = skip[0]
	}

	return r.skipCache
}

// Render executes the template chain, writing the output in the given
// io.Writer. The data is the user data, passed to the template upon execution.
// The context data will also be added the the template data. All context data
// using string keys will be added to the template data under the 'ctx' key.
// All framework data will be added to the template data under the 'base' key.
// This data may contain:
//   - "lang", the current language
//   - "langs", all supported languages
//   - "r", the current request
//   - "params", the current route params
//   - "config", the framework configuration
//   - "session", the session
//   - "logger", the error logger
//   - "firstTimer", if the session is newly created
// The list of data is partially dependant on the middleware chain
func (r *renderer) Render(w io.Writer, data RenderData, cdata context.ContextData, names ...string) error {
	var tmpl *template.Template

	if len(names) == 0 {
		if t, err := r.base(); err != nil {
			return err
		} else {
			var err error

			if tmpl, err = t.Clone(); err != nil {
				return err
			}
		}
	} else {
		var ok bool
		key := strings.Join(names, "-")
		tmpl, ok = r.get(key)

		if !ok {
			t, err := r.base()
			if err != nil {
				return err
			}

			t, err = t.Clone()
			if err != nil {
				return err
			}

			t, err = parseFiles(t, r.path, names...)

			if err != nil {
				return err
			}

			r.set(key, t)
			tmpl = t
		}
	}

	buf := util.BufferPool.GetBuffer()

	if data == nil {
		data = RenderData{}
	}

	baseData := RenderData{}

	data["base"] = baseData

	contextData := RenderData{}

	data["ctx"] = contextData

	baseData["template"] = tmpl
	for k, v := range cdata {
		switch t := k.(type) {
		case string:
			contextData[t] = v
		case context.BaseCtxKey:
			baseData[string(t)] = v
		}
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return err
	}

	buf.WriteTo(w)

	util.BufferPool.Put(buf)

	return nil
}

func (r *renderer) get(name string) (*template.Template, bool) {
	if r.skipCache {
		return nil, false
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	t, ok := r.tmpls[name]

	return t, ok
}

func (r *renderer) set(name string, t *template.Template) {
	if r.skipCache {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tmpls[name] = t
}

func (r *renderer) base() (*template.Template, error) {
	if !r.skipCache {
		r.mutex.RLock()

		if t, ok := r.tmpls[r.baseName]; ok {
			r.mutex.RUnlock()
			return t, nil
		}

		r.mutex.RUnlock()
	}
	return r.create(r.baseName)
}

func (r *renderer) create(name string) (*template.Template, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	t := template.New(name)
	/* Add any .Funcs before calling .Parse */

	if r.leftDelim != "" && r.rightDelim != "" {
		t.Delims(r.leftDelim, r.rightDelim)
	}

	for _, fm := range r.funcMaps {
		t.Funcs(fm)
	}

	if t, err := parseFiles(t, r.path, name); err == nil {
		if !r.skipCache {
			r.tmpls[name] = t
		}
	} else {
		return nil, err
	}

	return t, nil
}

func parseFiles(t *template.Template, root string, names ...string) (*template.Template, error) {
	for _, name := range names {
		var tmpl *template.Template
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}

		if file, err := fs.DefaultFS.OpenRoot(root, name); err == nil {
			if b, err := ioutil.ReadAll(file); err == nil {
				tmpl.Parse(string(b))
			} else {
				return nil, errors.New(fmt.Sprintf("Error reading '%s' with root '%s': %v\n", name, root, err))
			}
		} else {
			return nil, errors.New(fmt.Sprintf("Error opening '%s' with root '%s': %v\n", name, root, err))
		}
	}

	return t, nil
}
