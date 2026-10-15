package loadgen

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

const numberOfRoutines = 10

type Generator struct {
	spec   spec.TestSpec
	client *http.Client
}

type RequestResult struct {
	Success    bool
	StatusCode int
	Duration   time.Duration
	Err        error
}

func NewGenerator(testSpec spec.TestSpec, testClient *http.Client) *Generator {
	if testClient == nil {
		panic("Invalid client passed into generator")
	}

	return &Generator{
		spec:   testSpec,
		client: testClient,
	}
}

func (g *Generator) RunTest(ctx context.Context) (Summary, error) {
	if g.spec.RPS <= 0 {
		return Summary{}, errors.New("RPS is invalid")
	}

	if g.spec.Duration.Duration <= 0 {
		return Summary{}, errors.New("Duration is invalid")
	}

	collector := Collector{}

	interval := (1 * time.Second) / time.Duration(g.spec.RPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadlineAt := time.Now().Add(g.spec.Duration.Duration)
	deadline := time.NewTimer(g.spec.Duration.Duration)

	var wg sync.WaitGroup

	maxJobs := int(g.spec.Duration.Duration/interval) + 1
	jobs := make(chan struct{}, maxJobs)

	results := make(chan RequestResult)
	collectorDone := make(chan struct{})

	// drain the results channel so workers are not blocked
	go func() {
		defer close(collectorDone)
		for result := range results {
			if time.Now().Before(deadlineAt) {
				collector.RecordResult(result)
			} else {
				collector.RecordLateResult(result)
			}
		}
	}()

	for i := 0; i < numberOfRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done(): // context cancelled
					return

				case _, ok := <-jobs:
					if !ok { // jobs are finished
						return
					}

					if ctx.Err() != nil {
						return
					}

					collector.RecordStart()
					result := g.SendRequest(ctx)
					results <- result
				}
			}
		}()
	}

	// Feed jobs at the specified rate
	runErr := error(nil)

scheduleLoop:
	for {
		select {
		case <-ctx.Done():
			runErr = ctx.Err()
			break scheduleLoop

		case <-deadline.C:
			break scheduleLoop

		case <-ticker.C:
			if !time.Now().Before(deadlineAt) {
				break scheduleLoop
			}

			select {
			case <-ctx.Done():
				runErr = ctx.Err()
				break scheduleLoop

			case <-deadline.C:
				break scheduleLoop

			case jobs <- struct{}{}:
			}
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	<-collectorDone

	collector.CompileResults()
	return collector.summary, runErr
}

func (g *Generator) SendRequest(ctx context.Context) RequestResult {
	req, err := http.NewRequestWithContext(ctx, g.spec.Method, g.spec.TargetURL, strings.NewReader(g.spec.Body))
	if err != nil {
		return RequestResult{
			Success:  false,
			Err:      err,
			Duration: -1 * time.Second,
		}
	}

	for key, value := range g.spec.Headers {
		req.Header.Set(key, value)
	}

	start := time.Now()
	response, err := g.client.Do(req)
	if err != nil {
		return RequestResult{
			Success:  false,
			Err:      err,
			Duration: time.Since(start),
		}
	}
	defer response.Body.Close()

	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return RequestResult{
			Success:    false,
			StatusCode: response.StatusCode,
			Duration:   time.Since(start),
			Err:        err,
		}
	}
	end := time.Now()

	success := response.StatusCode >= 200 && response.StatusCode < 300

	return RequestResult{
		Success:    success,
		StatusCode: response.StatusCode,
		Duration:   end.Sub(start),
		Err:        nil,
	}
}
