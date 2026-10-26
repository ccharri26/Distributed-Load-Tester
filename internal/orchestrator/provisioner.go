package orchestrator

import "context"

type WorkerProvisioner interface {
	RunWorker(context.Context, WorkerAssignment) (WorkerResult, error)
}
