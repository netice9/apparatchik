package integration_test

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /_ping", func() {
	var err error
	var response *http.Response

	BeforeEach(func() {
		response, err = http.Get("http://localhost:12080/_ping")
	})

	AfterEach(func() {
		response.Body.Close()
	})

	It("Should not return error", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should return 200 status code", func() {
		Expect(response.StatusCode).To(Equal(200))
	})

	It("Should return 'OK' body", func() {
		data, err := ioutil.ReadAll(response.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("OK"))
	})

})
