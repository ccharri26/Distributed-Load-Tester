package orchestrator

import (
	"time"
)

// WorkerAssignment is the configuration supplied to a worker.
type WorkerAssignment struct {
	WorkerID    string            `json:"worker_id"`
	WorkerRPS   int               `json:"WORKER_RPS"`
	TargetUrl   string            `json:"TARGET_URL"`
	Method      string            `json:"METHOD"`
	Headers     map[string]string `json:"HEADERS"`
	Body        string            `json:"BODY"`
	Duration    time.Duration     `json:"DURATION"`
	Concurrency int               `json:"CONCURRENCY"`
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
