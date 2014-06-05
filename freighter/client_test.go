package freighter_test

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
			client = freighter.NewClient("")
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

	Describe("Refresh()", func() {
		var (
			server       *ghttp.Server
			statusCode   int
			responseBody string
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			client = freighter.NewClient(server.URL())
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
	})
})
