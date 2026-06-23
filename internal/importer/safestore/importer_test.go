package safestore_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestImportReplayRequestDefaultsToPostAndRedactsSecrets(t *testing.T) {
	// Given
	root := t.TempDir()
	outputPath := filepath.Join(root, "requests", "payment", "replay-TX123.request.yaml")
	flowOutputPath := filepath.Join(root, "flows", "payment", "replay-TX123.flow.yaml")
	store := fakeStore{rows: []safestore.RequestRow{{
		TransactionID: "TX123",
		Timestamp:     time.Date(2026, 6, 21, 10, 30, 0, 0, time.UTC),
		Body:          `{"amount":"12.34"}`,
		Headers:       `{"Content-Type":"application/json","Authorization":"Bearer raw-secret","X-API-Key":"api-secret"}`,
	}}}
	environment := model.Environment{
		Name:    "uat",
		BaseURL: "https://uat.api.example.test",
		Endpoints: []model.EndpointAlias{{
			Name: "payments",
			Path: "/payments",
		}},
	}
	options := safestore.ImportOptions{
		TransactionID:  "TX123",
		Environment:    environment,
		Endpoint:       "payments",
		Path:           "/v1/payments",
		OutputPath:     outputPath,
		FlowOutputPath: flowOutputPath,
		NonInteractive: true,
	}

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected import to succeed: %v", err)
	}
	if store.requestedTransactionID != "TX123" {
		t.Fatalf("expected store queried by transaction id, got %q", store.requestedTransactionID)
	}
	if result.Request.Method != "POST" {
		t.Fatalf("expected Safestore import to default to POST, got %q", result.Request.Method)
	}
	if result.Request.Body.Mode != "json" || result.Request.Body.Raw != `{"amount":"12.34"}` {
		t.Fatalf("expected Safestore body to be preserved as raw JSON, got %#v", result.Request.Body)
	}
	assertHeaderValue(t, result.Request.Headers, "Content-Type", "application/json")
	assertHeaderEnv(t, result.Request.Headers, "Authorization", "FLOWMAN_IMPORTED_TX123_AUTHORIZATION")
	assertHeaderEnv(t, result.Request.Headers, "X-API-Key", "FLOWMAN_IMPORTED_TX123_X_API_KEY")
	if len(result.Request.ImportWarnings) != 2 {
		t.Fatalf("expected two redaction warnings, got %#v", result.Request.ImportWarnings)
	}
	if strings.Contains(warningsText(result.Request.ImportWarnings), "raw-secret") || strings.Contains(warningsText(result.Request.ImportWarnings), "api-secret") {
		t.Fatalf("expected warnings to omit sensitive values, got %#v", result.Request.ImportWarnings)
	}

	written, err := storage.LoadRequest(outputPath)
	if err != nil {
		t.Fatalf("expected canonical YAML request to load: %v", err)
	}
	if written.Method != "POST" || written.Endpoint != "payments" || written.Path != "/v1/payments" {
		t.Fatalf("expected canonical route fields, got %#v", written)
	}
	assertHeaderEnv(t, written.Headers, "Authorization", "FLOWMAN_IMPORTED_TX123_AUTHORIZATION")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected written YAML to be readable: %v", err)
	}
	if strings.Contains(string(content), "raw-secret") || strings.Contains(string(content), "api-secret") {
		t.Fatalf("expected canonical YAML to omit raw sensitive values:\n%s", string(content))
	}
	flow, err := storage.LoadFlow(flowOutputPath)
	if err != nil {
		t.Fatalf("expected canonical YAML flow to load: %v", err)
	}
	if flow.Name != "replay-TX123" || len(flow.Steps) != 1 || flow.Steps[0].Request != "requests/payment/replay-TX123.request.yaml" {
		t.Fatalf("expected replay flow to point at canonical request, got %#v", flow)
	}
	if result.Flow.Name != flow.Name || result.FlowPath != flowOutputPath {
		t.Fatalf("expected import result to include written flow, got %#v", result)
	}
}

func TestImportReplayRequestPreservesMethodOverride(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{{TransactionID: "TX124", Headers: `{}`, Body: `{}`}}}
	options := validOptions(t, "TX124")
	options.Method = "PUT"

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected import to succeed: %v", err)
	}
	if result.Request.Method != "PUT" {
		t.Fatalf("expected method override PUT, got %q", result.Request.Method)
	}
}

func TestImportReplayRequestFailsWhenHeadersJSONInvalid(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{{TransactionID: "TX125", Headers: `{"Content-Type":`, Body: `{}`}}}

	// When
	_, err := safestore.ImportReplayRequest(context.Background(), &store, validOptions(t, "TX125"))

	// Then
	if !errors.Is(err, safestore.ErrInvalidHeaderJSON) {
		t.Fatalf("expected invalid header JSON error, got %v", err)
	}
}

