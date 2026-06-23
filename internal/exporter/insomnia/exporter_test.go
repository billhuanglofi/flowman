package insomnia_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/exporter/insomnia"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestExportWorkspace_writes_insomnia_document_and_reports_unsupported_flow_shapes(t *testing.T) {
	// Given
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	workspaceRoot := t.TempDir()
	writeFixtureWorkspace(t, workspaceRoot)
	outputPath := filepath.Join(t.TempDir(), "payments.insomnia.json")
	reportPath := filepath.Join(t.TempDir(), "report.json")

	// When
	result, err := insomnia.ExportWorkspace(ctx, insomnia.ExportOptions{WorkspacePath: workspaceRoot, OutputPath: outputPath, ReportPath: reportPath})

	// Then
	if err != nil {
		t.Fatalf("expected insomnia export to succeed: %v", err)
	}
	if result.ExportedCount != 2 {
		t.Fatalf("expected two exported requests, got %#v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning for multi-step flow subset limitation, got %#v", result)
	}
	content, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("expected exported document to be readable: %v", readErr)
	}
	text := string(content)
	for _, want := range []string{"\"_type\": \"export\"", "Create Payment", "Get Status", "{{ FLOWMAN_PAYMENT_AUTH }}"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected exported document to contain %q, got %s", want, text)
		}
	}
	if strings.Contains(text, "Bearer secret") {
		t.Fatalf("expected exported document to avoid raw secrets, got %s", text)
	}
	report, reportErr := os.ReadFile(reportPath)
	if reportErr != nil {
		t.Fatalf("expected warning report to be readable: %v", reportErr)
	}
	if !strings.Contains(string(report), "insomnia-flow-multistep-unsupported") {
		t.Fatalf("expected report warning in %s", string(report))
	}
}

func TestExportWorkspace_strict_warnings_fail_before_output_write(t *testing.T) {
	// Given
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	workspaceRoot := t.TempDir()
	writeFixtureWorkspace(t, workspaceRoot)
	outputPath := filepath.Join(t.TempDir(), "payments.insomnia.json")
	reportPath := filepath.Join(t.TempDir(), "report.json")

	// When
	result, err := insomnia.ExportWorkspace(ctx, insomnia.ExportOptions{WorkspacePath: workspaceRoot, OutputPath: outputPath, ReportPath: reportPath, Strict: true})

	// Then
	if !errors.Is(err, insomnia.ErrStrictWarnings) {
		t.Fatalf("expected strict warning failure, got %v", err)
	}
	if result.ExportedCount != 2 || len(result.Warnings) == 0 {
		t.Fatalf("expected result warnings with exported count, got %#v", result)
	}
	if _, statErr := os.Stat(outputPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected strict mode to avoid output write, stat err=%v", statErr)
	}
	report, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("expected strict warning report to be written: %v", readErr)
	}
	if !strings.Contains(string(report), "insomnia-flow-multistep-unsupported") {
		t.Fatalf("expected warning report content, got %s", string(report))
	}
}

func writeFixtureWorkspace(t *testing.T, root string) {
	t.Helper()
	project := model.Project{Version: "v1", Name: "fixture", DefaultEnv: "uat", Environments: []string{"envs/uat.yaml"}, Requests: []string{"requests/payments/create-payment.request.yaml", "requests/payments/get-status.request.yaml"}, Flows: []string{"flows/payments/payments.flow.yaml"}}
	environment := model.Environment{
		Version: "v1",
		Name:    "uat",
		BaseURL: "https://uat.api.example.test",
		Oracle:  model.OracleDatabase{DSNEnv: "FLOWMAN_UAT_ORACLE_DSN", UserEnv: "FLOWMAN_UAT_ORACLE_USER", PasswordEnv: "FLOWMAN_UAT_ORACLE_PASSWORD"},
		Trace:   model.TraceConfig{TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"}, PollInterval: model.NewDuration(2 * time.Second), Timeout: model.NewDuration(30 * time.Second)},
	}
	createRequest := model.Request{Version: "v1", Name: "Create Payment", Method: "POST", Path: "/payments", Query: []model.Parameter{{Name: "dry_run", Value: "false"}}, Headers: []model.Header{{Name: "Authorization", Env: "FLOWMAN_PAYMENT_AUTH"}, {Name: "Content-Type", Value: "application/json"}}, Body: model.RequestBody{Mode: "json", Raw: "{\"amount\":\"12.34\"}"}}
	statusRequest := model.Request{Version: "v1", Name: "Get Status", Method: "GET", URL: "https://api.example.test/status", Headers: []model.Header{{Name: "X-API-Key", Env: "FLOWMAN_STATUS_KEY"}}}
	flow := model.Flow{Version: "v1", Name: "Payments", Steps: []model.FlowStep{{Name: "Create Payment", Request: "requests/payments/create-payment.request.yaml", Trace: true}, {Name: "Get Status", Request: "requests/payments/get-status.request.yaml"}}}
	if err := storage.WriteCanonical(filepath.Join(root, "flowman.yaml"), project); err != nil {
		t.Fatalf("expected project fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "envs", "uat.yaml"), environment); err != nil {
		t.Fatalf("expected environment fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "requests", "payments", "create-payment.request.yaml"), createRequest); err != nil {
		t.Fatalf("expected request fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "requests", "payments", "get-status.request.yaml"), statusRequest); err != nil {
		t.Fatalf("expected request fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "flows", "payments", "payments.flow.yaml"), flow); err != nil {
		t.Fatalf("expected flow fixture write to succeed: %v", err)
	}
}
