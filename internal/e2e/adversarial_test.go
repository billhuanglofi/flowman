package e2e_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	postmanimport "github.com/billhuanglofi/flowman/internal/importer/postman"
	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestAdversarialFailures(t *testing.T) {
	// Given
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	outputRoot := t.TempDir()
	strictOutput := filepath.Join(outputRoot, "strict-postman")
	reportPath := filepath.Join(outputRoot, "strict-postman-report.json")
	store := fakeSafestoreStore{rows: []safestore.RequestRow{{TransactionID: "TX-NO-URL", Body: `{"transaction_id":"TX-NO-URL"}`, Headers: `{"Content-Type":"application/json"}`}}}
	environment := model.Environment{Name: "uat"}

	// When
	postmanResult, postmanErr := postmanimport.ImportCollection(ctx, postmanimport.ImportOptions{
		CollectionPath: fixturePath("postman_unsupported_collection.json"),
		OutputDir:      strictOutput,
		ReportPath:     reportPath,
		Strict:         true,
	})
	_, safestoreErr := safestore.ImportReplayRequest(ctx, store, safestore.ImportOptions{
		TransactionID:  "TX-NO-URL",
		Environment:    environment,
		OutputPath:     filepath.Join(outputRoot, "replay", "request.yaml"),
		FlowOutputPath: filepath.Join(outputRoot, "replay", "flow.yaml"),
		NonInteractive: true,
	})

	// Then
	if !errors.Is(postmanErr, postmanimport.ErrStrictWarnings) {
		t.Fatalf("expected strict Postman import to fail on unsupported script warnings, got %v", postmanErr)
	}
	if postmanResult.ImportedCount != 0 {
		t.Fatalf("expected strict Postman import to avoid writes on warnings, got %#v", postmanResult)
	}
	if len(postmanResult.Warnings) == 0 {
		t.Fatalf("expected unsupported Postman script warnings, got %#v", postmanResult)
	}
	if _, err := storage.LoadRequest(filepath.Join(strictOutput, "requests", "payments", "create-payment-with-script.request.yaml")); err == nil {
		t.Fatal("expected strict Postman import to skip canonical request writes")
	}
	reportContent, err := readText(reportPath)
	if err != nil {
		t.Fatalf("expected strict Postman report to be written: %v", err)
	}
	if !strings.Contains(reportContent, "postman-prerequest-script-unsupported") || !strings.Contains(reportContent, "$.item[0].item[0].event[0].script.exec") {
		t.Fatalf("expected strict Postman report to capture unsupported script details, got %s", reportContent)
	}
	assertNoRawBearerExamples(t, reportContent)
	if !errors.Is(safestoreErr, safestore.ErrCannotResolveURLNonInteractive) {
		t.Fatalf("expected noninteractive Safestore import to fail on unresolved URL, got %v", safestoreErr)
	}
	if !strings.Contains(safestoreErr.Error(), "needs --url, --endpoint, or env base_url + --path") {
		t.Fatalf("expected actionable unresolved Safestore URL error, got %v", safestoreErr)
	}
}
