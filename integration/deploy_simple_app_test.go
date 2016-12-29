package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

var _ = Describe("DeploySimpleApp", func() {
	var page *agouti.Page

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
	})

})
