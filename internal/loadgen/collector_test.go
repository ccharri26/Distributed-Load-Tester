package loadgen

import (
	"testing"
	"time"
)

func TestCollectorCompilesSuccessfulResults(t *testing.T) {
	// Arrange: create the collector and provide two completed requests.
	collector := Collector{}
	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
	})

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 300 * time.Millisecond,
	})

	// Act: calculate the summary from the recorded results.
	collector.CompileResults()

	// Assert: verify the collector tracked the requests and latency correctly.
	got := collector.summary

	if got.ScheduledRequests != 2 {
		t.Errorf("ScheduledRequests = %d, want 2", got.ScheduledRequests)
	}
	if got.StartedRequests != 2 {
		t.Errorf("StartedRequests = %d, want 2", got.StartedRequests)
	}
	if got.CompletedRequests != 2 {
		t.Errorf("CompletedRequests = %d, want 2", got.CompletedRequests)
	}
	if got.SuccessfulRequests != 2 {
		t.Errorf("SuccessfulRequests = %d, want 2", got.SuccessfulRequests)
	}
	if got.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0", got.FailedRequests)
	}
	if got.MinRequestDuration != 100*time.Millisecond {
		t.Errorf("MinRequestDuration = %v, want 100ms", got.MinRequestDuration)
	}
	if got.MaxRequestDuration != 300*time.Millisecond {
		t.Errorf("MaxRequestDuration = %v, want 300ms", got.MaxRequestDuration)
	}
	if got.AverageRequestDuration != 200*time.Millisecond {
		t.Errorf("AverageRequestDuration = %v, want 200ms", got.AverageRequestDuration)
	}
}

func TestCollectorCompilesFailedResults(t *testing.T) {
	// Arrange
	collector := Collector{}

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  false,
		Duration: 100 * time.Millisecond,
	})

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  false,
		Duration: 300 * time.Millisecond,
	})

	// Act
	collector.CompileResults()

	// Assert
	got := collector.summary

	if got.ScheduledRequests != 2 {
		t.Errorf("ScheduledRequests = %d, want 2", got.ScheduledRequests)
	}

	if got.StartedRequests != 2 {
		t.Errorf("StartedRequests = %d, want 2", got.StartedRequests)
	}

	if got.CompletedRequests != 2 {
		t.Errorf("CompletedRequests = %d, want 2", got.CompletedRequests)
	}

	if got.SuccessfulRequests != 0 {
		t.Errorf("SuccessfulRequests = %d, want 0", got.SuccessfulRequests)
	}

	if got.FailedRequests != 2 {
		t.Errorf("FailedRequests = %d, want 2", got.FailedRequests)
	}

	if got.MinRequestDuration != 100*time.Millisecond {
		t.Errorf("MinRequestDuration = %v, want 100ms", got.MinRequestDuration)
	}

	if got.MaxRequestDuration != 300*time.Millisecond {
		t.Errorf("MaxRequestDuration = %v, want 300ms", got.MaxRequestDuration)
	}

	if got.AverageRequestDuration != 200*time.Millisecond {
		t.Errorf("AverageRequestDuration = %v, want 200ms", got.AverageRequestDuration)
	}
}

func TestCollectorCompilesLatencyPercentiles(t *testing.T) {
	// Arrange: add four requests with deliberately different durations.
	collector := Collector{}
	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
	}

	for _, duration := range durations {
		collector.RecordStart()
		collector.RecordResult(RequestResult{
			Success:  true,
			Duration: duration,
		})
	}

	// Act
	collector.CompileResults()

	// Assert: these values match the current index-based Percentile function.
	got := collector.summary
	if got.P50 != float64(300*time.Millisecond) {
		t.Errorf("P50 = %v, want 300ms", time.Duration(got.P50))
	}
	if got.P95 != float64(400*time.Millisecond) {
		t.Errorf("P95 = %v, want 400ms", time.Duration(got.P95))
	}
	if got.P99 != float64(400*time.Millisecond) {
		t.Errorf("P99 = %v, want 400ms", time.Duration(got.P99))
	}
}

func TestEmptyCollectorCompilesWithoutLatencyMetrics(t *testing.T) {
	// Arrange
	collector := Collector{}

	// Act
	collector.CompileResults()

	// Assert: an empty collector should not panic or invent metrics.
	got := collector.summary
	if got.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0", got.CompletedRequests)
	}
	if got.MinRequestDuration != 0 {
		t.Errorf("MinRequestDuration = %v, want 0", got.MinRequestDuration)
	}
	if got.MaxRequestDuration != 0 {
		t.Errorf("MaxRequestDuration = %v, want 0", got.MaxRequestDuration)
	}
	if got.AverageRequestDuration != 0 {
		t.Errorf("AverageRequestDuration = %v, want 0", got.AverageRequestDuration)
	}
}

