package context

import (
	"net/http"
	"sync"
	"time"
)

type Context interface {
	Get(*http.Request, interface{}) (interface{}, bool)
	GetAll(*http.Request) ContextData
	Set(*http.Request, interface{}, interface{})
	GetGlobal(interface{}) (interface{}, bool)
	SetGlobal(interface{}, interface{})
	DeleteAll(*http.Request)
	Delete(*http.Request, interface{})
}

type ContextData map[interface{}]interface{}
type BaseCtxKey string

type context struct {
	mutex    sync.RWMutex
	data     map[*http.Request]ContextData
	global   ContextData
	lifespan map[*http.Request]int64
}

// NewContext creates a new Context object.
func NewContext() Context {
	return &context{
		data:     make(map[*http.Request]ContextData),
		global:   make(ContextData),
		lifespan: make(map[*http.Request]int64),
	}
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

// GetGlobal returns a value for a given key, not bound to a request.
func (c *context) GetGlobal(key interface{}) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if val, ok := c.global[key]; ok {
		return val, ok
	}
	return nil, false
}

// SetGlobal sets a global key-value pair, not bound to a request.
func (c *context) SetGlobal(key interface{}, val interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.global[key] = val
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
