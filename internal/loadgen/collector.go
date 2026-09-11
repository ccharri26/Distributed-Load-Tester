package loadgen

import (
	"slices"
	"time"
)

type Collector struct {
	summary   Summary
	durations []time.Duration
}

func (c *Collector) RecordStart() {
	c.summary.StartedRequests++
}

func (c *Collector) RecordResult(result RequestResult) {
	c.summary.CompletedRequests++

	if result.Success {
		c.summary.SuccessfulRequests++
	} else {
		c.summary.FailedRequests++
	}

	c.durations = append(c.durations, result.Duration)
}

func (c *Collector) CompileResults() {
	slices.Sort(c.durations)
	c.summary.MinRequestDuration = c.durations[0]
	c.summary.MaxRequestDuration = c.durations[len(c.durations)-1]

	var durationSum time.Duration
	for _, value := range c.durations {
		durationSum += value
	}

	c.summary.AverageRequestDuration = durationSum / time.Duration(len(c.durations))

	c.summary.P50 = Percentile(c.durations, 50)
	c.summary.P50 = Percentile(c.durations, 90)
	c.summary.P50 = Percentile(c.durations, 95)
}

func Percentile(durations []time.Duration, p float64) float64 {
	index := int(float64(len(durations)) * (p / 100.00))
	if index >= len(durations) {
		index = len(durations) - 1
	}

	if index < 0 {
		index = 0
	}

	return float64(durations[index])
}
