package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type HTTPReverseProxy struct{}

func (h *HTTPReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("Inbound request: %+v", req)

	// This handler only proxies HTTP requests.
	req.URL.Scheme = "http"

	// Make sure we set the Host header.
	req.Header.Set("Host", req.URL.Host)
	req.URL.Host = req.Host

	// This should not be set in a request.
	req.RequestURI = ""

	removeHopByHopHeaders(req)

	// Be a nice reverse proxy - add or append to X-Forwarded-For header.
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if xff, ok := req.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(xff, ", ") + ", " + clientIP
		}
		req.Header.Set("X-Forwarded-For", clientIP)
	}

	transport := http.DefaultTransport
	log.Printf("Outbound request: %+v,", req)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		log.Printf("ferry: http proxy error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("Received response: %+v", resp)
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode) // This must be done AFTER copying headers.
	h.copyResponse(w, resp.Body)
}

func copyHeaders(dst, src http.Header) {
	for k, v := range src {
		for _, vv := range v {
			log.Printf("Adding header: %v = %v", k, vv)
			dst.Add(k, vv)
		}
	}
}

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Remove hop-by-hop headers, which are meaningful only to proxies and we don't
// want to forward them to the backend.
// See http://www.w3.org/Protocols/rfc2616/rfc2616.txt section 13.5.1.
func removeHopByHopHeaders(req *http.Request) {
	for _, h := range hopByHopHeaders {
		req.Header.Del(h)
	}
	// Strip Proxy-Connection header. Curl will add this.
	// See http://homepage.ntlworld.com/jonathan.deboynepollard/FGA/web-proxy-connection-header.html
	req.Header.Del("Proxy-Connection")
}

func (h *HTTPReverseProxy) copyResponse(dst io.Writer, src io.Reader) {
	written, _ := io.Copy(dst, src)
	log.Printf("Copied %v bytes to client", written)
}
