package stats

import (
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type ContainerStats struct {
	sync.Mutex
	trackers map[string]*Tracker
	duration time.Duration
}

func NewContainerStats(duration time.Duration) *ContainerStats {
	stats := &ContainerStats{
		trackers: map[string]*Tracker{},
		duration: duration,
	}

	// event.DockerEvents.On("event", stats.OnDockerEvent)
	// event.ContainerStats.On("stats", stats.OnStats)

	return stats
}

func (c *ContainerStats) LastStats(containerID string) Entry {
	c.Lock()
	defer c.Unlock()
	tracker, found := c.trackers[containerID]
	if !found {
		return Entry{}
	}
	return tracker.LastEntry()
}

func (c *ContainerStats) CurrentStats(containerID string) []Entry {
	c.Lock()
	defer c.Unlock()
	tracker, found := c.trackers[containerID]
	if !found {
		return []Entry{}
	}
	return tracker.Entries()
}

func (c *ContainerStats) OnDockerEvent(evt *docker.APIEvents) {
	c.Lock()
	defer c.Unlock()

	switch evt.Status {
	case "create":
		c.trackers[evt.ID] = NewTracker(c.duration)
	case "destroy":
		delete(c.trackers, evt.ID)
	}

}

// func (c *ContainerStats) OnStats(stats event.StatsForContainer) {
// 	c.Lock()
// 	defer c.Unlock()
// 	tracker, found := c.trackers[stats.ContainerID]
// 	if !found {
// 		tracker = NewTracker(c.duration)
// 		c.trackers[stats.ContainerID] = tracker
// 	}
//
// 	tracker.Add(Entry{
// 		Time:   stats.Stats.Read,
// 		CPU:    (stats.Stats.CPUStats.CPUUsage.TotalUsage - stats.Stats.PreCPUStats.CPUUsage.TotalUsage),
// 		Memory: stats.Stats.MemoryStats.Usage,
// 	})
// }
