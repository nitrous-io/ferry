package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/nitrous-io/ferry/freighter"
	"github.com/nitrous-io/ferry/proxy"
	"github.com/nitrous-io/ferry/util"
)

type Server struct {
	// BackendDialTimeout specifies the amount of time to wait for a connection
	// to be established to the backend (i.e. container).
	// If 0, a default of 1 minute is used to avoid connection exhaustion.
	BackendDialTimeout time.Duration

	freighterClient freighter.Client
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request received: %+v", req)

	containerName := util.ParseLeftMostLabel(req.Host)
	ip := s.freighterClient.ContainerIP(containerName)
	if ip == nil {
		log.Printf("unmatched request for %v", req.Host)
		http.Error(w, "could not find a backend", http.StatusBadGateway)
		return
	}

	port := util.ParsePort(req)
	tcpAddr := &net.TCPAddr{IP: ip, Port: int(port)}
	log.Printf("Mapped to container IP and port: %v", tcpAddr)

	// Check if the container is still listening.
	// If it's not, refresh list of containers from Freighter.
	conn, err := net.DialTimeout("tcp", tcpAddr.String(), s.BackendDialTimeout)
	if err != nil {
		// TODO Refresh should be done in a goroutine.
		s.freighterClient.Refresh()

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			http.Error(w, "timeout connecting to backend", http.StatusGatewayTimeout)
		} else {
			// TODO End-users should see a response indicating that there is
			// nothing listening on this port.
			http.Error(w, "could not connect to backend", http.StatusInternalServerError)
		}
		log.Printf("could not connect to backend: %v", err)
		return
	}
	conn.Close()

	util.ConvertToIPBased(req, ip)

	if isWebSocket(req) {
		wsp := &proxy.WebSocketReverseProxy{TCPAddr: tcpAddr}
		wsp.ServeHTTP(w, req)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// No-op - request already modified earlier.
		},
	}
	proxy.ServeHTTP(w, req)
}

func isWebSocket(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(req.Header.Get("Connection"), "upgrade")
}

func (s *Server) Start(host string, port uint16) {
	err := s.freighterClient.Refresh()
	if err != nil {
		log.Printf("unable to refresh container info: %v", err)
	}
	s.freighterClient.StartEventListener()

	http.Handle("/", s)
	log.Printf("ferry: listening on %s:%d", host, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil))
}

func NewServer(
	freighterClient freighter.Client,
	backendDialTimeout time.Duration,
) *Server {
	timeout := time.Minute
	if backendDialTimeout != 0 {
		timeout = backendDialTimeout
	}

	return &Server{
		BackendDialTimeout: timeout,
		freighterClient:    freighterClient,
	}
}
