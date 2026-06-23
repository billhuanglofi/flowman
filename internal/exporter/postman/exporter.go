package postman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/model"
)

type ExportOptions struct {
	WorkspacePath string
	OutputPath    string
	ReportPath    string
	Strict        bool
}

type ExportResult struct {
	ExportedCount int                   `json:"exported_count"`
	OutputPath    string                `json:"output_path"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath    string                `json:"report_path,omitempty"`
}

var ErrStrictWarnings = errors.New("postman export: strict mode rejected warnings")

type collectionEnvelope struct {
	Info collectionInfo   `json:"info"`
	Item []collectionItem `json:"item"`
}

type collectionInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type collectionItem struct {
	Name    string           `json:"name"`
	Item    []collectionItem `json:"item,omitempty"`
	Request *requestShape    `json:"request,omitempty"`
}

type requestShape struct {
	Method string        `json:"method"`
	Header []headerShape `json:"header,omitempty"`
	URL    any           `json:"url,omitempty"`
	Body   *bodyShape    `json:"body,omitempty"`
	Auth   *authShape    `json:"auth,omitempty"`
}

type headerShape struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type bodyShape struct {
	Mode    string        `json:"mode"`
	Raw     string        `json:"raw,omitempty"`
	Options *bodyOptions  `json:"options,omitempty"`
	URLEnc  []headerShape `json:"urlencoded,omitempty"`
}

type bodyOptions struct {
	Raw map[string]string `json:"raw,omitempty"`
}

type authShape struct {
	Type   string      `json:"type"`
	Bearer []authEntry `json:"bearer,omitempty"`
	APIKey []authEntry `json:"apikey,omitempty"`
}

type authEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ExportWorkspace(_ context.Context, options ExportOptions) (ExportResult, error) {
	workspace, err := loadWorkspace(options.WorkspacePath)
	if err != nil {
		return ExportResult{}, err
	}
	entries := buildEntries(workspace)
	warnings := flowWarnings(workspace)
	collection := collectionEnvelope{
		Info: collectionInfo{Name: workspace.Project.Name, Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		Item: nestEntries(entries),
	}
	content, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return ExportResult{}, fmt.Errorf("marshal postman collection: %w", err)
	}
	content = append(content, '\n')
	if err := writeWarningReport(options.ReportPath, warnings); err != nil {
		return ExportResult{}, err
	}
	if options.Strict && len(warnings) > 0 {
		return ExportResult{ExportedCount: len(workspace.Requests), OutputPath: options.OutputPath, Warnings: warnings, ReportPath: options.ReportPath}, ErrStrictWarnings
	}
	if err := os.MkdirAll(filepath.Dir(options.OutputPath), 0o755); err != nil {
		return ExportResult{}, fmt.Errorf("create postman export directory for %s: %w", options.OutputPath, err)
	}
	if err := os.WriteFile(options.OutputPath, content, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write postman export %s: %w", options.OutputPath, err)
	}
	return ExportResult{ExportedCount: len(workspace.Requests), OutputPath: options.OutputPath, Warnings: warnings, ReportPath: options.ReportPath}, nil
}
