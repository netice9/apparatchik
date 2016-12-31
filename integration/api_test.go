package integration_test

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("API V1.0", func() {
	BeforeEach(clearApparatchik)

	Describe("GET /api/v1.0/applications", func() {
		Context("When there are no applications defined", func() {
			var response *http.Response
			BeforeEach(func() {
				var err error
				response, err = http.Get("http://localhost:12080/api/v1.0/applications")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(response.Body.Close()).To(Succeed())
			})

			It("Should return 200 status code", func() {
				Expect(response.StatusCode).To(Equal(200))
			})

			It("Should return application/json content type", func() {
				Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
			})

			It("Should return empty JSON array", func() {
				data, err := ioutil.ReadAll(response.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(data).To(MatchJSON("[]"))
			})
		})

	})
})
