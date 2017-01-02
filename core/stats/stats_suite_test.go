package stats_test

import (
	"time"

	"github.com/netice9/apparatchik/core/stats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stats Suite")
}

var zeroTime = time.Now()

var _ = Describe("Tracker", func() {
	var tracker *stats.Tracker

	BeforeEach(func() {
		tracker = stats.NewTracker(time.Second)
	})

	Describe("Add()", func() {
		Context("When tracker is empty", func() {
			Context("When I add a new entry", func() {
				var entry stats.Entry
				BeforeEach(func() {
					entry = stats.Entry{Time: zeroTime, CPU: 0.0, Memory: 0}
					tracker.Add(entry)
				})
				It("Should contain only that entry", func() {
					Expect(tracker.Entries()).To(Equal([]stats.Entry{entry}))
				})
				Context("When I add an entry that has time after the entry within the duration", func() {
					var secondEntry stats.Entry
					BeforeEach(func() {
						secondEntry = stats.Entry{Time: zeroTime.Add(time.Millisecond), CPU: 0.0, Memory: 0}
						tracker.Add(secondEntry)
					})
					It("Should contain both entries in the chronological order", func() {
						Expect(tracker.Entries()).To(Equal([]stats.Entry{entry, secondEntry}))
					})

				})
				Context("When I add an entry that has time after the entry within the duration", func() {
					var secondEntry stats.Entry
					BeforeEach(func() {
						secondEntry = stats.Entry{Time: zeroTime.Add(time.Millisecond + time.Second), CPU: 0.0, Memory: 0}
						tracker.Add(secondEntry)
					})
					It("Should contain only the new entry", func() {
						Expect(tracker.Entries()).To(Equal([]stats.Entry{secondEntry}))
					})

				})
				Context("When I add an entry that has time before the entry within the duration", func() {
					var secondEntry stats.Entry
					BeforeEach(func() {
						secondEntry = stats.Entry{Time: zeroTime.Add(-time.Millisecond), CPU: 0.0, Memory: 0}
						tracker.Add(secondEntry)
					})
					It("Should contain both entries in the chronological order", func() {
						Expect(tracker.Entries()).To(Equal([]stats.Entry{secondEntry, entry}))
					})

				})
			})
		})

	})

})
