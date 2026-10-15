package loadgen

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ccharri26/Distributed-Load-Tester/internal/spec"
)

// testServer wraps an httptest.Server with the handler and what the handler
// observed. Tests assert against observed to verify the generator built the
// request the spec described.
type testServer struct {
	*httptest.Server
	observed *observedRequest
}

type observedRequest struct {
	method string
	header http.Header
	body   string
}

func newTestServer(t *testing.T, status int) *testServer {
	t.Helper()

	obs := &observedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		obs.method = r.Method
		obs.header = r.Header.Clone()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("server: read body: %v", err)
		}
		obs.body = string(body)

		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	return &testServer{Server: srv, observed: obs}
}

// newGenerator wires a Generator pointed at the test server using a spec
// that has the given method and optional headers/body. Only fields the test
// cares about are set; everything else uses the spec's zero values.
func newGenerator(t *testing.T, srv *testServer, method string, headers map[string]string, body string) *Generator {
	t.Helper()

	return NewGenerator(spec.TestSpec{
		TargetURL: srv.URL,
		Method:    method,
		Headers:   headers,
		Body:      body,
	}, srv.Client())
}

func testGeneratorSendRequestSuccess(t *testing.T, method string) {
	t.Helper()

	const wantBody = "hello"
	headers := map[string]string{"X-Test-Header": "test-value"}
	srv := newTestServer(t, http.StatusCreated)

	g := newGenerator(t, srv, method, headers, wantBody)
	result := g.SendRequest(context.Background())

	assertNoError(t, result)
	assertSuccess(t, result, true)
	assertStatus(t, result, http.StatusCreated)
	assertDurationNonNegative(t, result)

	if got := srv.observed.method; got != method {
		t.Errorf("server saw method %q, want %q", got, method)
	}
	if got := srv.observed.header.Get("X-Test-Header"); got != "test-value" {
		t.Errorf("server saw X-Test-Header = %q, want %q", got, "test-value")
	}
	if got := srv.observed.body; got != wantBody {
		t.Errorf("server saw body = %q, want %q", got, wantBody)
	}
}

func TestGeneratorSendRequestSuccess_GET(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodGet)
}

func TestGeneratorSendRequestSuccess_POST(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodPost)
}

func TestGeneratorSendRequestSuccess_PUT(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodPut)
}

func TestGeneratorSendRequestSuccess_PATCH(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodPatch)
}

func TestGeneratorSendRequestSuccess_DELETE(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodDelete)
}

func TestGeneratorSendRequestSuccess_HEAD(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodHead)
}

func TestGeneratorSendRequestSuccess_OPTIONS(t *testing.T) {
	testGeneratorSendRequestSuccess(t, http.MethodOptions)
}

func TestGeneratorSendRequest_NonSuccessStatus(t *testing.T) {
	// Any 2xx is success and any other code is failure; cover both
	// ends plus a typical 4xx to lock the contract.
	cases := []struct {
		name       string
		statusCode int
		wantOK     bool
	}{
		{name: "200 OK", statusCode: http.StatusOK, wantOK: true},
		{name: "201 Created", statusCode: http.StatusCreated, wantOK: true},
		{name: "299 boundary", statusCode: 299, wantOK: true},
		{name: "300 redirect", statusCode: http.StatusMultipleChoices, wantOK: false},
		{name: "404 not found", statusCode: http.StatusNotFound, wantOK: false},
		{name: "500 server error", statusCode: http.StatusInternalServerError, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t, tc.statusCode)
			g := newGenerator(t, srv, http.MethodGet, nil, "")

			result := g.SendRequest(context.Background())

			assertNoError(t, result)
			assertSuccess(t, result, tc.wantOK)
			assertStatus(t, result, tc.statusCode)
		})
	}
}

