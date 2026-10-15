package loadgen

import (
	"slices"
	"sync"
	"time"
)

type Collector struct {
	mutex     sync.Mutex
	summary   Summary
	durations []time.Duration
}

func (c *Collector) RecordStart() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.summary.StartedRequests++
	c.summary.ScheduledRequests++
}

func (c *Collector) RecordResult(result RequestResult) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.summary.StartedRequests == 0 {
		return
	}

	c.summary.CompletedRequests++

	if result.Success {
		c.summary.SuccessfulRequests++
	} else {
		c.summary.FailedRequests++
	}

	if result.Duration >= 0 {
		c.durations = append(c.durations, result.Duration)
	}
}

func (c *Collector) RecordLateResult(result RequestResult) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.summary.StartedRequests == 0 {
		return
	}

	c.summary.LateCompletedRequests++

	if result.Success {
		c.summary.SuccessfulRequests++
	} else {
		c.summary.FailedRequests++
	}
}

// Results that arrive after the duration of the test is completed will not be counted in latency metrics.
func (c *Collector) CompileResults() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.durations) == 0 {
		return
	}

	slices.Sort(c.durations)
	c.summary.MinRequestDuration = c.durations[0]
	c.summary.MaxRequestDuration = c.durations[len(c.durations)-1]

	average := c.durations[0]
	for i := 1; i < len(c.durations); i++ {
		count := time.Duration(i + 1)
		average += (c.durations[i] - average) / count
	}
	c.summary.AverageRequestDuration = average

	c.summary.P50 = Percentile(c.durations, 50)
	c.summary.P95 = Percentile(c.durations, 95)
	c.summary.P99 = Percentile(c.durations, 99)
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
