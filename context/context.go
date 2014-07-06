package context

import (
	"net/http"
	"sync"
	"time"

	"github.com/urandom/webfw/types"
)

type Context struct {
	mutex    sync.RWMutex
	data     map[*http.Request]types.ContextData
	lifespan map[*http.Request]int64
}

// NewContext creates a new Context object.
func NewContext() *Context {
	return &Context{
		data:     make(map[*http.Request]types.ContextData),
		lifespan: make(map[*http.Request]int64),
	}
}

// Set binds a key-value pair for a given request in the context.
func (c *Context) Set(r *http.Request, key interface{}, val interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.data[r] == nil {
		c.data[r] = make(types.ContextData)
		c.lifespan[r] = time.Now().Unix()
	}
	c.data[r][key] = val
}

// Get returns a value for a given key, bound to a request.
func (c *Context) Get(r *http.Request, key interface{}) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if data, ok := c.data[r]; ok {
		val, ok := data[key]
		return val, ok
	}
	return nil, false
}

// GetAll returns all ContextData for a given request.
func (c *Context) GetAll(r *http.Request) types.ContextData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if data, ok := c.data[r]; ok {
		return data
	}
	return nil
}

// DeleteAll removes all context data for a request.
func (c *Context) DeleteAll(r *http.Request) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, r)
	delete(c.lifespan, r)
}

// Delete removes a key-value pair bound to a request.
func (c *Context) Delete(r *http.Request, key interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data[r], key)
}

// Cleanup cleans any ContextData older than a given age.
func (c *Context) Cleanup(age time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if age <= 0 {
		c.data = make(map[*http.Request]types.ContextData)
		c.lifespan = make(map[*http.Request]int64)
	} else {
		min := time.Now().Add(-age).Unix()
		for r := range c.data {
			if c.lifespan[r] < min {
				delete(c.data, r)
				delete(c.lifespan, r)
			}
		}
	}
}
