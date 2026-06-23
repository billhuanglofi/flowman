package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	insomniaexport "github.com/billhuanglofi/flowman/internal/exporter/insomnia"
	postmanexport "github.com/billhuanglofi/flowman/internal/exporter/postman"
	"github.com/billhuanglofi/flowman/internal/model"
)

func Test_ExportPostmanAndInsomniaCommands_write_json_reports(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	workspacePath := filepath.Join("..", "..", "testdata", "flowman")
	postmanRestore := stubPostmanExporter(func(_ context.Context, options postmanexport.ExportOptions) (postmanexport.ExportResult, error) {
		return postmanexport.ExportResult{ExportedCount: 1, OutputPath: options.OutputPath, Warnings: []model.ImportWarning{{Code: "postman-flow-multistep-unsupported", Path: "$.flows[0].steps", Message: "subset warning"}}, ReportPath: options.ReportPath}, nil
	})
	defer postmanRestore()
	insomniaRestore := stubInsomniaExporter(func(_ context.Context, options insomniaexport.ExportOptions) (insomniaexport.ExportResult, error) {
		return insomniaexport.ExportResult{ExportedCount: 1, OutputPath: options.OutputPath, Warnings: []model.ImportWarning{{Code: "insomnia-flow-multistep-unsupported", Path: "$.flows[0].steps", Message: "subset warning"}}, ReportPath: options.ReportPath}, nil
	})
	defer insomniaRestore()

	// When
	postmanErr := Execute(ctx, []string{"export", "postman", workspacePath, "--out", filepath.Join(t.TempDir(), "collection.json"), "--report", filepath.Join(t.TempDir(), "postman-report.json")}, stdout, stderr)
	postmanOutput := stdout.String()
	stdout.Reset()
	insomniaErr := Execute(ctx, []string{"export", "insomnia", workspacePath, "--out", filepath.Join(t.TempDir(), "collection.insomnia.json"), "--report", filepath.Join(t.TempDir(), "insomnia-report.json")}, stdout, stderr)

	// Then
	if postmanErr != nil || insomniaErr != nil {
		t.Fatalf("expected export commands to succeed, got postman=%v insomnia=%v", postmanErr, insomniaErr)
	}
	var postmanReport exportReport
	if err := json.Unmarshal([]byte(postmanOutput), &postmanReport); err != nil {
		t.Fatalf("expected postman JSON report, got %q: %v", postmanOutput, err)
	}
	var insomniaReport exportReport
	if err := json.Unmarshal(stdout.Bytes(), &insomniaReport); err != nil {
		t.Fatalf("expected insomnia JSON report, got %q: %v", stdout.String(), err)
	}
	if postmanReport.ExportedCount != 1 || insomniaReport.ExportedCount != 1 {
		t.Fatalf("expected export counts in reports, got postman=%#v insomnia=%#v", postmanReport, insomniaReport)
	}
	if postmanReport.Strict || insomniaReport.Strict {
		t.Fatalf("expected default export reports to mark strict=false, got postman=%#v insomnia=%#v", postmanReport, insomniaReport)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if strings.Contains(postmanOutput+stdout.String(), "secret") {
		t.Fatalf("expected export output to stay redacted")
	}
}

func Test_ExportPostmanAndInsomniaCommands_fail_in_strict_mode_on_warnings(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	workspacePath := filepath.Join("..", "..", "testdata", "flowman")
	postmanRestore := stubPostmanExporter(func(_ context.Context, options postmanexport.ExportOptions) (postmanexport.ExportResult, error) {
		result := postmanexport.ExportResult{ExportedCount: 1, OutputPath: options.OutputPath, Warnings: []model.ImportWarning{{Code: "postman-flow-multistep-unsupported", Path: "$.flows[0].steps", Message: "subset warning"}}, ReportPath: options.ReportPath}
		if options.Strict {
			return result, postmanexport.ErrStrictWarnings
		}
		return result, nil
	})
	defer postmanRestore()
	insomniaRestore := stubInsomniaExporter(func(_ context.Context, options insomniaexport.ExportOptions) (insomniaexport.ExportResult, error) {
		result := insomniaexport.ExportResult{ExportedCount: 1, OutputPath: options.OutputPath, Warnings: []model.ImportWarning{{Code: "insomnia-flow-multistep-unsupported", Path: "$.flows[0].steps", Message: "subset warning"}}, ReportPath: options.ReportPath}
		if options.Strict {
			return result, insomniaexport.ErrStrictWarnings
		}
		return result, nil
	})
	defer insomniaRestore()

	// When
	postmanErr := Execute(ctx, []string{"export", "postman", workspacePath, "--out", filepath.Join(t.TempDir(), "collection.json"), "--report", filepath.Join(t.TempDir(), "postman-report.json"), "--strict"}, stdout, stderr)
	postmanOutput := stdout.String()
	stdout.Reset()
	insomniaErr := Execute(ctx, []string{"export", "insomnia", workspacePath, "--out", filepath.Join(t.TempDir(), "collection.insomnia.json"), "--report", filepath.Join(t.TempDir(), "insomnia-report.json"), "--strict"}, stdout, stderr)

	// Then
	if !errors.Is(postmanErr, postmanexport.ErrStrictWarnings) {
		t.Fatalf("expected strict postman warning failure, got %v", postmanErr)
	}
	if !errors.Is(insomniaErr, insomniaexport.ErrStrictWarnings) {
		t.Fatalf("expected strict insomnia warning failure, got %v", insomniaErr)
	}
	var postmanReport exportReport
	if err := json.Unmarshal([]byte(postmanOutput), &postmanReport); err != nil {
		t.Fatalf("expected strict postman JSON report, got %q: %v", postmanOutput, err)
	}
	var insomniaReport exportReport
	if err := json.Unmarshal(stdout.Bytes(), &insomniaReport); err != nil {
		t.Fatalf("expected strict insomnia JSON report, got %q: %v", stdout.String(), err)
	}
	if !postmanReport.Strict || !insomniaReport.Strict {
		t.Fatalf("expected strict flag in reports, got postman=%#v insomnia=%#v", postmanReport, insomniaReport)
	}
	if postmanReport.ExportedCount != 1 || insomniaReport.ExportedCount != 1 {
		t.Fatalf("expected export counts in strict reports, got postman=%#v insomnia=%#v", postmanReport, insomniaReport)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func Test_ExportCommands_return_export_errors_after_writing_report(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	workspacePath := filepath.Join("..", "..", "testdata", "flowman")
	restore := stubPostmanExporter(func(_ context.Context, options postmanexport.ExportOptions) (postmanexport.ExportResult, error) {
		return postmanexport.ExportResult{OutputPath: options.OutputPath}, errors.New("boom")
	})
	defer restore()

	// When
	err := Execute(ctx, []string{"export", "postman", workspacePath, "--out", filepath.Join(t.TempDir(), "collection.json")}, stdout, stderr)

	// Then
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected export failure, got %v", err)
	}
	if stdout.Len() == 0 {
		t.Fatal("expected report output on export failure")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func stubPostmanExporter(exporter func(context.Context, postmanexport.ExportOptions) (postmanexport.ExportResult, error)) func() {
	previous := exportPostmanWorkspace
	exportPostmanWorkspace = exporter
	return func() {
		exportPostmanWorkspace = previous
	}
}

func stubInsomniaExporter(exporter func(context.Context, insomniaexport.ExportOptions) (insomniaexport.ExportResult, error)) func() {
	previous := exportInsomniaWorkspace
	exportInsomniaWorkspace = exporter
	return func() {
		exportInsomniaWorkspace = previous
	}
}
