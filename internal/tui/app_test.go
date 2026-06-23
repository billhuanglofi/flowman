package tui

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestModel_runsRequestExtractsTransactionAndRendersTrace_whenRequestSelected(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{
		response: runner.Response{
			StatusCode:     http.StatusAccepted,
			Headers:        http.Header{"X-Transaction-ID": []string{"TX-TUI-123"}},
			DisplayHeaders: http.Header{"X-Transaction-ID": []string{"TX-TUI-123"}},
			Body:           []byte(`{"accepted":true}`),
			Duration:       125 * time.Millisecond,
		},
		journey: flowtrace.MapProcessStateJourney("TX-TUI-123", []flowtrace.TraceRow{
			{TransactionID: "TX-TUI-123", Step: 0, Service: "gateway", State: "DONE", Outcome: "OK", Message: "accepted"},
			{TransactionID: "TX-TUI-123", Step: 1, Service: "payments", State: "RUNNING", Outcome: "PENDING", Message: "waiting for processor"},
		}, flowtrace.DefaultTerminalStateClassification()),
	}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services})

	// When
	updated, cmd := tuiModel.Update(press("r"))
	if cmd == nil {
		t.Fatalf("expected run key to return command")
	}
	msg := cmd()
	updated, cmd = updated.Update(msg)
	if cmd != nil {
		msg = cmd()
		updated, _ = updated.Update(msg)
	}
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Status: 202", "Transaction: TX-TUI-123", "gateway", "payments", "waiting for processor"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered TUI to contain %q, got:\n%s", expected, rendered)
		}
	}
	if services.runCalls != 1 || services.traceCalls != 1 {
		t.Fatalf("expected one runner call and one trace call, got runner=%d trace=%d", services.runCalls, services.traceCalls)
	}
	if services.traceTransactionID != "TX-TUI-123" {
		t.Fatalf("expected trace to use extracted transaction id, got %q", services.traceTransactionID)
	}
}

func TestModel_rendersEnvironmentFlowAndRequestLists(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: &fakeWorkflowServices{}})

	// When
	rendered := tuiModel.View().Content

	// Then
	for _, expected := range []string{"Environments", "> uat", "ppd", "Flows", "create payment flow", "create refund flow", "Requests", "create payment"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered TUI to list %q, got:\n%s", expected, rendered)
		}
	}
}

func TestModel_acceptsTypedTransactionIDThenRendersTrace_whenRunWasNotUsed(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{
		journey: flowtrace.MapProcessStateJourney("TX-MANUAL-7", []flowtrace.TraceRow{
			{TransactionID: "TX-MANUAL-7", Step: 0, Service: "manual", State: "COMPLETE", Outcome: "OK", Message: "accepted tx"},
		}, flowtrace.DefaultTerminalStateClassification()),
	}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services})

	// When
	updated, _ := tuiModel.Update(press("t"))
	for _, char := range "TX-MANUAL-7" {
		updated, _ = updated.Update(press(string(char)))
	}
	updated, cmd := updated.Update(pressSpecial(uv.KeyEnter))
	msg := cmd()
	updated, _ = updated.Update(msg)
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Transaction: TX-MANUAL-7", "manual", "accepted tx"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered TUI to contain %q, got:\n%s", expected, rendered)
		}
	}
	if services.traceTransactionID != "TX-MANUAL-7" {
		t.Fatalf("expected manual transaction id to drive trace, got %q", services.traceTransactionID)
	}
}

func TestModel_rendersActionableErrorAndStaysSelectable_whenTraceFails(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{
		response: runner.Response{StatusCode: http.StatusOK, Headers: http.Header{"X-Transaction-ID": []string{"TX-FAIL"}}, Duration: time.Millisecond},
		traceErr: errors.New("oracle support not enabled; rebuild with tags"),
	}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services})

	// When
	updated, cmd := tuiModel.Update(press("r"))
	updated, cmd = updated.Update(cmd())
	updated, _ = updated.Update(cmd())
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Trace error", "oracle support not enabled", "check --env Oracle settings or retry after import URL resolution", "r run"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected error state to contain %q, got:\n%s", expected, rendered)
		}
	}
	if finalModel.status != statusReady {
		t.Fatalf("expected model to return to selectable ready state, got %q", finalModel.status)
	}
}

func TestModel_opensSafestoreImportFlowAndRequiresURLResolution(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services})

	// When
	updated, _ := tuiModel.Update(press("i"))
	for _, char := range "TX-IMPORT" {
		updated, _ = updated.Update(press(string(char)))
	}
	updated, cmd := updated.Update(pressSpecial(uv.KeyEnter))
	msg := cmd()
	updated, _ = updated.Update(msg)
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Safestore Import", "requires --url, endpoint alias, or env base_url + path", "TX-IMPORT"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected Safestore import state to contain %q, got:\n%s", expected, rendered)
		}
	}
	if services.importCalls != 1 {
		t.Fatalf("expected import service to be called once, got %d", services.importCalls)
	}
}

