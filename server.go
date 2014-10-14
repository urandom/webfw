// +build go1.3

package webfw

import (
	"fmt"
	"net/http"
)

// Server is a helper object to handle dispatchers through the standard
// http.ListenAndServe and http.Handle interface. The address and port can
// be set through the configuration
type Server struct {
	Config  Config
	Address string
	Port    int

	dispatchers map[string]*Dispatcher
}

// NewServer creates a server with an optional path to a
// configuration file.
func NewServer(confpath ...string) Server {
	conf, err := ReadConfig(confpath...)

	if err != nil {
		panic(err)
	}

	return Server{
		Config:  conf,
		Address: conf.Server.Address,
		Port:    conf.Server.Port,

		dispatchers: make(map[string]*Dispatcher),
	}
}

// Dispatcher returns a dispatcher registered for a given base pattern.
// If no dispatcher exists for the given pattern, a new one is created.
func (s *Server) Dispatcher(pattern string) *Dispatcher {
	if d, ok := s.dispatchers[pattern]; ok {
		return d
	}

	d := NewDispatcher(pattern, s.Config)
	s.dispatchers[pattern] = &d

	return &d
}

// ListenAndServe initializes the registered dispatchers and calls the
// http.ListenAndServe function for the current server configuration
func (s Server) ListenAndServe() error {
	for p, d := range s.dispatchers {
		d.Initialize()
		http.Handle(d.Host+p, d)
	}

	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)
	if s.Config.Server.CertFile != "" && s.Config.Server.KeyFile != "" {
		return http.ListenAndServeTLS(
			addr,
			s.Config.Server.CertFile,
			s.Config.Server.KeyFile,
			nil,
		)
	} else {
		return http.ListenAndServe(addr, nil)
	}
}