func TestImportReplayRequestFailsWhenURLUnresolvedNonInteractive(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{{TransactionID: "TX126", Headers: `{}`, Body: `{}`}}}
	options := validOptions(t, "TX126")
	options.URL = ""
	options.Endpoint = ""
	options.Path = ""
	options.Environment.BaseURL = ""

	// When
	_, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if !errors.Is(err, safestore.ErrCannotResolveURLNonInteractive) {
		t.Fatalf("expected unresolved noninteractive URL error, got %v", err)
	}
	if !errors.Is(err, safestore.ErrCannotResolveURL) {
		t.Fatalf("expected unresolved URL error, got %v", err)
	}
}

func TestImportReplayRequestFailsWhenURLUnresolvedInteractiveMode(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{{TransactionID: "TX126A", Headers: `{}`, Body: `{}`}}}
	options := validOptions(t, "TX126A")
	options.URL = ""
	options.Endpoint = ""
	options.Path = ""
	options.Environment.BaseURL = ""
	options.NonInteractive = false

	// When
	_, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if !errors.Is(err, safestore.ErrInteractiveURLResolutionNotImplemented) {
		t.Fatalf("expected interactive URL resolution error, got %v", err)
	}
	if !errors.Is(err, safestore.ErrCannotResolveURL) {
		t.Fatalf("expected unresolved URL error, got %v", err)
	}
	if strings.Contains(err.Error(), "--non-interactive") {
		t.Fatalf("expected interactive error message to avoid noninteractive wording, got %v", err)
	}
}

func TestImportReplayRequestFailsWhenMultipleRowsNeedSelection(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{
		{TransactionID: "TX127", Timestamp: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), Headers: `{}`, Body: `first`},
		{TransactionID: "TX127", Timestamp: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC), Headers: `{}`, Body: `latest`},
	}}

	// When
	_, err := safestore.ImportReplayRequest(context.Background(), &store, validOptions(t, "TX127"))

	// Then
	if !errors.Is(err, safestore.ErrMultipleRequestRows) {
		t.Fatalf("expected multiple rows error, got %v", err)
	}
}

func TestImportReplayRequestSelectsLatestWhenRequested(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{
		{TransactionID: "TX128", Timestamp: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), Headers: `{}`, Body: `first`},
		{TransactionID: "TX128", Timestamp: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC), Headers: `{}`, Body: `latest`},
	}}
	options := validOptions(t, "TX128")
	options.Selection = safestore.SelectLatest

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected latest selection to succeed: %v", err)
	}
	if result.Request.Body.Raw != "latest" {
		t.Fatalf("expected latest body, got %q", result.Request.Body.Raw)
	}
}

func TestImportReplayRequestFailsWhenOutputExistsWithoutOverwrite(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{{TransactionID: "TX129", Headers: `{}`, Body: `{}`}}}
	options := validOptions(t, "TX129")
	if err := os.WriteFile(options.OutputPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("expected existing output fixture write to succeed: %v", err)
	}

	// When
	_, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if !errors.Is(err, safestore.ErrOutputExists) {
		t.Fatalf("expected existing output error, got %v", err)
	}
}

type fakeStore struct {
	rows                   []safestore.RequestRow
	requestedTransactionID string
}

func (store *fakeStore) RequestRows(_ context.Context, transactionID string) ([]safestore.RequestRow, error) {
	store.requestedTransactionID = transactionID
	return store.rows, nil
}

func validOptions(t *testing.T, transactionID string) safestore.ImportOptions {
	t.Helper()
	return safestore.ImportOptions{
		TransactionID: transactionID,
		Environment: model.Environment{
			Name:    "uat",
			BaseURL: "https://uat.api.example.test",
		},
		Path:           "/replay",
		OutputPath:     filepath.Join(t.TempDir(), "replay.request.yaml"),
		NonInteractive: true,
	}
}

func assertHeaderValue(t *testing.T, headers []model.Header, name string, want string) {
	t.Helper()
	for _, header := range headers {
		if header.Name == name {
			if header.Value != want {
				t.Fatalf("expected header %s value %q, got %#v", name, want, header)
			}
			return
		}
	}
	t.Fatalf("expected header %s in %#v", name, headers)
}

func assertHeaderEnv(t *testing.T, headers []model.Header, name string, want string) {
	t.Helper()
	for _, header := range headers {
		if header.Name == name {
			if header.Env != want || header.Value != "" {
				t.Fatalf("expected header %s env %q and empty value, got %#v", name, want, header)
			}
			return
		}
	}
	t.Fatalf("expected header %s in %#v", name, headers)
}

func warningsText(warnings []model.ImportWarning) string {
	var builder strings.Builder
	for _, warning := range warnings {
		builder.WriteString(warning.Code)
		builder.WriteString(warning.Path)
		builder.WriteString(warning.Message)
	}
	return builder.String()
}
