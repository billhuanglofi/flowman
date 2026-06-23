package runner_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func TestRunnerSendsConfiguredRequest(t *testing.T) {
	// Given
	received := make(chan observedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("expected request body read to succeed: %v", err)
		}
		received <- observedRequest{
			method:        r.Method,
			path:          r.URL.Path,
			rawQuery:      r.URL.RawQuery,
			contentType:   r.Header.Get("Content-Type"),
			correlationID: r.Header.Get("X-Correlation-ID"),
			authorization: r.Header.Get("Authorization"),
			body:          string(body),
		}
		w.Header().Set("X-Transaction-ID", "TX-123")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	t.Setenv("FLOWMAN_TEST_AUTH", "Bearer test-secret")
	runnerService := runner.New(runner.Options{Timeout: time.Second, Client: server.Client()})
	request := model.Request{
		Version:  "v1",
		Name:     "create payment",
		Method:   "POST",
		Endpoint: "payments",
		Path:     "/v1/payments",
		Query: []model.Parameter{
			{Name: "dry_run", Value: "false"},
			{Name: "source", Value: "flowman-fixture"},
		},
		Headers: []model.Header{
			{Name: "Content-Type", Value: "application/json"},
			{Name: "X-Correlation-ID", Value: "flowman-example-correlation"},
			{Name: "Authorization", Env: "FLOWMAN_TEST_AUTH"},
		},
		Body: model.RequestBody{Mode: "json", Raw: `{"amount":"12.34"}`},
	}
	environment := model.Environment{
		Name:    "test",
		BaseURL: server.URL,
		Endpoints: []model.EndpointAlias{{
			Name: "payments",
			Path: "/payments",
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	result, err := runnerService.Run(ctx, runner.Execution{Request: request, Environment: environment})

	// Then
	if err != nil {
		t.Fatalf("expected request to run: %v", err)
	}
	if result.StatusCode != http.StatusAccepted || string(result.Body) != `{"ok":true}` {
		t.Fatalf("expected captured accepted response, got %#v", result)
	}
	assertObservedRequest(t, <-received)
}

func TestRunnerCapturesResponseMetadataAndRedactedDisplayHeaders(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=test-secret")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()
	runnerService := runner.New(runner.Options{Timeout: time.Second, Client: server.Client()})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	result, err := runnerService.Run(ctx, runner.Execution{
		Request:     model.Request{Version: "v1", Name: "status", Method: "GET", URL: server.URL + "/status"},
		Environment: model.Environment{Name: "test"},
	})

	// Then
	if err != nil {
		t.Fatalf("expected request to run: %v", err)
	}
	if result.StatusCode != http.StatusCreated || result.Headers.Get("Content-Type") != "application/json" || string(result.Body) != `{"created":true}` {
		t.Fatalf("expected response metadata captured, got %#v", result)
	}
	if result.Duration <= 0 {
		t.Fatalf("expected positive duration, got %s", result.Duration)
	}
	if result.DisplayHeaders.Get("Set-Cookie") != secrets.RedactedValue {
		t.Fatalf("expected redacted display Set-Cookie, got %#v", result.DisplayHeaders)
	}
	if result.Headers.Get("Set-Cookie") != "session=test-secret" {
		t.Fatalf("expected raw response headers retained separately, got %#v", result.Headers)
	}
}

func TestRunnerResolvesEnvBaseURLWithEndpointAndPath(t *testing.T) {
	// Given
	pathSeen := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSeen <- r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	runnerService := runner.New(runner.Options{Timeout: time.Second, Client: server.Client()})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	_, err := runnerService.Run(ctx, runner.Execution{
		Request: model.Request{Version: "v1", Name: "create payment", Method: "GET", Endpoint: "payments", Path: "/v1/payments"},
		Environment: model.Environment{
			Name:      "test",
			BaseURL:   server.URL + "/api",
			Endpoints: []model.EndpointAlias{{Name: "payments", Path: "/payments"}},
		},
	})

	// Then
	if err != nil {
		t.Fatalf("expected env endpoint/path URL to resolve: %v", err)
	}
	if got := <-pathSeen; got != "/api/payments/v1/payments" {
		t.Fatalf("expected joined endpoint path, got %q", got)
	}
}

func TestRunnerReturnsCannotResolveRequestURLErrorWithoutHTTPCall(t *testing.T) {
	// Given
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	runnerService := runner.New(runner.Options{Timeout: time.Second, Client: server.Client()})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	_, err := runnerService.Run(ctx, runner.Execution{
		Request:     model.Request{Version: "v1", Name: "missing url", Method: "GET"},
		Environment: model.Environment{Name: "test", BaseURL: server.URL},
	})

	// Then
	if !errors.Is(err, runner.ErrCannotResolveRequestURL) {
		t.Fatalf("expected cannot resolve request URL error, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "cannot resolve request URL") || !strings.Contains(err.Error(), "missing url") {
		t.Fatalf("expected actionable URL resolution error, got %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("expected no HTTP call, got %d", calls.Load())
	}
}

func TestRunnerDoesNotMutateCanonicalRequest(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("FLOWMAN_TEST_AUTH", "Bearer test-secret")
	runnerService := runner.New(runner.Options{Timeout: time.Second, Client: server.Client()})
	request := model.Request{
		Version: "v1",
		Name:    "immutable",
		Method:  "POST",
		URL:     server.URL + "/immutable",
		Query:   []model.Parameter{{Name: "a", Value: "b"}},
		Headers: []model.Header{{Name: "Authorization", Env: "FLOWMAN_TEST_AUTH"}},
		Body:    model.RequestBody{Mode: "form", Form: []model.Parameter{{Name: "amount", Value: "12.34"}}},
	}
	before := request
	before.Query = append([]model.Parameter(nil), request.Query...)
	before.Headers = append([]model.Header(nil), request.Headers...)
	before.Body.Form = append([]model.Parameter(nil), request.Body.Form...)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	_, err := runnerService.Run(ctx, runner.Execution{Request: request, Environment: model.Environment{Name: "test"}})

	// Then
	if err != nil {
		t.Fatalf("expected request to run: %v", err)
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatalf("expected canonical request not to mutate, before=%#v after=%#v", before, request)
	}
}

type observedRequest struct {
	method        string
	path          string
	rawQuery      string
	contentType   string
	correlationID string
	authorization string
	body          string
}

func assertObservedRequest(t *testing.T, got observedRequest) {
	t.Helper()
	if got.method != "POST" || got.path != "/payments/v1/payments" {
		t.Fatalf("expected POST /payments/v1/payments, got %#v", got)
	}
	if got.rawQuery != "dry_run=false&source=flowman-fixture" {
		t.Fatalf("expected encoded query params, got %q", got.rawQuery)
	}
	if got.contentType != "application/json" || got.correlationID != "flowman-example-correlation" || got.authorization != "Bearer test-secret" {
		t.Fatalf("expected configured headers, got %#v", got)
	}
	if got.body != `{"amount":"12.34"}` {
		t.Fatalf("expected configured body, got %q", got.body)
	}
}
