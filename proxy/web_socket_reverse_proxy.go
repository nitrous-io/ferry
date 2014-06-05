package proxy

import "net/http"

type WebSocketReverseProxy struct{}

func (h *WebSocketReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("not implemented"))
}
