package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	"github.com/billhuanglofi/flowman/internal/trace"
)

func TestPreviewRendersFlowmanShell(t *testing.T) {
	// Given
	preview := Preview{
		Workspace: model.Workspace{
			Project:      model.Project{Name: "flowman-example"},
			Environments: []model.Environment{{Name: "uat", BaseURL: "https://uat.api.example.test"}},
			Requests:     []model.Request{{Name: "create payment", Method: "POST", Endpoint: "payments", Path: "/v1/payments"}},
		},
		SelectedEnv: "uat",
		Response:    ResponsePreview{State: "not run", Status: "response pending"},
		RunState:    "ready to run selected request",
		TraceState:  "trace waiting for transaction id",
		Journey: trace.MapProcessStateJourney("TX123", []trace.TraceRow{
			{TransactionID: "TX123", Timestamp: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), Step: 0, Service: "gateway", State: "RECEIVED", Outcome: "OK", Message: "accepted"},
			{TransactionID: "TX123", Timestamp: time.Date(2026, 6, 21, 10, 0, 1, 0, time.UTC), Step: 1, Service: "payments", State: "PROCESSING", Outcome: "PENDING", Message: "waiting"},
		}, trace.DefaultTerminalStateClassification()),
	}

	// When
	rendered := RenderPreview(preview, RenderSize{Width: 100, Height: 32})

	// Then
	for _, expected := range []string{
		"Flowman",
		"Environment: uat",
		"Base URL: https://uat.api.example.test",
		"Requests",
		"POST create payment",
		"Selected Request",
		"Response",
		"response pending",
		"Run: ready to run selected request",
		"Trace Journey",
		"gateway",
		"payments",
		"TX123",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected render to contain %q, got:\n%s", expected, rendered)
		}
	}
	if strings.Contains(rendered, "uathttps://") {
		t.Fatalf("expected environment and base URL to be separated, got:\n%s", rendered)
	}
}

func TestPreviewRendersActionableError_whenConfigOrEnvMissing(t *testing.T) {
	// Given
	preview := Preview{SelectedEnv: "missing", LoadError: errPreviewLoad{Message: "load config missing.yaml: open missing.yaml: no such file or directory"}}

	// When
	rendered := RenderPreview(preview, RenderSize{Width: 80, Height: 24})

	// Then
	for _, expected := range []string{"Flowman", "Unable to render preview", "check --config", "check --env", "missing.yaml"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected error render to contain %q, got:\n%s", expected, rendered)
		}
	}
}

func TestPreviewFromWorkspace_usesRunnerExtractionAndTraceModels(t *testing.T) {
	// Given
	workspace := model.Workspace{
		Project:      model.Project{Name: "flowman-example"},
		Environments: []model.Environment{{Name: "uat", BaseURL: "https://uat.api.example.test"}},
		Requests: []model.Request{{
			Name: "create payment", Method: "POST", Path: "/v1/payments",
			Trace: model.TraceConfig{TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"}},
		}},
	}
	response := runner.Response{StatusCode: 202, Headers: map[string][]string{"X-Transaction-ID": {"TX999"}}, Body: []byte(`{"ok":true}`), Duration: 125 * time.Millisecond}
	journey := trace.MapProcessStateJourney("TX999", []trace.TraceRow{{TransactionID: "TX999", Step: 0, Service: "gateway", State: "COMPLETE", Outcome: "OK", Message: "done"}}, trace.DefaultTerminalStateClassification())

	// When
	preview := PreviewFromWorkspace(workspace, PreviewOptions{EnvName: "uat", Response: response, Journey: journey})

	// Then
	rendered := RenderPreview(preview, RenderSize{Width: 100, Height: 32})
	for _, expected := range []string{"Status: 202", "Transaction: TX999", "Duration: 125ms", "gateway"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected render to contain %q, got:\n%s", expected, rendered)
		}
	}
}

type errPreviewLoad struct {
	Message string
}

func (err errPreviewLoad) Error() string {
	return err.Message
}
