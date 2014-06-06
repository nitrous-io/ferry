package util_test

import (
	"net"
	"net/http"

	"github.com/nitrous-io/ferry/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("ParsePort()", func() {
		var (
			req *http.Request
			err error
		)

		Context("when request has a port specified", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "http://example.com:3000/", nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the port", func() {
				port := util.ParsePort(req)
				Expect(port).To(BeNumerically("==", 3000))
			})
		})

		Context("when a non-HTTPS request does not have a port specified", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "http://example.com/", nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns 80", func() {
				port := util.ParsePort(req)
				Expect(port).To(BeNumerically("==", 80))
			})
		})

		Context("when an HTTPS request does not have a port specified", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "https://example.com/", nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns 443", func() {
				port := util.ParsePort(req)
				Expect(port).To(BeNumerically("==", 443))
			})
		})
	})

	Describe("ParseLeftMostLabel()", func() {
		It("returns the left-most DNS label of a domain name", func() {
			Expect(util.ParseLeftMostLabel("com")).To(Equal("com"))
			Expect(util.ParseLeftMostLabel("example.com")).To(Equal("example"))
			Expect(util.ParseLeftMostLabel("mage.example.com")).To(Equal("mage"))
			Expect(util.ParseLeftMostLabel("wizard.mage.example.com")).To(Equal("wizard"))
			Expect(util.ParseLeftMostLabel("arcanist.wizard.mage.example.com")).To(Equal("arcanist"))
		})
	})

	Describe("ConvertToIPBased()", func() {
		var (
			req *http.Request
			ip  net.IP
			err error
		)

		BeforeEach(func() {
			ip = net.ParseIP("4.3.2.1")
		})

		Context("when request includes a port", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "http://example.com:3000/fool", nil)
				Expect(err).NotTo(HaveOccurred())

				util.ConvertToIPBased(req, ip)
			})

			It("sets the Host header to the request's URL host:port", func() {
				Expect(req.Host).To(Equal("example.com:3000"))
			})

			It("changes the request's host to IP:port", func() {
				Expect(req.URL.Host).To(Equal("4.3.2.1:3000"))
				Expect(req.URL.String()).To(Equal("http://4.3.2.1:3000/fool"))
			})
		})

		Context("when request is an HTTP request that does not include a port", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "http://example.com/bard", nil)
				Expect(err).NotTo(HaveOccurred())

				util.ConvertToIPBased(req, ip)
			})

			It("sets the Host header to the request's URL host", func() {
				Expect(req.Host).To(Equal("example.com"))
			})

			It("changes the request's host to IP:80", func() {
				Expect(req.URL.Host).To(Equal("4.3.2.1:80"))
				Expect(req.URL.String()).To(Equal("http://4.3.2.1:80/bard"))
			})
		})

		Context("when request is an HTTPS request that does not include a port", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("GET", "https://example.com/rogue", nil)
				Expect(err).NotTo(HaveOccurred())

				util.ConvertToIPBased(req, ip)
			})

			It("sets the Host header to the request's URL host", func() {
				Expect(req.Host).To(Equal("example.com"))
			})

			It("changes the request's host to IP:443", func() {
				Expect(req.URL.Host).To(Equal("4.3.2.1:443"))
				Expect(req.URL.String()).To(Equal("https://4.3.2.1:443/rogue"))
			})
		})
	})
})
