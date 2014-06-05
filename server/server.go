package server

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/nitrous-io/ferry/proxy"
)

type Server struct{}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if isWebSocket(req) {
		wsp := &proxy.WebSocketReverseProxy{}
		wsp.ServeHTTP(w, req)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Leave request unmodified.
		},
	}
	proxy.ServeHTTP(w, req)
}

func isWebSocket(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(req.Header.Get("Connection"), "upgrade")
}

func NewServer() *Server {
	return &Server{}
}
