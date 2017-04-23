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
		page, err = agoutiDriver.NewPage(agouti.Desired(agouti.Capabilities{
			"chromeOptions": map[string][]string{
				"args": []string{
					"headless",
					// There is no GPU on our Ubuntu box!
					"disable-gpu",

					// Sandbox requires namespace permissions that we don't have on a container
					"no-sandbox",
				},
			},
		}))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(done Done) {
		// time.Sleep(20 * time.Second)
		Expect(page.Destroy()).To(Succeed())
		close(done)
	})

	Context("When I deploy simple app using web interface", func() {
		BeforeEach(func() {
			Expect(page.Navigate("http://localhost:12080/")).To(Succeed())

			Eventually(page.FindByID("deploy_button")).Should(BeVisible())
			Expect(page.FindByID("deploy_button").Click()).To(Succeed())

			Eventually(page.FindByID("descriptorFile")).Should(BeVisible())
			Expect(page.FindByID("descriptorFile").UploadFile("simple.json")).To(Succeed())

			Eventually(page.FindByButton("Deploy")).Should(HaveText("Deploy"))
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
						Eventually(page.FindByButton("Delete")).Should(BeVisible())
						Expect(page.FindByButton("Delete").Click()).To(Succeed())
					})

					It("Should go back to the index page", func() {
						getURL := func() string {
							url, err := page.URL()
							Expect(err).ToNot(HaveOccurred())
							return url
						}
						Eventually(getURL, 2.0).Should(Equal("http://localhost:12080/#/"))
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
