package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"text/template"
	"time"

	"github.com/nitrous-io/ferry/freighter"
	"github.com/nitrous-io/ferry/proxy"
	"github.com/nitrous-io/ferry/util"
)

var backendError = template.Must(template.New("httpError").Parse(backendErrorTmpl))

var serverNotRunningErrorString = "There does not seem to be a server running on this port"
var serverTimedOutErrorString = "The server timed out on this request"

type Server struct {
	// BackendDialTimeout specifies the amount of time to wait for a connection
	// to be established to the backend (i.e. container).
	// If 0, a default of 1 minute is used to avoid connection exhaustion.
	BackendDialTimeout time.Duration

	freighterClient freighter.Client
}

type backendErrorTemplateData struct {
	BoxName string
	Error   string
}

func (s *Server) errorPage(containerName string, errorString string) string {
	buf := &bytes.Buffer{}
	data := &backendErrorTemplateData{BoxName: containerName, Error: errorString}

	err := backendError.Execute(buf, &data)
	if err != nil {
		panic("Error compiling backendErrorTemplate")
	}

	return buf.String()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request received: %+v", req)

	containerName := util.ParseLeftMostLabel(req.Host)
	ip := s.freighterClient.ContainerIP(containerName)
	if ip == nil {
		log.Printf("unmatched request for %v", req.Host)
		http.Error(w, s.errorPage(containerName, serverNotRunningErrorString), http.StatusBadGateway)
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
			http.Error(w, s.errorPage(containerName, serverTimedOutErrorString), http.StatusGatewayTimeout)
		} else {
			http.Error(w, s.errorPage(containerName, serverNotRunningErrorString), http.StatusInternalServerError)
		}
		log.Printf("could not connect to backend: %v", err)
		return
	}
	defer conn.Close()

	if isWebSocket(req) {
		wsp := &proxy.WebSocketReverseProxy{}
		wsp.ServeHTTP(w, req)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			util.ConvertToIPBased(req, ip)
			log.Printf("Modified request: %+v", req)
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
