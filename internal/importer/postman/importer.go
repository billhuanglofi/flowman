package postman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

var ErrStrictWarnings = errors.New("postman import: strict mode rejected warnings")

type ImportOptions struct {
	CollectionPath string
	OutputDir      string
	ReportPath     string
	Strict         bool
}

type ImportResult struct {
	ImportedCount int                   `json:"imported_count"`
	RequestPaths  []string              `json:"request_paths,omitempty"`
	FlowPaths     []string              `json:"flow_paths,omitempty"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath    string                `json:"report_path,omitempty"`
}

func (result ImportResult) WarningText() string {
	return warningsText(result.Warnings)
}

func ImportCollection(_ context.Context, options ImportOptions) (ImportResult, error) {
	collection, err := loadCollection(options.CollectionPath)
	if err != nil {
		return ImportResult{}, err
	}
	plan, err := buildPlan(collection)
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{
		Warnings:   append([]model.ImportWarning(nil), plan.warnings...),
		ReportPath: options.ReportPath,
	}
	if err := writeWarningReport(options.ReportPath, result.Warnings); err != nil {
		return ImportResult{}, err
	}
	if options.Strict && len(result.Warnings) > 0 {
		return result, ErrStrictWarnings
	}
	for _, file := range plan.files {
		requestPath := filepath.Join(options.OutputDir, file.requestPath)
		flowPath := filepath.Join(options.OutputDir, file.flowPath)
		if err := storage.WriteCanonical(requestPath, file.request); err != nil {
			return ImportResult{}, fmt.Errorf("write postman request %s: %w", requestPath, err)
		}
		if err := storage.WriteCanonical(flowPath, file.flow); err != nil {
			return ImportResult{}, fmt.Errorf("write postman flow %s: %w", flowPath, err)
		}
		result.RequestPaths = append(result.RequestPaths, requestPath)
		result.FlowPaths = append(result.FlowPaths, flowPath)
	}
	result.ImportedCount = len(plan.files)
	return result, nil
}

func writeWarningReport(path string, warnings []model.ImportWarning) error {
	if path == "" {
		return nil
	}
	content, err := marshalWarningReport(warnings)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create warning report directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write warning report %s: %w", path, err)
	}
	return nil
}
