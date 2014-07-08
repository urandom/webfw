// +build go1.3
package webfw

import (
	"fmt"
	"net/http"
)

// Server is a helper object to handle dispatchers through the standard
// http.ListenAndServe and http.Handle interface. The host and port can
// be set through the configuration
type Server struct {
	dispatchers map[string]*Dispatcher
	conf        Config
	host        string
	port        int
}

// NewServer creates a server with an optional path to a
// configuration file.
func NewServer(confpath ...string) *Server {
	conf, err := ReadConfig(confpath...)

	if err != nil {
		panic(err)
	}

	return &Server{
		dispatchers: make(map[string]*Dispatcher),
		conf:        conf,
		host:        conf.Server.Host,
		port:        conf.Server.Port,
	}
}

// SetHost sets the host for the server.
func (s *Server) SetHost(host string) {
	s.host = host
}

// SetPort sets the port for the server
func (s *Server) SetPort(port int) {
	s.port = port
}

// Dispatcher returns a dispatcher registered for a given base pattern.
// If no dispatcher exists for the given pattern, a new one is created.
func (s *Server) Dispatcher(pattern string) *Dispatcher {
	if d, ok := s.dispatchers[pattern]; ok {
		return d
	}

	d := NewDispatcher(pattern, s.conf)
	s.dispatchers[pattern] = d

	return d
}

// ListenAndServe initializes the registered dispatchers and calls the
// http.ListenAndServe function for the current server configuration
func (s *Server) ListenAndServe() error {
	for p, d := range s.dispatchers {
		d.init()
		http.Handle(p, d)
	}

	if s.conf.Server.CertFile != "" && s.conf.Server.KeyFile != "" {
		return http.ListenAndServeTLS(
			fmt.Sprintf("%s:%d", s.host, s.port),
			s.conf.Server.CertFile,
			s.conf.Server.KeyFile,
			nil,
		)
	} else {
		return http.ListenAndServe(fmt.Sprintf("%s:%d", s.host, s.port), nil)
	}
}
