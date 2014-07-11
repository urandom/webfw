package webfw

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Trie is an object that stores routes in a prefix tree, and allows for
// an efficient lookup afterwards. The route paths may contain named route
// parameters and globs. The parameters are defined as ":key" inside the
// route path, where a piece of a subsequent request path will be extracted
// where such a parameter occurs. The value will be the text under a '/' or
// the end of the request path. A glob is similar to a parameter. It can be
// defined with "*key", the leading '*' being the different between it and
// parameters, and it will assing the rest of the request path as the glob
// value.
type Trie struct {
	root  *node
	named map[Method]map[string]*node
}

type node struct {
	routes   map[Method]Route
	nodeType nodeType
	param    string
	children map[string]*node
}

type Match struct {
	routes map[Method]Route
	params RouteParams
}

type nodeType int

const (
	normal = iota
	param
	glob
)

var methods []Method = []Method{MethodGet, MethodPost, MethodPut, MethodDelete}

// NewTrie creates a new Route trie, for efficient lookup of routes and names.
func NewTrie() *Trie {
	return &Trie{
		root: &node{},
	}
}

// AddRoute adds the given route to the trie.
func (t *Trie) AddRoute(route Route) error {
	if route.Name != "" && t.named != nil {
		for _, method := range methods {
			if route.Method&method > 0 {
				if _, ok := t.named[method]; ok {
					if _, ok := t.named[method][route.Name]; ok {
						return errors.New(fmt.Sprintf("Cannot add route '%s', another with the name '%s' is already added!", route.Pattern, route.Name))
					}
				}
			}
		}
	}
	urlObj, err := url.Parse(route.Pattern)
	if err != nil {
		return err
	}

	pattern := strings.Replace(urlObj.RequestURI(), "%2A", "*", -1)

	n, err := t.root.add(pattern, route, []string{}, t)

	if err != nil {
		return err
	}

	for _, method := range methods {
		if route.Method&method > 0 {
			if route.Name != "" {
				if t.named == nil {
					t.named = map[Method]map[string]*node{}
				}
				if t.named[method] == nil {
					t.named[method] = map[string]*node{}
				}
				t.named[method][route.Name] = n
			}
		}
	}

	return nil
}

// Lookup searches for routes registered for the given path and method, and
// returns then in the form of a Match object
func (t *Trie) Lookup(path string, method Method) (Match, bool) {
	match := Match{}

	node, params, found := t.root.lookup(path, RouteParams{})
	if found {
		for key, val := range node.routes {
			if method&key > 0 {
				if match.routes == nil {
					match.routes = map[Method]Route{}
				}
				match.routes[key] = val
			}
		}
		if match.routes == nil {
			found = false
		} else {
			match.params = params
		}
	}
	return match, found
}

// LookupNamed searches for routes registered under the given name
func (t *Trie) LookupNamed(name string, method Method) (Match, bool) {
	match, found := Match{}, false
	for _, m := range methods {
		if method&m > 0 {
			if names, ok := t.named[m]; ok {
				if node, ok := names[name]; ok {
					found = true
					if match.routes == nil {
						match.routes = map[Method]Route{}
					}
					match.routes[m] = node.routes[m]
				}
			}
		}
	}
	return match, found
}

func (n *node) add(term string, route Route, params []string, t *Trie) (*node, error) {
	if term == "" {
		for _, method := range methods {
			if route.Method&method > 0 {
				err := n.addRouteForMethod(route, method)
				if err != nil {
					return nil, err
				}
			}
		}
		return n, nil
	} else {
		head, tail := term[:1], term[1:]
		var child *node

		if tail != "" && (head == ":" || head == "*") {
			var paramName string
			var nodeType nodeType
			if head == ":" {
				paramName, tail = split(tail)
			} else {
				paramName, tail = tail, ""
			}

			for _, p := range params {
				if p == paramName {
					return nil, errors.New(fmt.Sprintf("Found a duplicate param '%s' along the route '%s'!", p, route.Pattern))
				}
			}

			params = append(params, paramName)

			if head == ":" {
				nodeType = param
			} else {
				nodeType = glob
			}

			if n.children == nil {
				n.children = map[string]*node{}
			} else {
				for _, val := range n.children {
					if (val.nodeType == param || val.nodeType == glob) &&
						(val.nodeType != nodeType || val.param != paramName) {
						return nil, errors.New("Found a conflicting route which contains a parameter in the same position!")
					}
				}
			}

			if _, ok := n.children[paramName]; !ok {
				n.children[paramName] = &node{}
			}

			child = n.children[paramName]
			child.param = paramName
			child.nodeType = nodeType
		} else {
			if n.children == nil {
				n.children = map[string]*node{}
			}

			if _, ok := n.children[head]; !ok {
				n.children[head] = &node{}
			}

			child = n.children[head]
			child.nodeType = normal
		}

		return child.add(tail, route, params, t)
	}
}

func (n *node) addRouteForMethod(route Route, method Method) error {
	if n.routes == nil {
		n.routes = map[Method]Route{}
	}

	if _, ok := n.routes[method]; ok {
		return errors.New(
			fmt.Sprintf("A route for the same pattern '%s' and method '%s' already exists!",
				route.Pattern, MethodNames[method]),
		)
	}
	n.routes[method] = route

	return nil
}

func (n *node) lookup(term string, params RouteParams) (*node, RouteParams, bool) {
	if term == "" {
		return n, params, true
	}

	if n.children == nil {
		return nil, params, false
	}

	head, tail := term[:1], term[1:]

	/* Reverse lookup */
	if tail != "" && (head == ":" || head == ":") {
		if head == ":" {
			head, tail = split(tail)
		} else {
			head, tail = tail, ""
		}

		if child, ok := n.children[head]; ok {
			return child.lookup(tail, params)
		} else {
			return nil, params, false
		}
	}

	for _, child := range n.children {
		if child.nodeType == param || child.nodeType == glob {
			if child.nodeType == param {
				head, tail = split(head + tail)
			} else {
				head, tail = head+tail, ""
			}

			params[child.param] = head

			return child.lookup(tail, params)
		}
	}

	if child, ok := n.children[head]; ok {
		return child.lookup(tail, params)
	}
	return nil, params, false
}

func split(tail string) (string, string) {
	i := 0
	for i < len(tail) && tail[i] != '/' {
		i++
	}

	return tail[:i], tail[i:]
}