func TestModel_importsSafestoreReplay_whenFullURLProvided(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{
		importResult: safestore.ImportResult{
			Request:    model.Request{Name: "replay-TX-IMPORT", Method: "POST", URL: "https://replay.example.test/payments"},
			OutputPath: filepath.Join("requests", "replay-TX-IMPORT.request.yaml"),
		},
	}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services, ImportURL: "https://replay.example.test/payments"})

	// When
	updated, _ := tuiModel.Update(press("i"))
	for _, char := range "TX-IMPORT" {
		updated, _ = updated.Update(press(string(char)))
	}
	updated, cmd := updated.Update(pressSpecial(uv.KeyEnter))
	msg := cmd()
	updated, _ = updated.Update(msg)
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Safestore Import", "Imported: replay-TX-IMPORT", "requests/replay-TX-IMPORT.request.yaml"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected Safestore import success to contain %q, got:\n%s", expected, rendered)
		}
	}
	if services.importURL != "https://replay.example.test/payments" {
		t.Fatalf("expected import URL override to reach service, got %q", services.importURL)
	}
}

func TestModel_importsSafestoreReplay_whenEndpointAndPathProvided(t *testing.T) {
	// Given
	workspace := tuiTestWorkspace()
	services := &fakeWorkflowServices{
		importResult: safestore.ImportResult{
			Request:    model.Request{Name: "replay-TX-ENDPOINT", Method: "POST", Endpoint: "payments", Path: "/v1/replay"},
			OutputPath: filepath.Join("requests", "replay-TX-ENDPOINT.request.yaml"),
		},
	}
	tuiModel := NewWorkflowModel(workspace, WorkflowOptions{EnvName: "uat", Services: services, ImportEndpoint: "payments", ImportPath: "/v1/replay"})

	// When
	updated, _ := tuiModel.Update(press("i"))
	for _, char := range "TX-ENDPOINT" {
		updated, _ = updated.Update(press(string(char)))
	}
	updated, cmd := updated.Update(pressSpecial(uv.KeyEnter))
	updated, _ = updated.Update(cmd())
	finalModel := updated.(Model)

	// Then
	rendered := finalModel.View().Content
	for _, expected := range []string{"Endpoint: payments", "Path: /v1/replay", "Imported: replay-TX-ENDPOINT"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected Safestore endpoint/path state to contain %q, got:\n%s", expected, rendered)
		}
	}
	if services.importEndpoint != "payments" || services.importPath != "/v1/replay" {
		t.Fatalf("expected endpoint/path to reach import service, got endpoint=%q path=%q", services.importEndpoint, services.importPath)
	}
}

func TestDefaultServices_doesNotCloseTraceStoreBeforeUse(t *testing.T) {
	// Given
	store := &closeTrackingTraceStore{journey: flowtrace.Journey{TransactionID: "TX-LIVE"}}
	services := NewServices(fakeRequestRunner{}, store, func(context.Context, safestore.ImportOptions) (safestore.ImportResult, error) {
		return safestore.ImportResult{}, nil
	})

	// When
	journey, err := services.TraceJourney(context.Background(), flowtrace.ProcessStateRequest{TransactionID: "TX-LIVE"})

	// Then
	if err != nil {
		t.Fatalf("expected trace store usable before close, got %v", err)
	}
	if journey.TransactionID != "TX-LIVE" {
		t.Fatalf("expected live journey, got %#v", journey)
	}
	if store.closed {
		t.Fatalf("expected trace store not to close before use")
	}
}

func TestRenderJourney_usesWideSafeBoundedCells(t *testing.T) {
	// Given
	journey := flowtrace.MapProcessStateJourney("TX-CJK", []flowtrace.TraceRow{{
		TransactionID: "TX-CJK",
		Step:          0,
		Service:       "결제서비스이름이매우김",
		State:         "처리중상태값",
		Outcome:       "대기중결과값",
		Message:       "메시지본문이길어도표경계가흐트러지지않아야합니다",
	}}, flowtrace.DefaultTerminalStateClassification())

	// When
	rendered := RenderPreview(Preview{Workspace: tuiTestWorkspace(), SelectedEnv: "uat", Response: placeholderResponse(), Journey: journey}, RenderSize{Width: 80, Height: 24})

	// Then
	for _, line := range strings.Split(stripANSI(rendered), "\n") {
		if strings.Contains(line, "│") && len([]rune(line)) > 90 {
			t.Fatalf("expected bounded CJK journey row, got width-risk line %q", line)
		}
	}
}
