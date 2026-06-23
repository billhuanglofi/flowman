package importer_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	insomniaexport "github.com/billhuanglofi/flowman/internal/exporter/insomnia"
	postmanexport "github.com/billhuanglofi/flowman/internal/exporter/postman"
	insomniaimport "github.com/billhuanglofi/flowman/internal/importer/insomnia"
	postmanimport "github.com/billhuanglofi/flowman/internal/importer/postman"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestPostmanInsomniaSupportedSubsetRoundTrip(t *testing.T) {
	// Given
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	workspaceRoot := writeRoundTripWorkspace(t)
	postmanPath := filepath.Join(t.TempDir(), "flowman.postman_collection.json")
	insomniaPath := filepath.Join(t.TempDir(), "flowman.insomnia.json")
	postmanImportRoot := filepath.Join(t.TempDir(), "postman-import")
	insomniaImportRoot := filepath.Join(t.TempDir(), "insomnia-import")

	// When
	postmanExport, postmanExportErr := postmanexport.ExportWorkspace(ctx, postmanexport.ExportOptions{WorkspacePath: workspaceRoot, OutputPath: postmanPath})
	insomniaExport, insomniaExportErr := insomniaexport.ExportWorkspace(ctx, insomniaexport.ExportOptions{WorkspacePath: workspaceRoot, OutputPath: insomniaPath})
	postmanImport, postmanImportErr := postmanimport.ImportCollection(ctx, postmanimport.ImportOptions{CollectionPath: postmanPath, OutputDir: postmanImportRoot})
	insomniaImport, insomniaImportErr := insomniaimport.ImportWorkspace(ctx, insomniaimport.ImportOptions{DocumentPath: insomniaPath, OutputDir: insomniaImportRoot})

	// Then
	if postmanExportErr != nil {
		t.Fatalf("expected postman export to succeed: %v", postmanExportErr)
	}
	if insomniaExportErr != nil {
		t.Fatalf("expected insomnia export to succeed: %v", insomniaExportErr)
	}
	if postmanImportErr != nil {
		t.Fatalf("expected postman re-import to succeed: %v", postmanImportErr)
	}
	if insomniaImportErr != nil {
		t.Fatalf("expected insomnia re-import to succeed: %v", insomniaImportErr)
	}
	if len(postmanExport.Warnings) != 0 {
		t.Fatalf("expected supported postman export without warnings, got %#v", postmanExport.Warnings)
	}
	if len(insomniaExport.Warnings) != 0 {
		t.Fatalf("expected supported insomnia export without warnings, got %#v", insomniaExport.Warnings)
	}
	assertRoundTripRequest(t, filepath.Join(postmanImportRoot, "requests", "payments", "create-payment.request.yaml"))
	assertRoundTripRequest(t, filepath.Join(insomniaImportRoot, "requests", "payments", "create-payment.request.yaml"))
	assertRoundTripFlow(t, filepath.Join(postmanImportRoot, "flows", "payments", "create-payment.flow.yaml"))
	assertRoundTripFlow(t, filepath.Join(insomniaImportRoot, "flows", "payments", "create-payment.flow.yaml"))
	if postmanImport.ImportedCount != 1 {
		t.Fatalf("expected one postman request imported, got %#v", postmanImport)
	}
	if insomniaImport.ImportedCount != 1 {
		t.Fatalf("expected one insomnia request imported, got %#v", insomniaImport)
	}
}

func assertRoundTripRequest(t *testing.T, requestPath string) {
	t.Helper()
	request, err := storage.LoadRequest(requestPath)
	if err != nil {
		t.Fatalf("expected canonical request at %s: %v", requestPath, err)
	}
	if request.Name != "Create Payment" || request.Method != "POST" {
		t.Fatalf("expected request metadata preserved, got %#v", request)
	}
	if request.Path != "/payments" {
		t.Fatalf("expected canonical path preserved, got %#v", request)
	}
	if len(request.Query) != 2 || request.Query[0].Name != "dry_run" || request.Query[1].Name != "source" {
		t.Fatalf("expected ordered query params preserved, got %#v", request.Query)
	}
	if request.Body.Mode != "json" || request.Body.Raw == "" {
		t.Fatalf("expected JSON body preserved, got %#v", request.Body)
	}
	if len(request.Headers) != 2 {
		t.Fatalf("expected two headers preserved in %s, got %#v", requestPath, request.Headers)
	}
	for _, header := range request.Headers {
		if header.Name == "Authorization" && header.Env == "" {
			t.Fatalf("expected sensitive authorization header to re-import as env reference, got %#v", header)
		}
	}
}

func assertRoundTripFlow(t *testing.T, flowPath string) {
	t.Helper()
	flow, err := storage.LoadFlow(flowPath)
	if err != nil {
		t.Fatalf("expected canonical flow at %s: %v", flowPath, err)
	}
	if flow.Name != "Create Payment" {
		t.Fatalf("expected flow name preserved, got %#v", flow)
	}
	if len(flow.Steps) != 1 || flow.Steps[0].Request != "requests/payments/create-payment.request.yaml" || !flow.Steps[0].Trace {
		t.Fatalf("expected single-step flow preserved, got %#v", flow.Steps)
	}
}

func writeRoundTripWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	project := model.Project{
		Version:      "v1",
		Name:         "roundtrip",
		DefaultEnv:   "uat",
		Environments: []string{"envs/uat.yaml"},
		Requests:     []string{"requests/payments/create-payment.request.yaml"},
		Flows:        []string{"flows/payments/create-payment.flow.yaml"},
	}
	environment := model.Environment{
		Version: "v1",
		Name:    "uat",
		BaseURL: "https://uat.api.example.test",
		Oracle: model.OracleDatabase{
			DSNEnv:      "FLOWMAN_UAT_ORACLE_DSN",
			UserEnv:     "FLOWMAN_UAT_ORACLE_USER",
			PasswordEnv: "FLOWMAN_UAT_ORACLE_PASSWORD",
		},
		Trace: model.TraceConfig{
			TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"},
			PollInterval:  model.NewDuration(2 * time.Second),
			Timeout:       model.NewDuration(30 * time.Second),
		},
	}
	request := model.Request{
		Version: "v1",
		Name:    "Create Payment",
		Method:  "POST",
		Path:    "/payments",
		Query:   []model.Parameter{{Name: "dry_run", Value: "false"}, {Name: "source", Value: "flowman"}},
		Headers: []model.Header{{Name: "Authorization", Env: "FLOWMAN_AUTHORIZATION"}, {Name: "Content-Type", Value: "application/json"}},
		Body:    model.RequestBody{Mode: "json", Raw: "{\n  \"amount\": \"12.34\"\n}"},
		Trace: model.TraceConfig{
			TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"},
			PollInterval:  model.NewDuration(2 * time.Second),
			Timeout:       model.NewDuration(30 * time.Second),
		},
	}
	flow := model.Flow{
		Version: "v1",
		Name:    "Create Payment",
		Steps: []model.FlowStep{{
			Name:    "Create Payment",
			Request: "requests/payments/create-payment.request.yaml",
			Trace:   true,
		}},
	}
	if err := storage.WriteCanonical(filepath.Join(root, "flowman.yaml"), project); err != nil {
		t.Fatalf("expected project fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "envs", "uat.yaml"), environment); err != nil {
		t.Fatalf("expected env fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "requests", "payments", "create-payment.request.yaml"), request); err != nil {
		t.Fatalf("expected request fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "flows", "payments", "create-payment.flow.yaml"), flow); err != nil {
		t.Fatalf("expected flow fixture write to succeed: %v", err)
	}
	return root
}
