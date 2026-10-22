// Package orchestrator coordinates a load-test run
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

type Orchestrator struct {
	provisioner WorkerProvisioner
}

func New(provisioner WorkerProvisioner) Orchestrator {
	return Orchestrator{provisioner: provisioner}
}

// LoadTestSpec reads a JSON file and uses shared test spec package to validate
func (Orchestrator) LoadTestSpec(ctx context.Context, path string) (*spec.TestSpec, error) {
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
func (Orchestrator) CreateWorkerAssignments(testSpec spec.TestSpec) ([]WorkerAssignment, error) {
	if err := testSpec.Validate(); err != nil {
		return nil, err
	}

	assignments := make([]WorkerAssignment, testSpec.Workers)
	if testSpec.RPS < testSpec.Workers {
		return nil, fmt.Errorf("RPS must be greater than the number of workers")
	} // log error for now but possibly find a better way to handle this case
	// Possibly if the RPS is less than the number of workers, default to one worker?
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
			WorkerID:    fmt.Sprintf("worker-%d", index+1),
			WorkerRPS:   workerSpec.RPS,
			TargetUrl:   workerSpec.TargetURL,
			Method:      workerSpec.Method,
			Headers:     workerSpec.Headers,
			Body:        workerSpec.Body,
			Concurrency: workerSpec.Concurrency,
			Duration:    workerSpec.Duration.Duration,
		}
	}

	return assignments, nil
}

func (o Orchestrator) Run(ctx context.Context, path string) ([]WorkerResult, error) {

	if o.provisioner == nil {
		return nil, fmt.Errorf("worker provisioner is required")
	}

	spec, err := o.LoadTestSpec(ctx, path)
	if err != nil {
		return nil, err
	}

	assignments, err := o.CreateWorkerAssignments(*spec)
	if err != nil {
		return nil, err
	}

	results := make([]WorkerResult, len(assignments))
	workerErrors := make(chan error, len(assignments))

	var wg sync.WaitGroup

	for i, assignment := range assignments {
		wg.Add(1)

		go func(i int, assignment WorkerAssignment) {
			defer wg.Done()

			result, err := o.provisioner.RunWorker(ctx, assignment)
			if err != nil {
				workerErr := fmt.Errorf("worker: %s: %w", assignment.WorkerID, err)
				results[i] = WorkerResult{
					WorkerID: assignment.WorkerID,
					Error:    workerErr,
				}
				workerErrors <- workerErr
				return
			}

			results[i] = result
		}(i, assignment)
	}

	wg.Wait()
	close(workerErrors)

	var errs []error
	for err := range workerErrors {
		errs = append(errs, err)
	}

	return results, errors.Join(errs...)
}
