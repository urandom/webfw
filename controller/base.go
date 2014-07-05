// Copyright 2011 Viktor Kojouharov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Package controller provides a partial implementation for a controller
// interface. It may be useful for embedding within real controller
// implementations.
package controller

import "webfw/types"

type Base struct {
	pattern string
	method  types.Method
	name    string
}

// New creates a base controller.
func New(pattern string, method types.Method, name string) Base {
	return Base{pattern: pattern, method: method, name: name}
}

// Pattern returns the pattern associated with the controller. It may contain
// named parameters and globs. For example:
//  - "/hello/:first/:last" will fill RouteParams with whatever the url
//    contains in place of :first under the key "first", and likewise for
//    "last". A parameter starts with the ':' character, and ends with a
//    '/'
//  - "/hello/*name" will fill RouteParams with a glob under the key "name".
//    The value will be everything that occurs in place of '*name' until
//    the end of the url path.
func (b Base) Pattern() string {
	return b.pattern
}

// Method returns the method(s) for the controller. Since the Method constants
// are bitmasks, a controller may handle more than one method at a time.
func (b Base) Method() types.Method {
	return b.method
}

// Name returns the name a controller may be referred to.
func (b Base) Name() string {
	return b.name
}
