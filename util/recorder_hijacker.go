package util

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
)

type RecorderHijacker interface {
	http.ResponseWriter
	http.Hijacker

	GetCode() int
	GetBody() *bytes.Buffer

	getHijacker() http.Hijacker
}

type responseHijacker struct {
	*httptest.ResponseRecorder
	w http.ResponseWriter
}

func NewRecorderHijacker(w http.ResponseWriter) RecorderHijacker {
	return &responseHijacker{
		ResponseRecorder: httptest.NewRecorder(),
		w:                w,
	}
}

func (rh *responseHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj := rh.getHijacker()
	if hj == nil {
		return nil, nil, errors.New("Original ResponseWriter is not a Hijacker")
	}

	return hj.Hijack()
}

func (rh *responseHijacker) GetCode() int {
	return rh.ResponseRecorder.Code
}

func (rh *responseHijacker) GetBody() *bytes.Buffer {
	return rh.ResponseRecorder.Body
}

func (rh *responseHijacker) getHijacker() http.Hijacker {
	switch t := rh.w.(type) {
	case http.Hijacker:
		return t
	case RecorderHijacker:
		return t.getHijacker()
	default:
		return nil
	}
}
