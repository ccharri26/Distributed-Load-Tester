package orchestrator

import (
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

// WorkerAssignment is the configuration supplied to a worker.
type WorkerAssignment struct {
	WorkerID string        `json:"worker_id"`
	RPS      int           `json:"rps"`
	TestSpec spec.TestSpec `json:"test_spec"`
}

// WorkerResult contains the outcome reported by one worker after a test run.
type WorkerResult struct {
	WorkerID       string        `json:"worker_id"`
	RequestsSent   int           `json:"requests_sent"`
	Successful     int           `json:"successful"`
	Failed         int           `json:"failed"`
	AverageLatency time.Duration `json:"average_latency"`
	Error          error         `json:"-"`
}
