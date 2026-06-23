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

	"github.com/billhuanglofi/flowman/internal/importer/insomnia"
	"github.com/billhuanglofi/flowman/internal/model"
)

func Test_ImportInsomniaCommand_writes_json_report_and_strict_failure(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	documentPath := filepath.Join(t.TempDir(), "payments.insomnia.json")
	if err := os.WriteFile(documentPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("expected document fixture write to succeed: %v", err)
	}
	outputDir := filepath.Join(t.TempDir(), "workspace")
	reportPath := filepath.Join(t.TempDir(), "report.json")
	restore := stubInsomniaImporter(func(_ context.Context, options insomnia.ImportOptions) (insomnia.ImportResult, error) {
		result := insomnia.ImportResult{Warnings: []model.ImportWarning{{Code: "insomnia-cookie-jar-unsupported", Path: "$.resources[4].cookieJar", Message: "Cookie jars are not canonical Flowman source and were not imported."}}, ReportPath: reportPath}
		if options.Strict {
			return result, insomnia.ErrStrictWarnings
		}
		result.ImportedCount = 2
		result.RequestPaths = []string{filepath.Join(outputDir, "requests", "payments", "create-payment.request.yaml")}
		result.FlowPaths = []string{filepath.Join(outputDir, "flows", "payments", "create-payment.flow.yaml")}
		return result, nil
	})
	defer restore()

	// When
	defaultErr := Execute(ctx, []string{"import", "insomnia", documentPath, "--out", outputDir, "--report", reportPath}, stdout, stderr)
	defaultOutput := stdout.String()
	stdout.Reset()
	strictErr := Execute(ctx, []string{"import", "insomnia", documentPath, "--out", outputDir, "--report", reportPath, "--strict"}, stdout, stderr)

	// Then
	if defaultErr != nil {
		t.Fatalf("expected default insomnia import to succeed: %v", defaultErr)
	}
	var defaultReport insomniaImportReport
	if err := json.Unmarshal([]byte(defaultOutput), &defaultReport); err != nil {
		t.Fatalf("expected JSON report, got %q: %v", defaultOutput, err)
	}
	if defaultReport.ImportedCount != 2 || defaultReport.OutputDir != outputDir || defaultReport.ReportPath != reportPath {
		t.Fatalf("expected insomnia report metadata, got %#v", defaultReport)
	}
	if !errors.Is(strictErr, insomnia.ErrStrictWarnings) {
		t.Fatalf("expected strict warning error, got %v", strictErr)
	}
	var strictReport insomniaImportReport
	if err := json.Unmarshal(stdout.Bytes(), &strictReport); err != nil {
		t.Fatalf("expected strict JSON report, got %q: %v", stdout.String(), err)
	}
	if strictReport.ImportedCount != 0 || len(strictReport.Warnings) != 1 || !strictReport.Strict {
		t.Fatalf("expected strict failure report with warnings, got %#v", strictReport)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if strings.Contains(defaultOutput+stdout.String(), "secret") {
		t.Fatalf("expected CLI output to stay redacted, got %q", defaultOutput+stdout.String())
	}
}

func stubInsomniaImporter(importer func(context.Context, insomnia.ImportOptions) (insomnia.ImportResult, error)) func() {
	previous := importInsomniaWorkspace
	importInsomniaWorkspace = importer
	return func() {
		importInsomniaWorkspace = previous
	}
}
