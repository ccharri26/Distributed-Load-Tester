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