func TestGeneratorSendRequest_InvalidURL(t *testing.T) {
	// No server: the spec is built directly so http.NewRequestWithContext
	// fails before any network call.
	g := NewGenerator(spec.TestSpec{
		TargetURL: "://not-a-url",
		Method:    http.MethodGet,
	}, http.DefaultClient)

	result := g.SendRequest(context.Background())

	if result.Err == nil {
		t.Fatal("SendRequest() error = nil, want an error for an invalid URL")
	}
	assertSuccess(t, result, false)
}

func TestGeneratorSendRequest_ContextCanceled(t *testing.T) {
	srv := newTestServer(t, http.StatusOK)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	g := newGenerator(t, srv, http.MethodGet, nil, "")
	result := g.SendRequest(ctx)

	if result.Err == nil {
		t.Fatal("SendRequest() error = nil, want context cancellation error")
	}
	if result.Success {
		t.Error("Success = true, want false for a canceled context")
	}
}

func TestGeneratorSendRequest_TransportError(t *testing.T) {
	wantErr := errors.New("connection failed")
	g := NewGenerator(spec.TestSpec{
		TargetURL: "http://example.test",
		Method:    http.MethodGet,
	}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})})

	result := g.SendRequest(context.Background())

	if !errors.Is(result.Err, wantErr) {
		t.Errorf("error = %v, want %v", result.Err, wantErr)
	}
	assertSuccess(t, result, false)
	if result.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0 when no response was received", result.StatusCode)
	}
	if result.Duration < 0 {
		t.Errorf("Duration = %v, want a non-negative duration", result.Duration)
	}
}

func TestGeneratorSendRequest_ResponseBodyReadError(t *testing.T) {
	wantErr := errors.New("body read failed")
	g := NewGenerator(spec.TestSpec{
		TargetURL: "http://example.test",
		Method:    http.MethodGet,
	}, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       errorReadCloser{err: wantErr},
			Request:    req,
		}, nil
	})})

	result := g.SendRequest(context.Background())

	if !errors.Is(result.Err, wantErr) {
		t.Errorf("error = %v, want %v", result.Err, wantErr)
	}
	assertSuccess(t, result, false)
	assertStatus(t, result, http.StatusOK)
	if result.Duration < 0 {
		t.Errorf("Duration = %v, want a non-negative duration", result.Duration)
	}
}

func TestGeneratorSendRequest_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
	}, server.Client())
	result := g.SendRequest(ctx)

	if result.Err == nil {
		t.Fatal("SendRequest() error = nil, want a context timeout error")
	}
	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context deadline exceeded", result.Err)
	}
	assertSuccess(t, result, false)
}

func TestNewGenerator_NilClientPanics(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("NewGenerator() did not panic for a nil client")
		} else if recovered != "Invalid client passed into generator" {
			t.Errorf("NewGenerator() panic = %v, want %q", recovered, "Invalid client passed into generator")
		}
	}()

	NewGenerator(spec.TestSpec{}, nil)
}

func TestGeneratorRunTest_InvalidSpec(t *testing.T) {
	tests := []struct {
		name     string
		testSpec spec.TestSpec
		wantErr  string
	}{
		{
			name:     "zero RPS",
			testSpec: spec.TestSpec{RPS: 0, Duration: spec.Duration{Duration: time.Second}},
			wantErr:  "RPS is invalid",
		},
		{
			name:     "negative RPS",
			testSpec: spec.TestSpec{RPS: -1, Duration: spec.Duration{Duration: time.Second}},
			wantErr:  "RPS is invalid",
		},
		{
			name:     "zero duration",
			testSpec: spec.TestSpec{RPS: 1, Duration: spec.Duration{}},
			wantErr:  "Duration is invalid",
		},
		{
			name:     "negative duration",
			testSpec: spec.TestSpec{RPS: 1, Duration: spec.Duration{Duration: -time.Second}},
			wantErr:  "Duration is invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGenerator(tc.testSpec, http.DefaultClient)

			summary, err := g.RunTest(context.Background())

			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("RunTest() error = %v, want %q", err, tc.wantErr)
			}
			if summary != (Summary{}) {
				t.Errorf("RunTest() summary = %+v, want zero summary", summary)
			}
		})
	}
}

