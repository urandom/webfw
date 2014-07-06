package types

import "net/http"

type Context interface {
	Set(*http.Request, interface{}, interface{})
	Get(*http.Request, interface{}) (interface{}, bool)
	GetAll(*http.Request) ContextData
	DeleteAll(*http.Request)
	Delete(*http.Request, interface{})
}

type ContextData map[interface{}]interface{}

type BaseCtxKey string
