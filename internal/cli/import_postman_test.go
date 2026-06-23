package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/postman"
	"github.com/billhuanglofi/flowman/internal/model"
)

func Test_ImportPostmanCommand_writes_json_report_and_strict_failure(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	collectionPath := filepath.Join(t.TempDir(), "payments.postman_collection.json")
	if err := os.WriteFile(collectionPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("expected collection fixture write to succeed: %v", err)
	}
	outputDir := filepath.Join(t.TempDir(), "workspace")
	reportPath := filepath.Join(t.TempDir(), "report.json")
	restore := stubPostmanImporter(func(_ context.Context, options postman.ImportOptions) (postman.ImportResult, error) {
		result := postman.ImportResult{
			ImportedCount: 0,
			Warnings: []model.ImportWarning{{
				Code:    "postman-test-script-unsupported",
				Path:    "$.item[0].event[0].script.exec",
				Message: "Test scripts are unsupported and were reported instead of discarded.",
			}},
			ReportPath: reportPath,
		}
		if options.Strict {
			return result, postman.ErrStrictWarnings
		}
		result.ImportedCount = 2
		result.RequestPaths = []string{filepath.Join(outputDir, "requests", "payments", "create-payment.request.yaml")}
		result.FlowPaths = []string{filepath.Join(outputDir, "flows", "payments", "create-payment.flow.yaml")}
		return result, nil
	})
	defer restore()

	// When
	defaultErr := Execute(ctx, []string{"import", "postman", collectionPath, "--out", outputDir, "--report", reportPath}, stdout, stderr)
	defaultOutput := stdout.String()
	stdout.Reset()
	strictErr := Execute(ctx, []string{"import", "postman", collectionPath, "--out", outputDir, "--report", reportPath, "--strict"}, stdout, stderr)

	// Then
	if defaultErr != nil {
		t.Fatalf("expected default postman import to succeed: %v", defaultErr)
	}
	var defaultReport postmanImportReport
	if err := json.Unmarshal([]byte(defaultOutput), &defaultReport); err != nil {
		t.Fatalf("expected JSON report, got %q: %v", defaultOutput, err)
	}
	if defaultReport.ImportedCount != 2 || defaultReport.OutputDir != outputDir || defaultReport.ReportPath != reportPath {
		t.Fatalf("expected postman report metadata, got %#v", defaultReport)
	}
	if !errors.Is(strictErr, postman.ErrStrictWarnings) {
		t.Fatalf("expected strict warning error, got %v", strictErr)
	}
	var strictReport postmanImportReport
	if err := json.Unmarshal(stdout.Bytes(), &strictReport); err != nil {
		t.Fatalf("expected strict JSON report, got %q: %v", stdout.String(), err)
	}
	if strictReport.ImportedCount != 0 || len(strictReport.Warnings) != 1 || !strictReport.Strict {
		t.Fatalf("expected strict failure report with warnings, got %#v", strictReport)
	}
	combined := defaultOutput + stdout.String() + stderr.String()
	if strings.Contains(combined, "secret") {
		t.Fatalf("expected CLI output to stay redacted, got %q", combined)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func stubPostmanImporter(importer func(context.Context, postman.ImportOptions) (postman.ImportResult, error)) func() {
	previous := importPostmanCollection
	importPostmanCollection = importer
	return func() {
		importPostmanCollection = previous
	}
}
