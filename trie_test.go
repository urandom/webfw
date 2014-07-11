package webfw

import "testing"

func TestAddRoute(t *testing.T) {
	trie := NewTrie()

	if trie.root == nil {
		t.Fatal()
	}

	if trie.named != nil {
		t.Fatal()
	}

	if trie.AddRoute(Route{Pattern: "/", Method: MethodGet}) != nil {
		t.Fatal()
	}

	if trie.root.children["/"] == nil {
		t.Fatal()
	}

	trie.AddRoute(Route{Pattern: "/1/2/3", Method: MethodGet})

	if trie.root.children["/"].children["1"].children["/"].children["2"].children["/"].children["3"] == nil {
		t.Fatal()
	}

	if trie.root.children["/"].nodeType != normal {
		t.Fatal()
	}

	trie = NewTrie()

	trie.AddRoute(Route{Pattern: "/1 /*2", Method: MethodGet})
	n := trie.root.children["/"].children["1"].children["%"].children["2"].children["0"].children["/"].children["2"]
	if n == nil {
		t.Fatal()
	}
	if n.nodeType != glob {
		t.Fatal()
	}
}

func TestAddRouteParam(t *testing.T) {
	trie := NewTrie()

	trie.AddRoute(Route{Pattern: "/f/:param1/t:param2", Method: MethodGet})

	if trie.root.children["/"].children["f"].children["/"].children["param1"].
		children["/"].children["t"].children["param2"] == nil {
		t.Fatal()
	}

	n := trie.root.children["/"].children["f"].children["/"].children["param1"]

	if n.nodeType != param {
		t.Fatal()
	}

	if n.param != "param1" {
		t.Fatal()
	}

	n = n.children["/"].children["t"].children["param2"]
	if n.nodeType != param {
		t.Fatal()
	}

	if n.param != "param2" {
		t.Fatal()
	}

	if err := trie.AddRoute(Route{Pattern: "/f/*error", Method: MethodGet}); err == nil {
		t.Fatal(err)
	}
}

func TestAddRouteGlob(t *testing.T) {
	trie := NewTrie()

	if trie.AddRoute(Route{Pattern: "/f/*param1/test/:fakeparam2", Method: MethodGet}) != nil {
		t.Fatal()
	}

	n := trie.root.children["/"].children["f"].children["/"].children["param1/test/:fakeparam2"]
	if n == nil {
		t.Fatal()
	}

	if n.nodeType != glob {
		t.Fatal()
	}

	if n.children != nil {
		t.Fatal()
	}
}

