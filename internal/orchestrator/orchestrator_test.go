package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

func TestLoadTestSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.json")
	data := []byte(`{"target_url":"https://example.com","duration":"1s","rps":5,"workers":1}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	testSpec, err := New(nil).LoadTestSpec(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if testSpec.Method != "GET" || testSpec.Concurrency != 10 {
		t.Fatalf("defaults were not applied: %+v", testSpec)
	}
	t.Logf("parsed test spec: %+v", testSpec)
}

func TestCreateWorkerAssignmentsSplitsRPSExactly(t *testing.T) {
	testSpec := spec.TestSpec{
		TargetURL:   "https://example.com",
		Method:      "GET",
		Duration:    spec.Duration{Duration: time.Second},
		RPS:         10,
		Concurrency: 1,
		Workers:     3,
	}

	assignments, err := New(nil).CreateWorkerAssignments(testSpec)
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 3 {
		t.Fatalf("assignment count = %d, want 3", len(assignments))
	}
	if got := []int{assignments[0].RPS, assignments[1].RPS, assignments[2].RPS}; got[0] != 4 || got[1] != 3 || got[2] != 3 {
		t.Fatalf("rates = %v, want [4 3 3]", got)
	}
	for _, assignment := range assignments {
		t.Logf("worker assignment: id=%s rps=%d", assignment.WorkerID, assignment.RPS)
	}
}

type fakeProvisioner struct {
	started chan struct{}
	release chan struct{}
}

func (f *fakeProvisioner) RunWorker(
	_ context.Context,
	assignment WorkerAssignment,
) (WorkerResult, error) {
	f.started <- struct{}{}
	<-f.release

	return WorkerResult{
		WorkerID: assignment.WorkerID,
	}, nil
}

func TestRunStartsWorkersConcurrently(t *testing.T) {
	const workerCount = 3

	fake := &fakeProvisioner{started: make(chan struct{}, workerCount), release: make(chan struct{})}

	service := Service{
		provisioner: fake,
	}

	testSpec := spec.TestSpec{
		TargetURL:   "https://example.com",
		Method:      "GET",
		Duration:    spec.Duration{Duration: time.Second},
		RPS:         10,
		Concurrency: 1,
		Workers:     workerCount,
	}

	done := make(chan struct{})
	var (
		results []WorkerResult
		err     error
	)

	go func() {
		results, err = service.Run(context.Background(), testSpec)
		close(done)
	}()

	for range workerCount {
		select {
		case <-fake.started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start concurrently")
		}
	}

	close(fake.release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not finish")
	}

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(results) != workerCount {
		t.Fatalf("got %d results, want %d", len(results), workerCount)
	}
}