func TestGeneratorRunTest_CompletesAndAggregatesRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
		RPS:       20,
		Duration:  spec.Duration{Duration: 260 * time.Millisecond},
	}, server.Client())

	summary, err := g.RunTest(context.Background())

	if err != nil {
		t.Fatalf("RunTest() error = %v, want nil", err)
	}
	if summary.ScheduledRequests == 0 {
		t.Fatal("ScheduledRequests = 0, want at least one scheduled request")
	}
	if summary.StartedRequests != summary.ScheduledRequests {
		t.Errorf("StartedRequests = %d, want it to match ScheduledRequests = %d",
			summary.StartedRequests, summary.ScheduledRequests)
	}
	if summary.CompletedRequests != summary.StartedRequests {
		t.Errorf("CompletedRequests = %d, want it to match StartedRequests = %d",
			summary.CompletedRequests, summary.StartedRequests)
	}
	if summary.LateCompletedRequests != 0 {
		t.Errorf("LateCompletedRequests = %d, want 0 for prompt responses", summary.LateCompletedRequests)
	}
	if summary.SuccessfulRequests != summary.CompletedRequests {
		t.Errorf("SuccessfulRequests = %d, want it to match CompletedRequests = %d",
			summary.SuccessfulRequests, summary.CompletedRequests)
	}
	if summary.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0", summary.FailedRequests)
	}
	if summary.MinRequestDuration < 0 ||
		summary.AverageRequestDuration <= 0 ||
		summary.MaxRequestDuration <= 0 ||
		summary.MinRequestDuration > summary.AverageRequestDuration ||
		summary.AverageRequestDuration > summary.MaxRequestDuration {
		t.Errorf("request duration summary is empty or inconsistent: min=%v average=%v max=%v",
			summary.MinRequestDuration, summary.AverageRequestDuration, summary.MaxRequestDuration)
	}
	if summary.P50 < 0 || summary.P95 < summary.P50 || summary.P99 < summary.P95 {
		t.Errorf("latency percentiles are inconsistent: P50=%v P95=%v P99=%v",
			summary.P50, summary.P95, summary.P99)
	}
}

func TestGeneratorRunTest_CountsHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
		RPS:       20,
		Duration:  spec.Duration{Duration: 220 * time.Millisecond},
	}, server.Client())

	summary, err := g.RunTest(context.Background())

	if err != nil {
		t.Fatalf("RunTest() error = %v, want nil", err)
	}
	if summary.CompletedRequests == 0 {
		t.Fatal("CompletedRequests = 0, want at least one response")
	}
	if summary.SuccessfulRequests != 0 {
		t.Errorf("SuccessfulRequests = %d, want 0 for HTTP %d responses",
			summary.SuccessfulRequests, http.StatusServiceUnavailable)
	}
	if summary.FailedRequests != summary.CompletedRequests {
		t.Errorf("FailedRequests = %d, want it to match CompletedRequests = %d",
			summary.FailedRequests, summary.CompletedRequests)
	}
}

func TestGeneratorRunTest_PreCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server received a request for a pre-canceled test")
	}))
	defer server.Close()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
		RPS:       100,
		Duration:  spec.Duration{Duration: time.Second},
	}, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summary, err := g.RunTest(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunTest() error = %v, want context.Canceled", err)
	}
	if summary != (Summary{}) {
		t.Errorf("RunTest() summary = %+v, want zero summary", summary)
	}
}

