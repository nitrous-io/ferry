package freightertest

import (
	"net/http"

	"github.com/nitrous-io/ferry/freighter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client", func() {
	var (
		client *freighter.DefaultClient
	)

	Describe("ContainerIP()", func() {
		BeforeEach(func() {
			client = freighter.NewClient("", "")
			client.ContainerMap = map[string]string{
				"id-1":     "4.3.2.1",
				"deadbeef": "4.3.2.1",
				"id-2":     "8.8.8.8",
				"cafebad":  "8.8.8.8",
			}
		})

		Context("when given the identifier of a container that exists in the map", func() {
			It("returns the container's IP address", func() {
				ip := client.ContainerIP("id-1")
				Expect(ip.String()).To(Equal("4.3.2.1"))

				ip = client.ContainerIP("cafebad")
				Expect(ip.String()).To(Equal("8.8.8.8"))
			})
		})

		Context("when given the identifier of a container that does not exist in the map", func() {
			It("returns nil", func() {
				ip := client.ContainerIP("non-existent")
				Expect(ip).To(BeNil())
			})
		})
	})

	Describe("Update()", func() {
		var (
			server       *ghttp.Server
			statusCode   int
			responseBody string
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			client = freighter.NewClient(server.URL(), "")
			responseBody = `{
				"container": {
					"id":"5ca7067d",
					"name":"yosemite",
					"image":"freighter.yosemite",
					"host":"172.17.0.3",
					"ssh_port":60001,
					"state":"running"
				}
			}`

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/containers/5ca7067d"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("User-Agent", "ferry"),
				ghttp.RespondWithPtr(&statusCode, &responseBody),
			))
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the request succeeds", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
			})

			It("populates the ContainerMap with the container's info", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				err := client.Update("5ca7067d")

				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(client.ContainerMap).To(Equal(map[string]string{
					"5ca7067d": "172.17.0.3",
					"yosemite": "172.17.0.3",
				}))
			})
		})

		Context("when the response is HTTP 404 not found", func() {
			BeforeEach(func() {
				statusCode = http.StatusNotFound
				responseBody = `{"error":"no such container"}`
			})

			It("does not add to the ContainerMap", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				err := client.Update("5ca7067d")

				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(client.ContainerMap).To(BeEmpty())
			})

			Context("when the requested container info already exists", func() {
				BeforeEach(func() {
					client.AddOrUpdateContainer(&freighter.Container{
						ID:    "5ca7067d",
						Name:  "yosemite",
						Host:  "172.17.0.3",
						State: "running",
					})
					Expect(client.ContainerMap).To(Equal(map[string]string{
						"5ca7067d": "172.17.0.3",
						"yosemite": "172.17.0.3",
					}))
				})

				It("removes its entries from the ContainerMap", func() {
					err := client.Update("5ca7067d")

					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).To(HaveLen(1))
					Expect(client.ContainerMap).To(BeEmpty())
				})
			})
		})

		Context("when receiving an unexpected response", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				responseBody = ""
			})

			It("returns an error and does not add to the ContainerMap", func() {
				err := client.Update("5ca7067d")

				Expect(err).To(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(client.ContainerMap).To(BeEmpty())
			})
		})
	})

	Describe("Refresh()", func() {
		var (
			server       *ghttp.Server
			statusCode   int
			responseBody string
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			client = freighter.NewClient(server.URL(), "")
			responseBody = `{
				"containers": [
					{
						"id":"5ca7067d",
						"name":"yosemite",
						"image":"freighter.yosemite",
						"host":"172.17.0.3",
						"ssh_port":60001,
						"state":"running"
					},
					{
						"id":"554e835",
						"name":"mavericks",
						"image":"freighter.mavericks",
						"host":"172.17.0.5",
						"ssh_port":60002,
						"state":"running"
					}
				]
			}`

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/containers"),
				ghttp.VerifyHeaderKV("Content-Type", "application/json"),
				ghttp.VerifyHeaderKV("User-Agent", "ferry"),
				// Don't use RepondWithJSONEncoded() because we want to test
				// that our JSON "schemas" (ContainerResponse and
				// ContainersResponse) are correct.
				ghttp.RespondWithPtr(&statusCode, &responseBody),
			))
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the request succeeds", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
			})

			It("populates the ContainerMap with container info", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				err := client.Refresh()

				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(client.ContainerMap).To(Equal(map[string]string{
					"5ca7067d":  "172.17.0.3",
					"yosemite":  "172.17.0.3",
					"554e835":   "172.17.0.5",
					"mavericks": "172.17.0.5",
				}))
			})
		})

		Context("when the response is empty", func() {
			BeforeEach(func() {
				statusCode = http.StatusNotFound
				responseBody = ""
			})

			It("returns an error and does not populate the ContainerMap", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				err := client.Refresh()

				Expect(err).To(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(client.ContainerMap).To(BeEmpty())
			})
		})

		Context("when a subsequent Refresh() is made before the first one completes", func() {
			PIt("should have tests")
		})
	})

	Describe("StartEventListener()", func() {
		PIt("should have tests")
	})

	Describe("StopEventListener()", func() {
		PIt("should have tests")
	})

	Describe("AddOrUpdateContainer()", func() {
		var (
			container *freighter.Container
		)

		BeforeEach(func() {
			client = freighter.NewClient("", "")
		})

		Context("when container is running and has a host (IP address)", func() {
			BeforeEach(func() {
				container = &freighter.Container{
					ID:    "5ca7067d",
					Name:  "yosemite",
					Host:  "172.17.0.3",
					State: "running",
				}
			})

			It("adds entries by ID and name to the ContainerMap", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				client.AddOrUpdateContainer(container)

				Expect(client.ContainerMap).To(Equal(map[string]string{
					"5ca7067d": "172.17.0.3",
					"yosemite": "172.17.0.3",
				}))
			})

			Context("when the container already exists in the ContainerMap", func() {
				BeforeEach(func() {
					client.ContainerMap = map[string]string{
						"5ca7067d": "172.17.0.3",
						"yosemite": "172.17.0.3",
					}
				})

				It("does not modify the ContainerMap", func() {
					client.AddOrUpdateContainer(container)

					Expect(client.ContainerMap).To(Equal(map[string]string{
						"5ca7067d": "172.17.0.3",
						"yosemite": "172.17.0.3",
					}))
				})
			})
		})

		Context("when container is not running", func() {
			BeforeEach(func() {
				container = &freighter.Container{
					ID:    "5ca7067d",
					Name:  "yosemite",
					Host:  "172.17.0.3",
					State: "stopped",
				}
			})

			It("does not add to the ContainerMap", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				client.AddOrUpdateContainer(container)

				Expect(client.ContainerMap).To(BeEmpty())
			})

			Context("when the container already exists in the ContainerMap", func() {
				BeforeEach(func() {
					client.AddOrUpdateContainer(&freighter.Container{
						ID:    "5ca7067d",
						Name:  "yosemite",
						Host:  "172.17.0.3",
						State: "running",
					})
					Expect(client.ContainerMap).To(Equal(map[string]string{
						"5ca7067d": "172.17.0.3",
						"yosemite": "172.17.0.3",
					}))
				})

				It("removes its entries from the ContainerMap", func() {
					client.AddOrUpdateContainer(container)

					Expect(client.ContainerMap).To(BeEmpty())
				})
			})
		})

		Context("when container does not have a host (IP address)", func() {
			BeforeEach(func() {
				container = &freighter.Container{
					ID:    "5ca7067d",
					Name:  "yosemite",
					Host:  "172.17.0.3",
					State: "stopped",
				}
			})

			It("does not add to the ContainerMap", func() {
				Expect(client.ContainerMap).To(BeEmpty())

				client.AddOrUpdateContainer(container)

				Expect(client.ContainerMap).To(BeEmpty())
			})

			Context("when the container already exists in the ContainerMap", func() {
				BeforeEach(func() {
					client.AddOrUpdateContainer(&freighter.Container{
						ID:    "5ca7067d",
						Name:  "yosemite",
						Host:  "172.17.0.3",
						State: "running",
					})
					Expect(client.ContainerMap).To(Equal(map[string]string{
						"5ca7067d": "172.17.0.3",
						"yosemite": "172.17.0.3",
					}))
				})

				It("removes its entries from the ContainerMap", func() {
					client.AddOrUpdateContainer(container)

					Expect(client.ContainerMap).To(BeEmpty())
				})
			})
		})
	})

	Describe("RemoveContainer()", func() {
		BeforeEach(func() {
			client = freighter.NewClient("", "")
			client.AddOrUpdateContainer(&freighter.Container{
				ID:    "5ca7067d",
				Name:  "yosemite",
				Host:  "172.17.0.3",
				State: "running",
			})
			Expect(client.ContainerMap).To(Equal(map[string]string{
				"5ca7067d": "172.17.0.3",
				"yosemite": "172.17.0.3",
			}))
		})

		It("removes the container's entries from the ContainerMap", func() {
			client.RemoveContainer("5ca7067d")

			Expect(client.ContainerMap).To(BeEmpty())
		})
	})
})
