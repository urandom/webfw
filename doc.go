// Copyright 2011 Viktor Kojouharov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
/*
Package webfw provides a set of tools for building web applications and
services in go

Some of the provided features are:

    * Dispatchers that provide custom routing with support for path
	  parameters, and act as handlers for go's net/http server
    * Customizable middleware handlers and support for custom ones.
      Some of the included middleware provide:
        - Panic handling
        - GZipping response
        - Static files and directory listing
        - I18N via github.com/nicksnyder/go-i18n/i18n
        - Access logging
        - Context and sessions

    * Configuration via code.google.com/p/gcfg
    * Controllers registered for a particular pattern and method(s),
      which ultimately provide http.HandlerFunc funcions

    * A helper renderer utility that caches html/template chains and
      provides context data for the Dot

Since webfw uses sync.Pool internally, it currently requires go1.3 as
its lowest supported version.
*/
package webfw
