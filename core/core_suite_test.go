package core_test

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
	"github.com/netice9/apparatchik/core"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
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

		waitForGoalStatus := func(app, goalName, status string) {
			for {
				appStatus, err := apparatchik.ApplicationStatus(app)
				Expect(err).To(BeNil())
				goal, found := appStatus.Goals[goalName]

				fmt.Println(found, goal, status)

				if found && goal.Status == status {
					return
				}

				time.Sleep(10 * time.Millisecond)
			}
		}

		AfterEach(apparatchik.Stop)

		XDescribe("NewApplication()", func() {
			It("Should start and execute a new application", func(done Done) {
				status, err := apparatchik.NewApplication("app1", &core.ApplicationConfiguration{
					Goals: map[string]*core.GoalConfiguration{
						"g1": {
							Image:      "alpine:3.4",
							Command:    []string{"sleep 0.1; echo executed"},
							Entrypoint: []string{"/bin/sh", "-c"},
						},
					},
					MainGoal: "g1",
				})

				Expect(err).To(BeNil())
				Expect(status).NotTo(BeNil())

				Expect(status.MainGoal).To(Equal("g1"))
				goalStatus := status.Goals["g1"]
				Expect(goalStatus).NotTo(BeNil())
				Expect(goalStatus.Status).To(Equal("fetching_image"))
				waitForGoalStatus("app1", "g1", "terminated")
				close(done)
			}, 1)
		})

	})

})
