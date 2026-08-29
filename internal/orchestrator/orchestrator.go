// Package orchestrator coordinates a load-test run. Worker provisioning and
// execution will be added behind this boundary later.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

type Orchestrator interface {
	LoadTestSpec(context.Context, string) (*spec.TestSpec, error)
	CreateWorkerAssignments(spec.TestSpec) ([]WorkerAssignment, error)
	Run(context.Context, spec.TestSpec) ([]WorkerResult, error)
}

type Service struct{ provisioner WorkerProvisioner }

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

// CreateWorkerAssignments - distributes the RPS evenly with remainder given to the earliest created workers
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

func (s Service) Run(ctx context.Context, testSpec spec.TestSpec) ([]WorkerResult, error) {
	assignments, err := Service{}.CreateWorkerAssignments(testSpec)
	if err != nil {
		return nil, err
	}

	results := make([]WorkerResult, len(assignments))
	errors := make(chan error, len(assignments))

	var wg sync.WaitGroup

	for i, assignment := range assignments {
		wg.Add(1)

		go func(i int, assignment WorkerAssignment) {
			defer wg.Done()

			result, err := s.provisioner.RunWorker(ctx, assignment)
			if err != nil {
				errors <- err
				return
			}

			results[i] = result
		}(i, assignment)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
