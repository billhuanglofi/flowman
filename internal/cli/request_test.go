package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/runner"
	tracemodel "github.com/billhuanglofi/flowman/internal/trace"
)

func Test_RequestRunCommand_prints_redacted_json_report(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	requestPath := filepath.Join("..", "..", "testdata", "flowman", "requests", "payment", "create.request.yaml")
	factory := &capturingRequestExecutorFactory{response: runner.Response{
		StatusCode: http.StatusAccepted,
		Headers: http.Header{
			"X-Transaction-ID": []string{"TX-123"},
			"Set-Cookie":       []string{"session=raw-secret"},
		},
		DisplayHeaders: http.Header{
			"X-Transaction-ID": []string{"TX-123"},
			"Set-Cookie":       []string{"[REDACTED]"},
		},
		Body:     []byte(`{"transaction_id":"TX-123","token":"secret-body"}`),
		Duration: 45 * time.Millisecond,
	}}
	restore := stubRequestExecutorFactory(factory)
	defer restore()

	// When
	err := Execute(ctx, []string{"request", "run", requestPath, "--env", "uat"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected request run to succeed: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if factory.timeout != 30*time.Second {
		t.Fatalf("expected timeout from request trace config, got %s", factory.timeout)
	}
	if factory.execution.Request.Name != "create payment" || factory.execution.Environment.Name != "uat" {
		t.Fatalf("expected loaded request and environment, got %#v", factory.execution)
	}
	var report requestRunReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected JSON report, got %q: %v", stdout.String(), err)
	}
	if report.TransactionID != "TX-123" || report.StatusCode != http.StatusAccepted {
		t.Fatalf("expected extracted tx and status in report, got %#v", report)
	}
	if report.BodyBytes != len(factory.response.Body) {
		t.Fatalf("expected body bytes in report, got %#v", report)
	}
	if got := report.DisplayHeaders["Set-Cookie"]; len(got) != 1 || got[0] != "[REDACTED]" {
		t.Fatalf("expected redacted display headers, got %#v", report.DisplayHeaders)
	}
	combined := stdout.String() + stderr.String()
	for _, secret := range []string{"raw-secret", "secret-body"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("expected secret %q to stay out of output, got %q", secret, combined)
		}
	}
	if !strings.HasSuffix(stdout.String(), "\n") {
		t.Fatalf("expected JSON output to end with newline, got %q", stdout.String())
	}
}

func Test_RequestRunCommand_returns_validation_error_when_environment_missing(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	requestPath := filepath.Join("..", "..", "testdata", "flowman", "requests", "payment", "create.request.yaml")

	// When
	err := Execute(ctx, []string{"request", "run", requestPath, "--env", "missing"}, stdout, stderr)

	// Then
	if err == nil {
		t.Fatal("expected request run to fail for missing env")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected env name in error, got %v", err)
	}
	if strings.Contains(strings.ToLower(stdout.String()+stderr.String()), "panic") {
		t.Fatalf("expected validation failure without panic, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no success output on validation failure, got %q", stdout.String())
	}
}

type capturingRequestExecutorFactory struct {
	timeout   time.Duration
	execution runner.Execution
	response  runner.Response
	err       error
}

func (factory *capturingRequestExecutorFactory) new(timeout time.Duration) requestExecutor {
	factory.timeout = timeout
	return capturingRequestExecutor{factory: factory}
}

type capturingRequestExecutor struct {
	factory *capturingRequestExecutorFactory
}

func (executor capturingRequestExecutor) Run(_ context.Context, execution runner.Execution) (runner.Response, error) {
	executor.factory.execution = execution
	return executor.factory.response, executor.factory.err
}

func stubRequestExecutorFactory(factory *capturingRequestExecutorFactory) func() {
	previous := newRequestExecutor
	newRequestExecutor = factory.new
	return func() {
		newRequestExecutor = previous
	}
}

func Test_RequestRunCommand_returns_error_when_transaction_id_cannot_be_extracted(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	requestPath := filepath.Join("..", "..", "testdata", "flowman", "requests", "payment", "create.request.yaml")
	factory := &capturingRequestExecutorFactory{response: runner.Response{
		StatusCode:     http.StatusAccepted,
		Headers:        http.Header{},
		DisplayHeaders: http.Header{},
		Body:           []byte(`{"message":"no transaction id here"}`),
		Duration:       time.Millisecond,
	}}
	restore := stubRequestExecutorFactory(factory)
	defer restore()

	// When
	err := Execute(ctx, []string{"request", "run", requestPath, "--env", "uat"}, stdout, stderr)

	// Then
	if !errors.Is(err, tracemodel.ErrTransactionIDNotFound) {
		t.Fatalf("expected extraction failure, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no report on extraction failure, got %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "secret") {
		t.Fatalf("expected stderr to stay redacted, got %q", stderr.String())
	}
}
