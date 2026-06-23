package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/oracle"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

func Test_TraceCommand_prints_text_report_from_store(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opened := traceOpenCapture{}
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", "secret-dsn")
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", "secret-user")
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", "secret-password")
	restore := stubTraceStoreOpener(func(_ context.Context, cfg oracle.EnvConfig) (flowtrace.ProcessStateStore, func() error, error) {
		opened.config = cfg
		return fakeTraceStore{journey: flowtrace.Journey{
				TransactionID: "TX-123",
				Rows: []flowtrace.JourneyRow{{
					Timestamp:    time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC),
					Step:         1,
					Service:      "payments",
					State:        "RUNNING",
					Outcome:      "PENDING",
					Message:      "waiting for downstream",
					DisplayLabel: flowtrace.DisplayLabelStuck,
				}},
				Warnings: []flowtrace.JourneyWarning{{Code: "process_state_empty", Message: "noisy warning"}},
			}, closeCalled: &opened.closed}, func() error {
				opened.closed = true
				return nil
			}, nil
	})
	defer restore()

	// When
	err := Execute(ctx, []string{"trace", "--tx", "TX-123", "--env", "uat", "--config", "../../testdata/flowman/flowman.yaml"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected trace command to succeed: %v", err)
	}
	if !opened.closed {
		t.Fatal("expected trace store closer to run")
	}
	wantConfig := oracle.EnvConfig{
		DSNEnv:      "FLOWMAN_UAT_ORACLE_DSN",
		UserEnv:     "FLOWMAN_UAT_ORACLE_USER",
		PasswordEnv: "FLOWMAN_UAT_ORACLE_PASSWORD",
	}
	if !reflect.DeepEqual(opened.config, wantConfig) {
		t.Fatalf("expected oracle env-name config, got %#v", opened.config)
	}
	output := stdout.String()
	for _, want := range []string{"Transaction: TX-123", "payments", "RUNNING", "stuck", `"waiting for downstream"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected trace output to contain %q, got:\n%s", want, output)
		}
	}
	combined := output + stderr.String()
	for _, secret := range []string{"secret-dsn", "secret-user", "secret-password"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("expected secret %q to stay out of trace output, got %q", secret, combined)
		}
	}
}

type traceOpenCapture struct {
	config oracle.EnvConfig
	closed bool
}

type fakeTraceStore struct {
	journey     flowtrace.Journey
	err         error
	closeCalled *bool
}

func (store fakeTraceStore) Journey(_ context.Context, _ flowtrace.ProcessStateRequest) (flowtrace.Journey, error) {
	return store.journey, store.err
}

func stubTraceStoreOpener(opener func(context.Context, oracle.EnvConfig) (flowtrace.ProcessStateStore, func() error, error)) func() {
	previous := openTraceStore
	openTraceStore = opener
	return func() {
		openTraceStore = previous
	}
}
