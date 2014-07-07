package context

import (
	"net/http"
	"sync"
	"time"
)

type Context interface {
	Set(*http.Request, interface{}, interface{})
	Get(*http.Request, interface{}) (interface{}, bool)
	GetAll(*http.Request) ContextData
	DeleteAll(*http.Request)
	Delete(*http.Request, interface{})
}

type ContextData map[interface{}]interface{}
type BaseCtxKey string

type context struct {
	mutex    sync.RWMutex
	data     map[*http.Request]ContextData
	lifespan map[*http.Request]int64
}

// NewContext creates a new Context object.
func NewContext() Context {
	return &context{
		data:     make(map[*http.Request]ContextData),
		lifespan: make(map[*http.Request]int64),
	}
}

// Set binds a key-value pair for a given request in the context.
func (c *context) Set(r *http.Request, key interface{}, val interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.data[r] == nil {
		c.data[r] = make(ContextData)
		c.lifespan[r] = time.Now().Unix()
	}
	c.data[r][key] = val
}

// Get returns a value for a given key, bound to a request.
func (c *context) Get(r *http.Request, key interface{}) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if data, ok := c.data[r]; ok {
		val, ok := data[key]
		return val, ok
	}
	return nil, false
}

// GetAll returns all ContextData for a given request.
func (c *context) GetAll(r *http.Request) ContextData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if data, ok := c.data[r]; ok {
		return data
	}
	return nil
}

// DeleteAll removes all context data for a request.
func (c *context) DeleteAll(r *http.Request) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, r)
	delete(c.lifespan, r)
}

// Delete removes a key-value pair bound to a request.
func (c *context) Delete(r *http.Request, key interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data[r], key)
}

// Cleanup cleans any ContextData older than a given age.
func (c *context) Cleanup(age time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if age <= 0 {
		c.data = make(map[*http.Request]ContextData)
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
