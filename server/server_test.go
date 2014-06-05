package server_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/nitrous-io/ferry/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeClient struct {
	ContainerMap map[string]string

	RefreshCalled            bool
	RefreshError             error
	UpdateCalled             bool
	UpdateError              error
	StartEventListenerCalled bool
	StopEventListenerCalled  bool
}

func (c *FakeClient) ContainerIP(identifier string) net.IP {
	if ip, ok := c.ContainerMap[identifier]; ok {
		return net.ParseIP(ip)
	}
	return nil
}

func (c *FakeClient) Update(id string) error {
	c.UpdateCalled = true
	return c.UpdateError
}

func (c *FakeClient) Refresh() error {
	c.RefreshCalled = true
	return c.RefreshError
}

func (c *FakeClient) StartEventListener() {
	c.StartEventListenerCalled = true
}

func (c *FakeClient) StopEventListener() {
	c.StopEventListenerCalled = true
}

var _ = Describe("Server", func() {
	var (
		backend       *httptest.Server
		backendIP     string
		backendPort   string
		backendCalled int
		frontend      *httptest.Server
		proxyServer   *server.Server
		client        *http.Client
		err           error
		fakeClient    *FakeClient
	)

	BeforeEach(func() {
		fakeClient = &FakeClient{}
		proxyServer = server.NewServer(fakeClient, 5*time.Second)
	})

	Describe("serving requests as a reverse proxy", func() {
		BeforeEach(func() {
			// Setup backend (upstream) server.
			backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				backendCalled++

				Expect(r.Header.Get("X-Forwarded-For")).NotTo(BeEmpty())
				// TODO Add more expectations

				w.Header().Set("X-Race", "half-elf")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("I'm your backend"))
			}))
			hostPort := strings.Replace(backend.URL, "http://", "", -1)
			backendIP, backendPort, err = net.SplitHostPort(hostPort)
			Expect(err).NotTo(HaveOccurred())

			// Setup frontend (reverse proxy) server.
			frontend = httptest.NewServer(proxyServer)
			frontendURL, err := url.Parse(frontend.URL)
			Expect(err).NotTo(HaveOccurred())

			// Setup HTTP client that uses the frontend server as a proxy.
			tr := &http.Transport{
				Proxy:           http.ProxyURL(frontendURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client = &http.Client{Transport: tr}

			// Setup fake Freighter client to respond with backend.
			fakeClient.ContainerMap = map[string]string{
				"container-1": backendIP,
			}
		})

		AfterEach(func() {
			backend.Close()
			frontend.Close()
		})

		Context("when the Freighter client responds with a container that can serve the request", func() {

			It("blah blah blah blah blah", func() {
				url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
				resp, err := client.Get(url)
				Expect(err).NotTo(HaveOccurred())
				log.Printf("resp: %v", resp)
				Expect(resp.Header.Get("X-Race")).To(Equal("half-elf"))

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
				Expect(string(body)).To(Equal("I'm your backend"))
			})

			Context("when there is a timeout connecting to the container", func() {
				PIt("asks the Freighter client to refresh container info")
				PIt("responds with HTTP 502 Gateway Timeout")
			})

			Context("when the container is not listening to requests", func() {
				PIt("asks the Freighter client to refresh container info")
				PIt("responds with HTTP 503 Internal Server Error")
			})

			Context("when the container can respond to the request", func() {
				Context("when it is an HTTP request", func() {
					PIt("proxies the request to the container")
					PIt("responds with the container's response")
				})

				Context("when it is a WebSocket request", func() {
					PIt("hijacks and proxies the WebSocket connection")
				})
			})
		})

		Context("when the Freighter client says there is no container that can serve the request", func() {
			PIt("responds with 502 Bad Gateway")
		})
	})

	Describe("Start()", func() {
		PIt("asks the Freighter client to refresh container info")
		PIt("asks the Freighter client to start listening to container events")
		PIt("should probably test that it listens on the given host and port")
	})

	Describe("NewServer()", func() {
		It("returns a *Server with a default backend dial timeout of 1 minute", func() {
			s := server.NewServer(fakeClient, 0)
			Expect(s.BackendDialTimeout).To(Equal(time.Minute))
		})

		It("returns a *Server with backend dial timeout set", func() {
			s := server.NewServer(fakeClient, 5*time.Second)
			Expect(s.BackendDialTimeout).To(Equal(5 * time.Second))
		})
	})
})
