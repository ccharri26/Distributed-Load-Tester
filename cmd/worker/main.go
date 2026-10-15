package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/loadgen"
	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

func main() {
	// read in test spec from env
	var spec spec.TestSpec

	spec.ID = os.Getenv("SPEC_ID")
	spec.TargetURL = os.Getenv("TARGET_URL")
	spec.Method = os.Getenv("METHOD")
	//spec.Headers = os.Getenv("HEADERS")
	spec.Body = os.Getenv("BODY")
	//testDuration, err := time.ParseDuration(os.Getenv("DURATION"))
	//if err != nil {
	//	fmt.Println("Failed to parse test duration")
	//}

	rpsString := os.Getenv("RPS")
	rps, err := strconv.Atoi(rpsString)
	if err != nil {
		log.Fatal("Failed to parse RPS")
	}
	spec.RPS = rps

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	generator := loadgen.NewGenerator(spec, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := generator.RunTest(ctx)
	if err != nil {
		log.Fatal("Error while running load test")
	}

	res.OutputResults()
}
