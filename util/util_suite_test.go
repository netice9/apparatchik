package util_test

import (
	"time"

	"github.com/netice9/apparatchik/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util Suite")
}

var _ = Describe("OutputTracker", func() {

	var tracker *util.OutputTracker
	BeforeEach(func() {
		tracker = util.NewOutputTracker(10)
	})

	Describe("Write()", func() {

		Context("When tracker is Closed", func() {
			BeforeEach(func() {
				tracker.Close()
			})

			It("should return error", func() {
				_, err := tracker.Write([]byte("x"))
				Expect(err).NotTo(BeNil())
			})
		})

		Context("When I have a listener added", func() {
			var listeningChannel chan []string
			BeforeEach(func() {
				listeningChannel = tracker.AddListener(1)
			})

			Context("When tracker is closed", func() {
				BeforeEach(func() {
					tracker.Close()
				})

				It("should close the notification channel", func(done Done) {
					_, open := <-listeningChannel
					Expect(open).To(BeFalse())
					close(done)
				})
			})

			Context("When I write a line to the tracker", func() {
				BeforeEach(func() {
					n, err := tracker.Write([]byte("1\n"))
					Expect(err).To(BeNil())
					Expect(n).NotTo(Equal(0))
					time.Sleep(20 * time.Millisecond)
				})

				It("should send notification to the listener", func(done Done) {
					notification := <-listeningChannel
					Expect(notification).To(Equal([]string{"1"}))
					close(done)
				})

			})
		})

		Context("When the tracker already has max lines", func() {
			BeforeEach(func() {
				n, err := tracker.Write([]byte("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"))
				Expect(err).To(BeNil())
				Expect(n).NotTo(Equal(0))
			})

			Context("when I write another line to it", func() {
				BeforeEach(func() {
					n, err := tracker.Write([]byte("11\n"))
					Expect(err).To(BeNil())
					Expect(n).NotTo(Equal(0))

				})
				It("should drop the first line", func() {
					Expect(tracker.Lines).To(Equal([]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}))
				})
			})

		})

		Context("When I write more than one line to the tracker", func() {
			BeforeEach(func() {
				n, err := tracker.Write([]byte("1\n2\n3\n"))
				Expect(err).To(BeNil())
				Expect(n).NotTo(Equal(0))
			})

			It("should contain all lines", func() {
				Expect(tracker.Lines).To(Equal([]string{"1", "2", "3"}))
			})
		})
		Context("When I write one line to the tracker", func() {
			BeforeEach(func() {
				n, err := tracker.Write([]byte("this is a test\n"))
				Expect(err).To(BeNil())
				Expect(n).NotTo(Equal(0))
			})

			It("should contain one line", func() {
				Expect(tracker.Lines).To(Equal([]string{"this is a test"}))
			})

			Context("And I write another line to the tracker", func() {
				BeforeEach(func() {
					n, err := tracker.Write([]byte("this is another line\n"))
					Expect(err).To(BeNil())
					Expect(n).NotTo(Equal(0))
				})
				It("should contain two lines", func() {
					Expect(tracker.Lines).To(Equal([]string{"this is a test", "this is another line"}))
				})
			})
		})
	})
})
