package server

import (
	"bytes"
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

const (
	serverNotRunning = "There does not seem to be a server running on this port"
	serverTimedOut   = "The server timed out on this request"
)

type (
	Server struct {
		// BackendDialTimeout specifies the amount of time to wait for a connection
		// to be established to the backend (i.e. container).
		// If 0, a default of 1 minute is used to avoid connection exhaustion.
		BackendDialTimeout time.Duration

		freighterClient freighter.Client
	}

	errTemplateData struct {
		BoxName string
		Error   string
	}
)

var noProxyTransport = &http.Transport{Proxy: nil}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request received: %+v", req)

	containerName := util.ParseLeftMostLabel(req.Host)
	ip := s.freighterClient.ContainerIP(containerName)
	if ip == nil {
		log.Printf("unmatched request for %v", req.Host)
		errorPage(w, http.StatusBadGateway, containerName, serverNotRunning)
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
			errorPage(w, http.StatusGatewayTimeout, containerName, serverTimedOut)
		} else {
			errorPage(w, http.StatusInternalServerError, containerName, serverNotRunning)
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
		Transport: noProxyTransport,
		Director: func(req *http.Request) {
			// No-op - request already modified earlier.
		},
	}
	proxy.ServeHTTP(w, req)
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

func isWebSocket(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(req.Header.Get("Connection"), "upgrade")
}

func errorPage(w http.ResponseWriter, statusCode int, containerName, error string) {
	buf := &bytes.Buffer{}
	err := errTemplate.Execute(buf, &errTemplateData{BoxName: containerName, Error: error})
	if err != nil {
		http.Error(w, "an unknown error has occurred", http.StatusInternalServerError)
	}
	http.Error(w, buf.String(), statusCode)
}
