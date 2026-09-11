package loadgen

import (
	"time"
)

type Summary struct {
	ScheduledRequests      int
	StartedRequests        int
	CompletedRequests      int
	SuccessfulRequests     int
	FailedRequests         int
	MinRequestDuration     time.Duration
	MaxRequestDuration     time.Duration
	AverageRequestDuration time.Duration
	P99                    float64
	P95                    float64
	P50                    float64
}
