package server

import (
	"net/http"
	"net/http/httputil"
)

type Server struct{}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Leave request unmodified.
		},
	}
	proxy.ServeHTTP(w, req)
}

func NewServer() *Server {
	return &Server{}
}
