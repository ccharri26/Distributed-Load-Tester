package provisioner

import (
	"context"

	"github.com/ccharri26/Distributed-Load-Tester/internal/orchestrator"
)

type DockerProvisioner struct {
	WorkerImage string
	// client *client.Client // add Docker SDK client here later
}

func NewDocker(workerImage string) *DockerProvisioner {
	return &DockerProvisioner{
		WorkerImage: workerImage,
		// Client in the future
	}
}

func (p *DockerProvisioner) RunWorker(ctx context.Context, assignment orchestrator.WorkerAssignment) (orchestrator.WorkerResult, error) {

	return orchestrator.WorkerResult{}, nil
}
