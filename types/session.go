package types

import (
	"errors"
	"net/http"
	"time"
)

var (
	ErrExpired        = errors.New("Session expired")
	ErrNotExist       = errors.New("Session does not exist")
	ErrCookieNotExist = errors.New("Session cookie does not exist")
)

type Session interface {
	Read(*http.Request, Context) error
	Write(http.ResponseWriter) error
	Name() string
	SetName(string)
	MaxAge() time.Duration
	SetMaxAge(time.Duration)
	CookieName() string
	SetCookieName(string)

	Set(interface{}, interface{})
	Get(interface{}) (interface{}, bool)
	GetAll() SessionValues
	DeleteAll()
	Delete(interface{})
	Flash(interface{}) interface{}
	SetFlash(interface{}, interface{})
}

type SessionValues map[interface{}]interface{}
type FlashValues map[interface{}]interface{}
