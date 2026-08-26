// Package orchestrator coordinates a load-test run. Worker provisioning and
// execution will be added behind this boundary later.
package orchestrator

import (
	"context"
	"fmt"
	"os"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

type Orchestrator interface {
	LoadTestSpec(context.Context, string) (*spec.TestSpec, error)
	CreateWorkerAssignments(spec.TestSpec) ([]WorkerAssignment, error)
}

type Service struct{}

// WorkerAssignment is the configuration supplied to a worker
type WorkerAssignment struct {
	WorkerID string        `json:"worker_id"`
	RPS      int           `json:"rps"`
	TestSpec spec.TestSpec `json:"test_spec"`
}

func New() Orchestrator {
	return Service{}
}

// LoadTestSpec reads a JSON file and uses shared test spec package to validate
func (Service) LoadTestSpec(ctx context.Context, path string) (*spec.TestSpec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read test spec %q: %w", path, err)
	}
	return spec.Parse(data)
}

// Creates Worker Assignments, distributes the RPS evenly with remainder given to the earliest created workers
func (Service) CreateWorkerAssignments(testSpec spec.TestSpec) ([]WorkerAssignment, error) {
	if err := testSpec.Validate(); err != nil {
		return nil, err
	}

	assignments := make([]WorkerAssignment, testSpec.Workers)
	baseRPS := testSpec.RPS / testSpec.Workers
	remainder := testSpec.RPS % testSpec.Workers

	for index := range assignments {
		workerRPS := baseRPS
		if index < remainder {
			workerRPS++
		}

		workerSpec := testSpec
		workerSpec.RPS = workerRPS
		assignments[index] = WorkerAssignment{
			WorkerID: fmt.Sprintf("worker-%d", index+1),
			RPS:      workerRPS,
			TestSpec: workerSpec,
		}
	}

	return assignments, nil
}
