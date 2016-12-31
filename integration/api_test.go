package integration_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/netice9/apparatchik/core"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("API V1.0", func() {
	BeforeEach(clearApparatchik)

	Describe("PUT /api/v1.0/applications/:app_name", func() {
		Context("When a valid application descriptor is provided in the body", func() {
			var response *http.Response
			BeforeEach(func() {
				reader := strings.NewReader(`
          {
            "goals": {
              "simple-goal": {
                "image": "alpine:3.4",
                "command": ["sleep","999999999"]
              }
            },
            "main_goal": "simple-goal"
          }`,
				)
				req, err := http.NewRequest("PUT", "http://localhost:12080/api/v1.0/applications/test-app", reader)
				Expect(err).ToNot(HaveOccurred())
				response, err = http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(response.Body.Close()).To(Succeed())
			})

			It("Should return 201 status code", func() {
				Expect(response.StatusCode).To(Equal(201))
			})

			It("Should return application/json content type", func() {
				Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
			})

			It("Should return application descriptor in the body", func() {
				data, err := ioutil.ReadAll(response.Body)
				Expect(err).ToNot(HaveOccurred())

				// TODO replace this with a test-specific struct!
				respData := core.ApplicationStatus{}
				Expect(json.Unmarshal(data, &respData)).To(Succeed())
				Expect(respData.Name).To(Equal("test-app"))

			})

			It("Should return location with the url of the app", func() {
				location, err := response.Location()
				Expect(err).ToNot(HaveOccurred())
				Expect(location.String()).To(Equal("http://localhost:12080/api/v1.0/applications/test-app"))

			})

		})

	})

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

		Context("When there is one application deployed", func() {
			BeforeEach(func() {
				reader := strings.NewReader(`
          {
            "goals": {
              "simple-goal": {
                "image": "alpine:3.4",
                "command": ["sleep","999999999"]
              }
            },
            "main_goal": "simple-goal"
          }`,
				)
				req, err := http.NewRequest("PUT", "http://localhost:12080/api/v1.0/applications/test-app", reader)
				Expect(err).ToNot(HaveOccurred())
				response, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.StatusCode).To(Equal(201))
			})

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

			It("Should return JSON containing one application", func() {
				data, err := ioutil.ReadAll(response.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(data).To(MatchJSON(`["test-app"]`))
			})

		})

	})
})
