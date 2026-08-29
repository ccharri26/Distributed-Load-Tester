package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ccharri26/Distributed-Load-Tester/internal/orchestrator"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type DockerProvisioner struct {
	WorkerImage string
	client      client.APIClient
}

func NewDocker(workerImage string) (*DockerProvisioner, error) {
	dockerClient, err := client.New(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)

	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}

	return &DockerProvisioner{
		WorkerImage: workerImage,
		client:      dockerClient,
	}, nil
}

func (p *DockerProvisioner) RunWorker(ctx context.Context, assignment orchestrator.WorkerAssignment) (orchestrator.WorkerResult, error) {
	assignmentJSON, err := json.Marshal((assignment))
	if err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf("marhsal assignment: %w", err)
	}

	// Creates container with the LOAD_TEST_ASSIGNMENT in env of container.
	created, err := p.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image: p.WorkerImage,
		Config: &container.Config{
			Env: []string{
				"LOAD_TEST_ASSIGNMENT=" + string(assignmentJSON),
			},
		},
	})

	if err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf("create worker %q: %w", assignment.WorkerID, err)
	}

	defer p.client.ContainerRemove(
		context.Background(),
		created.ID,
		client.ContainerRemoveOptions{Force: true},
	)

	if _, err := p.client.ContainerStart(
		ctx,
		created.ID,
		client.ContainerStartOptions{},
	); err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf(
			"start worker %q: %w",
			assignment.WorkerID,
			err,
		)
	}

	wait := p.client.ContainerWait(
		ctx,
		created.ID,
		client.ContainerWaitOptions{},
	)

	select {
	case err := <-wait.Error:
		if err != nil {
			return orchestrator.WorkerResult{}, fmt.Errorf(
				"wait for worker %q: %w",
				assignment.WorkerID,
				err,
			)
		}

	case status := <-wait.Result:
		if status.StatusCode != 0 {
			return orchestrator.WorkerResult{}, fmt.Errorf(
				"worker %q exited with status %d",
				assignment.WorkerID,
				status.StatusCode,
			)
		}
	}

	logs, err := p.client.ContainerLogs(ctx, created.ID, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})

	if err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf("read worker %q logs: %w", assignment.WorkerID, err)
	}

	defer logs.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logs); err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf(
			"copy worker %q logs: %w",
			assignment.WorkerID,
			err,
		)
	}

	var result orchestrator.WorkerResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return orchestrator.WorkerResult{}, fmt.Errorf(
			"decode worker %q result: %w; stderr: %s",
			assignment.WorkerID,
			err,
			stderr.String(),
		)
	}

	return result, nil
}
