package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

var _ = Describe("DeploySimpleApp", func() {
	var page *agouti.Page

	BeforeEach(clearApparatchik)

	BeforeEach(func() {
		var err error
		page, err = agoutiDriver.NewPage()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(done Done) {
		Expect(page.Destroy()).To(Succeed())
		close(done)
	})

	Context("When I deploy simple app using web interface", func() {
		BeforeEach(func() {
			Expect(page.Navigate("http://localhost:12080/")).To(Succeed())
			Expect(page.FindByID("deploy_button").Click()).To(Succeed())
			Expect(page.FindByID("descriptorFile").UploadFile("simple.json")).To(Succeed())
			Expect(page.FindByButton("Deploy").Click()).To(Succeed())
		})
		It("Should have one application deployed", func() {
			Eventually(page.FindByLink("simple")).Should(BeFound())
		})
		Context("When I click on the application name", func() {
			BeforeEach(func() {
				Eventually(page.FindByLink("simple")).Should(BeFound())
				Expect(page.FindByLink("simple").Click()).To(Succeed())
			})
			It("Should have panel with the application name", func() {
				Eventually(page.FindByClass("panel-heading")).Should(HaveText("simple"))
			})

			Context("When I click on the goal", func() {
				BeforeEach(func() {
					Expect(page.FindByLink("simple-goal").Click()).To(Succeed())

				})

				It("Should have CPU stats", func() {
					Eventually(page.First("div.panel-heading")).Should(HaveText("CPU Stats"))
				})

				It("Should have Memory stats", func() {
					Eventually(page.All("div.panel-heading").At(1)).Should(HaveText("Memory Stats"))
				})
			})

			Context("When I click on Delete! button", func() {
				BeforeEach(func() {
					Expect(page.FindByButton("Delete!").Click()).To(Succeed())
				})
				It("Should show deletion confirmation modal", func() {
					Eventually(page.Find("h4.modal-title")).Should(HaveText("Confirm Deleting Application"))
				})

				Context("When I click on the Delete button", func() {
					BeforeEach(func() {
						Expect(page.FindByButton("Delete").Click()).To(Succeed())
					})

					It("Should go back to the index page", func() {
						getURL := func() string {
							url, err := page.URL()
							Expect(err).ToNot(HaveOccurred())
							return url
						}
						Eventually(getURL).Should(Equal("http://localhost:12080/#/"))
					})

					Context("When it is ack to the index page", func() {
						BeforeEach(func() {
							getURL := func() string {
								url, err := page.URL()
								Expect(err).ToNot(HaveOccurred())
								return url
							}
							Eventually(getURL).Should(Equal("http://localhost:12080/#/"))
						})

						It("Should not have link to the simple application", func() {
							Expect(page.FindByLink("simple")).ToNot(BeFound())
						})
					})

				})

			})
		})

	})

})
