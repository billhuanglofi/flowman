package insomnia

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

var ErrStrictWarnings = errors.New("insomnia import: strict mode rejected warnings")

type ImportOptions struct {
	DocumentPath string
	OutputDir    string
	ReportPath   string
	Strict       bool
}

type ImportResult struct {
	ImportedCount int                   `json:"imported_count"`
	RequestPaths  []string              `json:"request_paths,omitempty"`
	FlowPaths     []string              `json:"flow_paths,omitempty"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath    string                `json:"report_path,omitempty"`
}

func (result ImportResult) WarningText() string {
	var builder strings.Builder
	for _, warning := range result.Warnings {
		builder.WriteString(warning.Code)
		builder.WriteString(warning.Path)
		builder.WriteString(warning.Message)
	}
	return builder.String()
}

func ImportWorkspace(_ context.Context, options ImportOptions) (ImportResult, error) {
	doc, err := loadDocument(options.DocumentPath)
	if err != nil {
		return ImportResult{}, err
	}
	files, warnings := buildPlan(doc)
	result := ImportResult{Warnings: warnings, ReportPath: options.ReportPath}
	if err := writeWarningReport(options.ReportPath, warnings); err != nil {
		return ImportResult{}, err
	}
	if options.Strict && len(warnings) > 0 {
		return result, ErrStrictWarnings
	}
	for _, file := range files {
		requestPath := filepath.Join(options.OutputDir, file.requestPath)
		flowPath := filepath.Join(options.OutputDir, file.flowPath)
		if err := storage.WriteCanonical(requestPath, file.request); err != nil {
			return ImportResult{}, fmt.Errorf("write insomnia request %s: %w", requestPath, err)
		}
		if err := storage.WriteCanonical(flowPath, file.flow); err != nil {
			return ImportResult{}, fmt.Errorf("write insomnia flow %s: %w", flowPath, err)
		}
		result.RequestPaths = append(result.RequestPaths, requestPath)
		result.FlowPaths = append(result.FlowPaths, flowPath)
	}
	result.ImportedCount = len(files)
	return result, nil
}
