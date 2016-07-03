package util

import (
	"fmt"
	"strings"

	"github.com/netice9/apparatchik/apparatchik/core"
)

func TimeSeriesToLine(samples []core.Sample, width, height int) string {
	if len(samples) == 0 {
		return ""
	}
	// sample := samples[0]

	minValue := samples[0].Value
	maxValue := samples[0].Value
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
		normalisedTime := float64(sample.Time.UnixNano()-minTime.UnixNano()) / float64(maxTime.UnixNano()-minTime.UnixNano())
		scaledTime := int(normalisedTime * float64(width))

		normalisedValue := 1.0 - float64(sample.Value-minValue)/float64(maxValue-minValue)
		scaledValue := int(normalisedValue * float64(height))
		// normalised := append(normalised, []float64{normalisedTime, normalisedValue})
		points = append(points, fmt.Sprintf("%d,%d", scaledTime, scaledValue))
	}

	return strings.Join(points, " ")

}
