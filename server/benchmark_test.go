package server_test

import (
	"net/http"
	"testing"

	"github.com/nitrous-io/ferry/server"
)

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func BenchmarkHTTPReverseProxy(b *testing.B) {
	backend := &http.Server{Addr: "0.0.0.0:60000", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing.
	})}
	go backend.ListenAndServe()
	fakeClient := &FakeClient{
		ContainerMap: map[string]string{
			"container-1": "0.0.0.0",
		},
	}
	server := server.NewServer(fakeClient, 0, 0)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest("GET", "/", nil)
	r.Host = "container-1.example.com:60000"

	for i := 0; i < b.N; i++ {
		server.ServeHTTP(w, r)
	}
}
