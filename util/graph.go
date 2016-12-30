package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/netice9/apparatchik/core"
)

func TimeSeriesToLine(samples []core.Sample, width, height int, lowestMax uint64) (string, uint64) {
	if len(samples) == 0 {
		return "", 0
	}
	// sample := samples[0]

	minValue := uint64(0)
	maxValue := lowestMax
	minTime := samples[0].Time
	maxTime := samples[0].Time

	for _, sample := range samples {
		if minValue > sample.Value {
			minValue = sample.Value
		}
		if maxValue < sample.Value {
			maxValue = sample.Value
		}
		if minTime.After(sample.Time) {
			minTime = sample.Time
		}
		if maxTime.Before(sample.Time) {
			maxTime = sample.Time
		}
	}

	// normalised := [][]float64{}

	points := []string{}

	for _, sample := range samples {
		// normalisedTime := float64(sample.Time.UnixNano()-minTime.UnixNano()) / float64(maxTime.UnixNano()-minTime.UnixNano())
		normalisedTime := float64(sample.Time.UnixNano()-minTime.UnixNano()) / float64((time.Second * 120).Nanoseconds())

		scaledTime := int(normalisedTime * float64(width))

		normalisedValue := 1.0 - float64(sample.Value-minValue)/float64(maxValue-minValue)
		scaledValue := int(normalisedValue * float64(height))
		// normalised := append(normalised, []float64{normalisedTime, normalisedValue})
		points = append(points, fmt.Sprintf("%d,%d", scaledTime, scaledValue))
	}

	return strings.Join(points, " "), maxValue

}
