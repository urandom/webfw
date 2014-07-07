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
	"html/template"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"
)

type Renderer struct {
	mutex    sync.RWMutex
	tmpls    map[string]*template.Template
	path     string
	baseName string
}

type RenderCtx func(w io.Writer, data RenderData, names ...string) error

type RenderData map[string]interface{}

// NewRenderer creates a new Renderer object. The path points to a directory
// containing the template files. The base is the name of the file for the
// 'base' template.
func NewRenderer(path, base string) *Renderer {
	return &Renderer{
		tmpls:    make(map[string]*template.Template),
		path:     path,
		baseName: base,
	}
}

// Funcs registers a template.FuncMap object for the 'base' template.
func (r *Renderer) Funcs(funcMap template.FuncMap) error {
	t, err := r.base()
	if err != nil {
		return err
	}

	t.Funcs(funcMap)

	return nil
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
func (r *Renderer) Render(w io.Writer, data RenderData, cdata context.ContextData, names ...string) error {
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

			relativePaths := []string{}
			for _, name := range names {
				relativePaths = append(relativePaths, path.Join(r.path, name))
			}

			t, err = t.Clone()
			if err != nil {
				return err
			}

			t, err = t.ParseFiles(relativePaths...)
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
		switch k.(type) {
		case string:
			contextData[k.(string)] = v
		case context.BaseCtxKey:
			baseData[string(k.(context.BaseCtxKey))] = v
		}
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return err
	}

	buf.WriteTo(w)

	util.BufferPool.Put(buf)

	return nil
}

func (r *Renderer) get(name string) (*template.Template, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	t, ok := r.tmpls[name]

	return t, ok
}

func (r *Renderer) set(name string, t *template.Template) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tmpls[name] = t
}

func (r *Renderer) base() (*template.Template, error) {
	r.mutex.RLock()

	if t, ok := r.tmpls[r.baseName]; ok {
		r.mutex.RUnlock()
		return t, nil
	}

	r.mutex.RUnlock()
	return r.create(r.baseName)
}

func (r *Renderer) create(name string) (*template.Template, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	/* Add any .Funcs before calling .Parse */
	t, err := template.New(name).ParseFiles(path.Join(r.path, name))

	if err != nil {
		return nil, err
	}

	r.tmpls[name] = t

	return t, nil
}
