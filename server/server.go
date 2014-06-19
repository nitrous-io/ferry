package server

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
		*http.Server

		// BackendDialTimeout specifies the amount of time to wait for a connection
		// to be established to the backend (i.e. container).
		// If 0, a default of 1 minute is used to avoid connection exhaustion.
		BackendDialTimeout time.Duration

		// StopTimeout specifies the amount of time to wait for existing
		// connections to be closed when the server is stopped. After the
		// StopTimeout has passed, remaining connections are forcibly closed.
		// Defaults to 0, where no timeout will be applied (i.e. the server
		// will only stop after all connections are )
		StopTimeout time.Duration

		SignalChan chan os.Signal

		freighterClient freighter.Client
	}

	errTemplateData struct {
		BoxName string
		Error   string
	}
)

var noProxyTransport = &http.Transport{Proxy: nil}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	util.Logger.Debugf("Request received: %+v", req)

	containerName := util.ParseLeftMostLabel(req.Host)
	ip := s.freighterClient.ContainerIP(containerName)
	if ip == nil {
		util.Logger.Infof("Unmatched request for %v", req.Host)
		errorPage(w, http.StatusBadGateway, containerName, serverNotRunning)
		return
	}

	port := util.ParsePort(req)
	tcpAddr := &net.TCPAddr{IP: ip, Port: int(port)}
	util.Logger.Debugf("Mapped to container IP and port: %v", tcpAddr)

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
		util.Logger.Warnf("Could not connect to backend: %v", err)
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
			// Proxies in front of Ferry, e.g. Nginx, will strip away the scheme
			// (i.e. "http://example.com:3000/foo" becomes "/foo"), so we
			// explicitly set it.
			req.URL.Scheme = "http"
		},
	}
	proxy.ServeHTTP(w, req)
}

func (s *Server) Start(host string, port uint16) {
	s.startFreighterListener()

	// FIXME
	// 1. Abstract all this shit into another package.
	// 2. Make this server more like a http.Server.
	connections := map[net.Conn]struct{}{} // Keep a map of active connections.
	add := make(chan net.Conn)             // New active connections are sent here.
	remove := make(chan net.Conn)          // Connections that stop being active are sent here.

	// Send a bool channel to this channel that will be notified when there are
	// no more active connections.
	stop := make(chan chan bool)
	kill := make(chan bool) // Force-kill active connections.

	addr := fmt.Sprintf("%s:%d", host, port)
	s.Server = &http.Server{Addr: addr, Handler: s}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		util.Logger.Fatalf("ferry: cannot start server: %v", err)
	}

	s.Server.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateActive:
			add <- conn
		case http.StateClosed, http.StateIdle:
			remove <- conn
		case http.StateHijacked:
			util.Logger.Debugf("Connection hijacked, not sure what to do: %+v", conn.RemoteAddr())
		}
	}

	// Manage connections.
	go func() {
		var done chan bool
		for {
			select {
			case conn := <-add:
				util.Logger.Debugf("Connection added: %v", conn.RemoteAddr())
				connections[conn] = struct{}{}
			case conn := <-remove:
				util.Logger.Debugf("Connection removed: %v", conn.RemoteAddr())
				delete(connections, conn)
				if done != nil && len(connections) == 0 {
					done <- true
					return
				}
			case done = <-stop:
				util.Logger.Debug("Stop requested")
				if len(connections) == 0 {
					done <- true
					return
				}
			case <-kill:
				util.Logger.Debug("Killing all connections")
				for conn := range connections {
					conn.Close()
				}
				return
			}
		}
	}()

	// Wait for stop signals.
	signal.Notify(s.SignalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-s.SignalChan
		util.Logger.Debugf("Signal received: %v", sig)
		s.Server.SetKeepAlivesEnabled(false)
		listener.Close()
		signal.Stop(s.SignalChan)
		close(s.SignalChan)
	}()

	util.Logger.Debugf("ferry: started listening on %v", addr)
	err = s.Server.Serve(listener)
	if err != nil {
		util.Logger.Fatalf("ferry: cannot start server: %v", err)
	}

	done := make(chan bool)
	stop <- done

	if s.StopTimeout > 0 {
		select {
		case <-done:
			util.Logger.Debug("ferry: all connections closed gracefully")
		case <-time.After(s.StopTimeout):
			util.Logger.Debug("ferry: timeout reached, force-closing existing connections")
			kill <- true
		}
	} else {
		<-done
	}

	util.Logger.Info("ferry: stopped")
}

// TODO This should probably belong in freighter.Client.
func (s *Server) startFreighterListener() {
	if err := s.freighterClient.Refresh(); err != nil {
		util.Logger.Warnf("Unable to refresh container info, polling until it becomes available: %v", err)
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			for _ = range ticker.C {
				err := s.freighterClient.Refresh()
				if err == nil {
					util.Logger.Info("Successfully refreshed container info")
					s.freighterClient.StartEventListener()
					break
				}
				util.Logger.Warnf("Polling for container info failed: %v", err)
			}
		}()
	} else {
		s.freighterClient.StartEventListener()
	}
}

func NewServer(
	freighterClient freighter.Client,
	stopTimeout time.Duration,
	backendDialTimeout time.Duration,
) *Server {
	dialTimeout := time.Minute
	if backendDialTimeout != 0 {
		dialTimeout = backendDialTimeout
	}

	return &Server{
		BackendDialTimeout: dialTimeout,
		StopTimeout:        stopTimeout,
		SignalChan:         make(chan os.Signal, 1),
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
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	w.Write([]byte(buf.String()))
}
