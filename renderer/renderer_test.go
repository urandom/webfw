package renderer

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"webfw/context"
	"webfw/types"
)

func TestRenderer(t *testing.T) {
	r := NewRenderer("testdata", "test.tmpl")
	cd := context.ContextData{}

	buf := new(bytes.Buffer)

	if err := r.Render(buf, nil, cd); err != nil {
		t.Fatal(err)
	}

	res := strings.TrimSpace(buf.String())
	expected := ""
	if res != expected {
		t.Fatalf("Expected result to be '%s', got '%s'\n", expected, res)
	}

	buf.Reset()
	if err := r.Render(buf, nil, cd, "test_normal.tmpl"); err != nil {
		t.Fatal(err)
	}

	res = strings.TrimSpace(buf.String())

	expected = `[content: test1]

[ctx: ]
[base: ]`
	if res != expected {
		t.Fatalf("Expected '%s', got '%s'\n", expected, res)
	}

	buf.Reset()
	if err := r.Render(buf, nil, cd, "test_inner.tmpl", "test_normal.tmpl"); err != nil {
		t.Fatal(err)
	}

	res = strings.TrimSpace(buf.String())

	expected = `[content: test1]

[inner: test3]

[ctx: ]
[base: ]`
	if res != expected {
		t.Fatalf("Expected '%s', got '%s'\n", expected, res)
	}

	buf.Reset()
	cd[types.BaseCtxKey("test")] = "foo"
	cd["test"] = "bar"
	if err := r.Render(buf, nil, cd, "test_inner.tmpl", "test_normal.tmpl"); err != nil {
		t.Fatal(err)
	}

	res = strings.TrimSpace(buf.String())

	expected = `[content: test1]

[inner: test3]

[ctx: bar]
[base: foo]`
	if res != expected {
		t.Fatalf("Expected '%s', got '%s'\n", expected, res)
	}

	if err := r.Funcs(template.FuncMap{
		"foo": func(dot string) string {
			return strings.ToUpper(dot)
		},
	}); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	data := types.RenderData{"test": "stuff"}
	if err := r.Render(buf, data, cd, "test_inner.tmpl", "test_func.tmpl"); err != nil {
		t.Fatal(err)
	}

	res = strings.TrimSpace(buf.String())

	expected = `STUFF`
	if res != expected {
		t.Fatalf("Expected '%s', got '%s'\n", expected, res)
	}
}
