package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
)

type WebSocketReverseProxy struct {
	// TCPAddr is the TCP address of the backend.
	TCPAddr *net.TCPAddr
}

func (h *WebSocketReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, err := net.Dial("tcp", h.TCPAddr.String())
	if err != nil {
		http.Error(w, "could not connect to WebSocket backend", http.StatusInternalServerError)
		log.Printf("could not connect to WebSocket backend: %v", err)
		return
	}
	defer conn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	newConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "hijacking failed", http.StatusInternalServerError)
		log.Printf("hijack error: %v", err)
		return
	}
	defer newConn.Close()

	err = req.Write(conn)
	if err != nil {
		http.Error(w, "error writing WebSocket request to backend", http.StatusInternalServerError)
		log.Print("error writing WebSocket request to backend: %v", err)
		return
	}

	bidirectionalCopy(conn, newConn)
}

func bidirectionalCopy(rw1, rw2 io.ReadWriter) {
	errChan := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errChan <- err
	}
	go cp(rw1, rw2)
	go cp(rw2, rw1)
	<-errChan
}
