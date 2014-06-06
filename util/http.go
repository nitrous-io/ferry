package util

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

var portMap = map[string]uint16{
	"http":  80,
	"https": 443,
}

// ParsePort returns the port of an HTTP request. If there is no port specified
// in the request, 80 or 443 (HTTPS) is assumed.
func ParsePort(req *http.Request) uint16 {
	if _, portStr, err := net.SplitHostPort(req.Host); err == nil {
		if p, err := strconv.ParseUint(portStr, 10, 16); err == nil {
			return uint16(p)
		}
	}
	return portMap[req.URL.Scheme]
}

// ParseLeftMostLabel returns the left-most DNS label of the given domain name.
// Examples:
//   "winterfell-nodejs-42.example.com" -> "winterfell-nodejs-42"
//   "cersei.foo.bar.example.com" -> "cersei"
func ParseLeftMostLabel(domain string) string {
	splits := strings.Split(domain, ".")
	return splits[0]
}

// ConvertToIPBased modifies an HTTP request to use an IP address as its host.
// The "Host" header is preserved.
func ConvertToIPBased(req *http.Request, ip net.IP) *http.Request {
	// Change Host to "IP:Port".
	port := ParsePort(req)
	req.URL.Host = net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))

	return req
}