func TestCollectorIgnoresNegativeDurations(t *testing.T) {
	// Arrange: negative durations represent a request that never produced
	// a usable latency measurement.
	collector := Collector{}
	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  false,
		Duration: -1 * time.Second,
	})

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 200 * time.Millisecond,
	})

	// Act
	collector.CompileResults()

	// Assert: the request still counts, but its negative duration is excluded
	// from latency calculations.
	got := collector.summary
	if got.CompletedRequests != 2 {
		t.Errorf("CompletedRequests = %d, want 2", got.CompletedRequests)
	}
	if got.SuccessfulRequests != 1 {
		t.Errorf("SuccessfulRequests = %d, want 1", got.SuccessfulRequests)
	}
	if got.FailedRequests != 1 {
		t.Errorf("FailedRequests = %d, want 1", got.FailedRequests)
	}
	if got.MinRequestDuration != 200*time.Millisecond {
		t.Errorf("MinRequestDuration = %v, want 200ms", got.MinRequestDuration)
	}
	if got.MaxRequestDuration != 200*time.Millisecond {
		t.Errorf("MaxRequestDuration = %v, want 200ms", got.MaxRequestDuration)
	}
	if got.AverageRequestDuration != 200*time.Millisecond {
		t.Errorf("AverageRequestDuration = %v, want 200ms", got.AverageRequestDuration)
	}
}

func TestCollectorIncludesZeroDuration(t *testing.T) {
	// Arrange
	collector := Collector{}
	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 0,
	})

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
	})

	// Act
	collector.CompileResults()

	// Assert: zero is a valid non-negative duration and is included.
	got := collector.summary
	if got.MinRequestDuration != 0 {
		t.Errorf("MinRequestDuration = %v, want 0", got.MinRequestDuration)
	}
	if got.MaxRequestDuration != 100*time.Millisecond {
		t.Errorf("MaxRequestDuration = %v, want 100ms", got.MaxRequestDuration)
	}
	if got.AverageRequestDuration != 50*time.Millisecond {
		t.Errorf("AverageRequestDuration = %v, want 50ms", got.AverageRequestDuration)
	}
}

func TestCollectorCompileResultsCanBeCalledMoreThanOnce(t *testing.T) {
	// Arrange
	collector := Collector{}
	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
	})

	// Act
	collector.CompileResults()
	firstSummary := collector.summary

	collector.CompileResults()
	secondSummary := collector.summary

	// Assert: compiling again should not change the existing summary.
	if firstSummary != secondSummary {
		t.Errorf("summary changed after compiling twice: first = %+v, second = %+v", firstSummary, secondSummary)
	}
}

func TestCollectorRecordResultWithoutRecordStart(t *testing.T) {
	// Arrange
	collector := Collector{}

	// Act: record a result without recording a scheduled or started request.
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
	})

	// Assert: the result is still counted as completed, but no request is
	// counted as scheduled or started.
	if collector.summary.ScheduledRequests != 0 {
		t.Errorf("ScheduledRequests = %d, want 0", collector.summary.ScheduledRequests)
	}
	if collector.summary.StartedRequests != 0 {
		t.Errorf("StartedRequests = %d, want 0", collector.summary.StartedRequests)
	}
	if collector.summary.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0", collector.summary.CompletedRequests)
	}
	if collector.summary.SuccessfulRequests != 0 {
		t.Errorf("SuccessfulRequests = %d, want 0", collector.summary.SuccessfulRequests)
	}
}

func TestCollectorHandlesLargeDurations(t *testing.T) {
	// Arrange: use the largest duration representable by time.Duration twice.
	maxDuration := time.Duration(1<<63 - 1)
	collector := Collector{}

	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: maxDuration,
	})
	collector.RecordStart()
	collector.RecordResult(RequestResult{
		Success:  true,
		Duration: maxDuration,
	})

	// Act
	collector.CompileResults()

	// Assert: averaging two maximum durations should remain within the valid
	// time.Duration range instead of overflowing during the sum.
	if collector.summary.AverageRequestDuration != maxDuration {
		t.Errorf("AverageRequestDuration = %v, want %v", collector.summary.AverageRequestDuration, maxDuration)
	}
}
