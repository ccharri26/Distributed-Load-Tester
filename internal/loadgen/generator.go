package loadgen

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

// responsibilities: create go routines, build request, send request, collect metrics, report metrics

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

// ig that each worker can pass a context to the generator for sending requests
func NewGenerator(testSpec spec.TestSpec, testClient *http.Client) *Generator {
	if testClient == nil {
		panic("Invalid client passed into generator")
	}

	return &Generator{
		spec:   testSpec,
		client: testClient,
	}
}

func (g *Generator) RunTest() error {

	return nil
}

func (g *Generator) SendRequest(ctx context.Context) RequestResult {
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	req, err := http.NewRequestWithContext(ctx, g.spec.Method, g.spec.TargetURL, strings.NewReader(g.spec.Body))
	if err != nil {
		return RequestResult{
			Success: false,
			Err:     err,
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
