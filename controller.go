package webfw

import (
	"net/http"

	"github.com/urandom/webfw/context"
)

// In general, a controller is a simple wrapper around an http.HandlerFunc.
// The Handler method recieves the context, which may then be either stored
// within the implementing type, or closed over when creating the HandlerFunc.
type Controller interface {
	Handler(context.Context) http.Handler
}

type PatternController interface {
	Controller
	Pattern() string

	Method() Method
	Name() string
}

// A MultiPatternController may be used in places where a single controller
// handles more than one patterns. The Patterns method should return a map,
// whose keys are the patterns to be matched, and the values are a tuple of
// Method and some pattern identifier, the later of which will be stored in the
// context. The regular Pattern, Method and Name methods will not be called.
type MultiPatternController interface {
	Controller
	Patterns() map[string]MethodIdentifierTuple
}

type MethodIdentifierTuple struct {
	Method     Method
	Identifier string
}

type BasePatternController struct {
	pattern string
	method  Method
	name    string
}

// New creates a base controller.
func NewBasePatternController(pattern string, method Method, name string) BasePatternController {
	return BasePatternController{pattern: pattern, method: method, name: name}
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
func (b BasePatternController) Pattern() string {
	return b.pattern
}

// Method returns the method(s) for the controller. Since the Method constants
// are bitmasks, a controller may handle more than one method at a time.
func (b BasePatternController) Method() Method {
	return b.method
}

// Name returns the name a controller may be referred to.
func (b BasePatternController) Name() string {
	return b.name
}