func TestGeneratorRunTest_CancellationCompletesInFlightRequests(t *testing.T) {
	requestStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestStarted <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
		RPS:       100,
		Duration:  spec.Duration{Duration: 2 * time.Second},
	}, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan struct {
		summary Summary
		err     error
	}, 1)

	go func() {
		summary, err := g.RunTest(ctx)
		resultCh <- struct {
			summary Summary
			err     error
		}{summary: summary, err: err}
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("timed out waiting for RunTest() to start a request")
	}
	cancel()

	var result struct {
		summary Summary
		err     error
	}
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("RunTest() did not return after its context was canceled")
	}

	if !errors.Is(result.err, context.Canceled) {
		t.Errorf("RunTest() error = %v, want context.Canceled", result.err)
	}
	if result.summary.StartedRequests == 0 {
		t.Fatal("StartedRequests = 0, want an in-flight request")
	}
	if result.summary.CompletedRequests != result.summary.StartedRequests {
		t.Errorf("CompletedRequests = %d, want all %d started requests to complete",
			result.summary.CompletedRequests, result.summary.StartedRequests)
	}
	if result.summary.FailedRequests != result.summary.CompletedRequests {
		t.Errorf("FailedRequests = %d, want all canceled requests (%d) to fail",
			result.summary.FailedRequests, result.summary.CompletedRequests)
	}
	if result.summary.SuccessfulRequests != 0 || result.summary.LateCompletedRequests != 0 {
		t.Errorf("canceled test has unexpected success/late counts: %+v", result.summary)
	}
}

func TestGeneratorRunTest_RecordsResponsesCompletedAfterDeadlineAsLate(t *testing.T) {
	releaseRequests := make(chan struct{})
	requestStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestStarted <- struct{}{}:
		default:
		}
		<-releaseRequests
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	g := NewGenerator(spec.TestSpec{
		TargetURL: server.URL,
		Method:    http.MethodGet,
		RPS:       50,
		Duration:  spec.Duration{Duration: 80 * time.Millisecond},
	}, server.Client())
	resultCh := make(chan struct {
		summary Summary
		err     error
	}, 1)

	go func() {
		summary, err := g.RunTest(context.Background())
		resultCh <- struct {
			summary Summary
			err     error
		}{summary: summary, err: err}
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		close(releaseRequests)
		t.Fatal("timed out waiting for RunTest() to start a request")
	}
	time.Sleep(120 * time.Millisecond)
	close(releaseRequests)

	var result struct {
		summary Summary
		err     error
	}
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("RunTest() did not return after delayed requests were released")
	}

	if result.err != nil {
		t.Fatalf("RunTest() error = %v, want nil", result.err)
	}
	if result.summary.ScheduledRequests == 0 {
		t.Fatal("ScheduledRequests = 0, want at least one scheduled request")
	}
	if result.summary.LateCompletedRequests == 0 {
		t.Fatalf("LateCompletedRequests = 0, want delayed responses to be counted as late: %+v",
			result.summary)
	}
	if result.summary.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0 because every response arrived after the deadline",
			result.summary.CompletedRequests)
	}
	if result.summary.SuccessfulRequests != result.summary.LateCompletedRequests {
		t.Errorf("SuccessfulRequests = %d, want all late 200 responses (%d) to count as successful",
			result.summary.SuccessfulRequests, result.summary.LateCompletedRequests)
	}
	if result.summary.MinRequestDuration != 0 ||
		result.summary.MaxRequestDuration != 0 ||
		result.summary.AverageRequestDuration != 0 ||
		result.summary.P50 != 0 || result.summary.P95 != 0 || result.summary.P99 != 0 {
		t.Errorf("late responses should not contribute to latency metrics: %+v", result.summary)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (r errorReadCloser) Close() error {
	return nil
}

// --- assertion helpers --------------------------------------------------

func assertNoError(t *testing.T, r RequestResult) {
	t.Helper()
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
}

func assertSuccess(t *testing.T, r RequestResult, want bool) {
	t.Helper()
	if r.Success != want {
		t.Errorf("Success = %v, want %v", r.Success, want)
	}
}

func assertStatus(t *testing.T, r RequestResult, want int) {
	t.Helper()
	if r.StatusCode != want {
		t.Errorf("StatusCode = %d, want %d", r.StatusCode, want)
	}
}

func assertDurationNonNegative(t *testing.T, r RequestResult) {
	t.Helper()
	if r.Duration < 0 {
		t.Errorf("Duration = %v, want a non-negative duration", r.Duration)
	}
}
