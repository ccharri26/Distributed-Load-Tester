package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

func main() {
	targetURL := flag.String(
		"url",
		"",
		"Target URL to load test",
	)

	method := flag.String(
		"method",
		"GET",
		"HTTP method",
	)

	rps := flag.Int(
		"rps",
		0,
		"Requests per second",
	)

	duration := flag.Duration(
		"duration",
		0,
		"How long to run the test",
	)

	concurrency := flag.Int(
		"concurrency",
		10,
		"Maximum concurrent requests",
	)

	flag.Parse()

	testSpec := spec.TestSpec{
		TargetURL:   *targetURL,
		Method:      *method,
		RPS:         *rps,
		Duration:    spec.Duration{Duration: *duration},
		Concurrency: *concurrency,
	}

	testSpec.ApplyDefaults()

	if err := testSpec.Validate(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", testSpec)
}
