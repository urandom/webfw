package context

import (
	"net/http"
	"testing"
)

func TestContext(t *testing.T) {
	c := NewContext()

	r, _ := http.NewRequest("GET", "http://localhost:8080", nil)

	if _, ok := c.Get(r, "test1"); ok {
		t.Error()
	}

	c.Set(r, "test1", "data1")

	if d, ok := c.Get(r, "test1"); ok {
		if sd, ok := d.(string); ok {
			if sd != "data1" {
				t.Error()
			}
		} else {
			t.Error()
		}
	} else {
		t.Error()
	}

	c.Set(r, "test2", map[string]string{"foo": "bar"})

	if d, ok := c.Get(r, "test2"); ok {
		if sd, ok := d.(map[string]string); ok {
			if v, ok := sd["foo"]; ok {
				if v != "bar" {
					t.Error()
				}
			} else {
				t.Error()
			}
		} else {
			t.Error()
		}
	} else {
		t.Error()
	}

	r2, _ := http.NewRequest("GET", "http://localhost:8080", nil)

	if _, ok := c.Get(r2, "test1"); ok {
		t.Error()
	}

	c.Set(r2, "test1", "data2")

	if d, ok := c.Get(r2, "test1"); ok {
		if sd, ok := d.(string); ok {
			if sd != "data2" {
				t.Error()
			}
		} else {
			t.Error()
		}
	} else {
		t.Error()
	}

	c.DeleteAll(r)

	if _, ok := c.Get(r, "test1"); ok {
		t.Error()
	}

	if _, ok := c.Get(r2, "test1"); !ok {
		t.Error()
	}

	c.Set(r, "test1", "data1")
	c.Set(r, "test2", "data2")

	if _, ok := c.Get(r, "test1"); !ok {
		t.Error()
	}

	if _, ok := c.Get(r, "test2"); !ok {
		t.Error()
	}

	c.Delete(r, "test1")
	if _, ok := c.Get(r, "test1"); ok {
		t.Error()
	}

	if _, ok := c.Get(r, "test2"); !ok {
		t.Error()
	}
}
