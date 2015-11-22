package main

import (
	"github.com/fsouza/go-dockerclient"
	"time"
)

type StatsRequest struct {
	responseChannel chan *Stats
	since           time.Time
}

type StatsTracker struct {
	stats      Stats
	lastSample *docker.Stats

	containerId string
	client      *docker.Client

	statsRequest chan *StatsRequest
}

type Sample struct {
	Value uint64    `json:"value"`
	Time  time.Time `json:"time"`
}

type Stats struct {
	CpuStats []Sample `json:"cpu_stats"`
	MemStats []Sample `json:"mem_stats"`
}

func NewStatsTracker(containerId string, client *docker.Client) *StatsTracker {

	tracker := StatsTracker{
		containerId:  containerId,
		client:       client,
		statsRequest: make(chan *StatsRequest),
		stats: Stats{
			CpuStats: make([]Sample, 0, 120),
			MemStats: make([]Sample, 0, 120),
		},
	}

	go tracker.Actor()

	return &tracker
}

func (tracker *StatsTracker) Actor() {

	ch := make(chan *docker.Stats)

	go tracker.client.Stats(docker.StatsOptions{
		ID:     tracker.containerId,
		Stats:  ch,
		Stream: true,
	})

	for {
		select {
		case stats, ok := <-ch:

			if ok && stats != nil {
				if tracker.lastSample == nil {
					tracker.lastSample = stats
				} else {
					cpuDiff := stats.CPUStats.CPUUsage.TotalUsage - tracker.lastSample.CPUStats.CPUUsage.TotalUsage
					memory := stats.MemoryStats.Usage

					tracker.stats.CpuStats = append(tracker.stats.CpuStats, Sample{Value: cpuDiff, Time: stats.Read})
					if len(tracker.stats.CpuStats) > 120 {
						tracker.stats.CpuStats = tracker.stats.CpuStats[1:]
					}

					tracker.stats.MemStats = append(tracker.stats.MemStats, Sample{Value: memory, Time: stats.Read})
					if len(tracker.stats.MemStats) > 120 {
						tracker.stats.MemStats = tracker.stats.MemStats[1:]
					}
					tracker.lastSample = stats
				}
			} else if !ok {
				ch = nil
			}
		case statRequest, ok := <-tracker.statsRequest:
			if ok {
				stats := Stats{
					CpuStats: LimitSampleByTime(tracker.stats.CpuStats, statRequest.since),
					MemStats: LimitSampleByTime(tracker.stats.MemStats, statRequest.since),
				}
				statRequest.responseChannel <- &stats
			} else {
				return
			}
		}
	}
}

func LimitSampleByTime(samples []Sample, since time.Time) []Sample {
	result := []Sample{}
	for _, sample := range samples {
		if sample.Time.After(since) {
			result = append(result, sample)
		}
	}
	return result
}

func (tracker *StatsTracker) Close() {
	close(tracker.statsRequest)
}

func (tracker *StatsTracker) CurrentStats(since time.Time) *Stats {
	if tracker != nil {
		responseChan := make(chan *Stats)
		tracker.statsRequest <- &StatsRequest{responseChan, since}
		return <-responseChan
	} else {
		return nil
	}
}

func (tracker *StatsTracker) MomentaryStats() *docker.Stats {
	if tracker != nil {
		return tracker.lastSample
	} else {
		return nil
	}
}
