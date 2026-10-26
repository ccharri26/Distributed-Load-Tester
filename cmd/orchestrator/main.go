package main

import (
	"context"
	"log"

	"github.com/ccharri26/Distributed-Load-Tester/internal/orchestrator"
	"github.com/ccharri26/Distributed-Load-Tester/internal/provisioner"
)

func main() {
	ctx := context.Background()

	workerProvisioner, err := provisioner.NewDocker("load-tester-worker:test")
	if err != nil {
		log.Fatalf("failed to create Docker provisioner: %v", err)
	}

	service := orchestrator.New(workerProvisioner)

	results, err := service.Run(ctx, "test-specs/test1.json")
	if err != nil {
		log.Fatalf("failed to run load test: %v", err)
	}

	log.Printf("Load test results: %+v", results)
}
