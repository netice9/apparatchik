package core_test

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/apparatchik/apparatchik/core"
	"github.com/netice9/cine"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Suite")
}

func init() {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}
	dockerClient = c
	cine.Init("localhost:63433")
}

var dockerClient *docker.Client

var _ = Describe("apparatchik", func() {
	Describe("StartApparatchick()", func() {
		It("Should start apparatchik core", func() {
			apparatchik, err := core.StartApparatchik(dockerClient)
			Expect(err).To(BeNil())
			Expect(apparatchik).NotTo(BeNil())
		})
	})

	Context("When apparatchik is started", func() {

		var apparatchik *core.Apparatchik

		BeforeEach(func() {
			var err error
			apparatchik, err = core.StartApparatchik(dockerClient)
			Expect(err).To(BeNil())
			Expect(apparatchik).NotTo(BeNil())
		})

		AfterEach(func() {
			// TODO: shutdown apparatchik
		})

		Describe("NewApplication()", func() {
			It("Should start a new application", func() {
				status, err := apparatchik.NewApplication("app1", &core.ApplicationConfiguration{
					Goals: map[string]*core.GoalConfiguration{
						"g1": {
							Image:   "alpine:3.2",
							Command: []string{"/bin/sh", "-c", "sleep 0.5; echo executed"},
						},
					},
					MainGoal: "g1",
				})

				Expect(err).To(BeNil())
				Expect(status).NotTo(BeNil())

			})
		})

	})

})
