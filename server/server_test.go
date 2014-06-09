package server_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"code.google.com/p/go.net/websocket"

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

		frontend     *httptest.Server
		frontendIP   string
		frontendPort string

		proxyServer *server.Server
		client      *http.Client
		err         error
		fakeClient  *FakeClient
	)

	BeforeEach(func() {
		fakeClient = &FakeClient{}
		proxyServer = server.NewServer(fakeClient, 5*time.Second)
		backendCalled = 0
	})

	Describe("serving requests as a reverse proxy", func() {
		BeforeEach(func() {
			// Setup backend (upstream) server.
			backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				backendCalled++

				Expect(r.Host).To(Equal(fmt.Sprintf("container-1.example.com:%s", backendPort)))
				Expect(r.Header.Get("X-Forwarded-For")).To(ContainSubstring(frontendIP))

				w.Header().Set("X-Custom-Header", "your-head")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("I'm your backend"))
			}))
			backendAddr := backend.Listener.Addr().String()
			backendIP, backendPort, err = net.SplitHostPort(backendAddr)
			Expect(err).NotTo(HaveOccurred())

			// Setup frontend (reverse proxy) server.
			frontend = httptest.NewServer(proxyServer)
			frontendAddr := frontend.Listener.Addr().String()
			frontendIP, _, err = net.SplitHostPort(frontendAddr)
			Expect(err).NotTo(HaveOccurred())
			frontendURL, err := url.Parse(frontend.URL)
			Expect(err).NotTo(HaveOccurred())

			// Setup HTTP client that uses the frontend server as a proxy.
			tr := &http.Transport{Proxy: http.ProxyURL(frontendURL)}
			client = &http.Client{Transport: tr}
		})

		AfterEach(func() {
			backend.Close()
			frontend.Close()
		})

		Context("when the Freighter client responds with a container that can serve the request", func() {
			BeforeEach(func() {
				// Setup fake Freighter client to respond with backend.
				fakeClient.ContainerMap = map[string]string{
					"container-1": backendIP,
				}
			})

			It("makes a request to the container and returns its response", func() {
				url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
				resp, err := client.Get(url)
				Expect(err).NotTo(HaveOccurred())

				Expect(backendCalled).To(Equal(1))

				Expect(resp.Header.Get("X-Custom-Header")).To(Equal("your-head"))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
				Expect(string(body)).To(Equal("I'm your backend"))
			})

			Context("when there is a timeout connecting to the container", func() {
				BeforeEach(func() {
					proxyServer.BackendDialTimeout = time.Microsecond // Such fast.
				})

				It("asks the Freighter client to refresh container info", func() {
					url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
					_, err := client.Get(url)
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeClient.RefreshCalled).To(BeTrue())
				})

				It("responds with HTTP 502 Gateway Timeout", func() {
					url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
					resp, err := client.Get(url)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout))

					body, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()
					Expect(body).To(ContainSubstring("timeout connecting to backend"))
				})
			})

			Context("when the container is not listening to requests", func() {
				BeforeEach(func() {
					backend.Close()
				})

				It("asks the Freighter client to refresh container info", func() {
					url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
					_, err := client.Get(url)
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeClient.RefreshCalled).To(BeTrue())
				})

				It("responds with HTTP 503 Internal Server Error", func() {
					url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
					resp, err := client.Get(url)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

					body, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()
					Expect(body).To(ContainSubstring("could not connect to backend"))
				})
			})
		})

		Context("when the Freighter client says there is no container that can serve the request", func() {
			BeforeEach(func() {
				// Setup fake Freighter client with no containers.
				fakeClient.ContainerMap = map[string]string{}
			})

			It("responds with an HTTP 502 Bad Gateway", func() {
				url := fmt.Sprintf("http://container-1.example.com:%s/", backendPort)
				resp, err := client.Get(url)
				Expect(err).NotTo(HaveOccurred())

				Expect(backendCalled).To(Equal(0))

				Expect(resp.StatusCode).To(Equal(http.StatusBadGateway))

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
				Expect(body).To(ContainSubstring("could not find a backend"))
			})
		})

		Describe("serving a file download response as a reverse proxy", func() {
			var file string

			BeforeEach(func() {
				file = "test-fixtures/test-image.png"
				backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer GinkgoRecover()
					backendCalled++

					Expect(r.Host).To(Equal(fmt.Sprintf("container-1.example.com:%s", backendPort)))
					Expect(r.Header.Get("X-Forwarded-For")).To(ContainSubstring(frontendIP))

					handler := http.FileServer(http.Dir("test-fixtures"))
					handler.ServeHTTP(w, r)
				}))
				backendAddr := backend.Listener.Addr().String()
				backendIP, backendPort, err = net.SplitHostPort(backendAddr)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the Freighter client responds with a container that can serve the request", func() {
				BeforeEach(func() {
					// Setup fake Freighter client to respond with backend.
					fakeClient.ContainerMap = map[string]string{
						"container-1": backendIP,
					}
				})

				It("makes a request to the container and returns the file", func() {
					url := fmt.Sprintf("http://container-1.example.com:%s/test-image.png", backendPort)
					resp, err := client.Get(url)
					Expect(err).NotTo(HaveOccurred())

					Expect(backendCalled).To(Equal(1))
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					body, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()

					fileData, err := ioutil.ReadFile(file)
					Expect(err).NotTo(HaveOccurred())

					Expect(body).To(Equal(fileData))
				})
			})
		})

		Context("when it is a WebSocket request", func() {
			var (
				frontendAddr string
			)

			BeforeEach(func() {
				// Backend echo server.
				backend = httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
					defer GinkgoRecover()
					backendCalled++

					io.Copy(ws, ws)
				}))
				backendAddr := backend.Listener.Addr().String()
				backendIP, backendPort, err = net.SplitHostPort(backendAddr)
				Expect(err).NotTo(HaveOccurred())

				// Setup fake Freighter client to respond with backend.
				fakeClient.ContainerMap = map[string]string{
					"container-1": backendIP,
				}

				// Setup frontend (reverse proxy) server.
				frontend = httptest.NewServer(proxyServer)
				frontendAddr = frontend.Listener.Addr().String()
				frontendIP, frontendPort, err = net.SplitHostPort(frontendAddr)
				Expect(err).NotTo(HaveOccurred())
			})

			It("hijacks and proxies the WebSocket connection", func() {
				// Connect to the frontend server via its IP:port.
				c, err := net.Dial("tcp", frontendAddr)
				Expect(err).NotTo(HaveOccurred())

				// But use a working container's host + the port the backend
				// is listening on.
				containerHost := "container-1.example.com"
				containerURL := fmt.Sprintf("ws://%s:%s/", containerHost, backendPort)
				origin := "http://" + containerHost
				config, err := websocket.NewConfig(containerURL, origin)
				Expect(err).NotTo(HaveOccurred())

				wsConn, err := websocket.NewClient(config, c)
				Expect(err).NotTo(HaveOccurred(), "WebSocket handshake error")

				Expect(backendCalled).To(Equal(1))

				msg := []byte("hello hello hello\n")
				_, err = wsConn.Write(msg)
				Expect(err).NotTo(HaveOccurred(), "Writing to WebSocket failed")

				reply := make([]byte, 512)
				n, err := wsConn.Read(reply)
				Expect(err).NotTo(HaveOccurred(), "Reading from WebSocket failed")

				reply = reply[0:n]
				Expect(bytes.Equal(msg, reply)).To(BeTrue())
				Expect(string(reply)).To(Equal("hello hello hello\n"))

				wsConn.Close()
			})
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