func TestAddRouteNamed(t *testing.T) {
	trie := NewTrie()

	trie.AddRoute(Route{Pattern: "/f/:param1/t:param2", Method: MethodGet, Name: "test1"})

	if trie.named == nil {
		t.Fatal()
	}

	if nodes, ok := trie.named[MethodGet]; ok {
		if n, ok := nodes["test1"]; ok {
			if n.nodeType != param {
				t.Fatal()
			}

			if n.param != "param2" {
				t.Fatal()
			}

			if n.children != nil {
				t.Fatal()
			}

			if len(n.routes) != 1 {
				t.Fatal()
			}

			if r, ok := n.routes[MethodGet]; ok {
				if r.Pattern != "/f/:param1/t:param2" {
					t.Fatal()
				}
			} else {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	if trie.AddRoute(Route{Pattern: "bla", Method: MethodGet, Name: "test1"}) == nil {
		t.Fatal()
	}
}

func TestAddRouteMethods(t *testing.T) {
	trie := NewTrie()

	if trie.AddRoute(Route{Pattern: "/f", Method: MethodGet | MethodPost, Name: "test1"}) != nil {
		t.Fatal()
	}

	if trie.AddRoute(Route{Pattern: "/f", Method: MethodPut | MethodDelete, Name: "test2"}) != nil {
		t.Fatal()
	}

	n := trie.root.children["/"].children["f"]
	if len(n.routes) != 4 {
		t.Fatal()
	}

	if n.routes[MethodGet].Name != "test1" {
		t.Fatal()
	}

	if n.routes[MethodPost].Name != "test1" {
		t.Fatal()
	}

	if n.routes[MethodPut].Name != "test2" {
		t.Fatal()
	}

	if n.routes[MethodDelete].Name != "test2" {
		t.Fatal()
	}

	if trie.AddRoute(Route{Pattern: "/f", Method: MethodGet | MethodDelete, Name: "test3"}) == nil {
		t.Fatal()
	}

	if trie.AddRoute(Route{Pattern: "/f", Method: MethodAll, Name: "test3"}) == nil {
		t.Fatal()
	}
}

func TestLookupNamed(t *testing.T) {
	trie := NewTrie()

	trie.AddRoute(Route{Pattern: "/f", Method: MethodGet | MethodPost, Name: "test1"})
	trie.AddRoute(Route{Pattern: "/f", Method: MethodPut | MethodDelete, Name: "test2"})
	trie.AddRoute(Route{Pattern: "/b", Method: MethodAll, Name: "test3"})

	if m, ok := trie.LookupNamed("test1", MethodAll); ok {
		if len(m.RouteMap) != 2 {
			t.Fatal()
		}

		if r, ok := m.RouteMap[MethodGet]; ok {
			if r.Pattern != "/f" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}

		if r, ok := m.RouteMap[MethodPost]; ok {
			if r.Pattern != "/f" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	if m, ok := trie.LookupNamed("test2", MethodAll); ok {
		if len(m.RouteMap) != 2 {
			t.Fatal()
		}

		if r, ok := m.RouteMap[MethodPut]; ok {
			if r.Pattern != "/f" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}

		if r, ok := m.RouteMap[MethodDelete]; ok {
			if r.Pattern != "/f" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}

	if m, ok := trie.LookupNamed("test3", MethodDelete); ok {
		if len(m.RouteMap) != 1 {
			t.Fatal()
		}

		if r, ok := m.RouteMap[MethodDelete]; ok {
			if r.Pattern != "/b" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestLookup(t *testing.T) {
	trie := NewTrie()
	trie.AddRoute(Route{Pattern: "/f/:param1/t:param2", Method: MethodGet})

	if match, ok := trie.Lookup("/f/hello/tWorld", MethodAll); ok {
		if len(match.RouteMap) != 1 {
			t.Fatal()
		}

		if r, ok := match.RouteMap[MethodGet]; ok {
			if r.Pattern != "/f/:param1/t:param2" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}

		if match.Params == nil {
			t.Fatal()
		} else if len(match.Params) != 2 {
			t.Fatal()
		} else {
			if match.Params["param1"] != "hello" {
				t.Fatal()
			}

			if match.Params["param2"] != "World" {
				t.Fatal()
			}
		}
	} else {
		t.Fatal()
	}

	if err := trie.AddRoute(Route{Pattern: "/f/:param1/*glob/conti:nuing", Method: MethodGet}); err != nil {
		t.Fatal(err)
	}

	if match, ok := trie.Lookup("/f/hello/tWorld", MethodAll); ok {
		if len(match.RouteMap) != 1 {
			t.Fatal()
		}

		if r, ok := match.RouteMap[MethodGet]; ok {
			if r.Pattern != "/f/:param1/*glob/conti:nuing" {
				t.Fatal()
			}
		} else {
			t.Fatal()
		}

		if match.Params == nil {
			t.Fatal()
		} else if len(match.Params) != 2 {
			t.Fatal()
		} else {
			if match.Params["param1"] != "hello" {
				t.Fatal()
			}

			if match.Params["glob/conti:nuing"] != "tWorld" {
				t.Fatal()
			}
		}
	} else {
		t.Fatal()
	}

	trie.AddRoute(Route{Pattern: "/", Method: MethodGet})

	if match, ok := trie.Lookup("/f/hello/tWorld", MethodAll); ok {
		if len(match.RouteMap) != 1 {
			t.Fatal()
		}

		if match.Params == nil {
			t.Fatal()
		} else if len(match.Params) != 2 {
			t.Fatal()
		} else {
			if match.Params["param1"] != "hello" {
				t.Fatal()
			}

			if match.Params["glob/conti:nuing"] != "tWorld" {
				t.Fatal()
			}
		}
	} else {
		t.Fatal()
	}
}
