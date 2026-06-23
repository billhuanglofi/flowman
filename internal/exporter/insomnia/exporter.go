package insomnia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	postmanexport "github.com/billhuanglofi/flowman/internal/exporter/postman"
	"github.com/billhuanglofi/flowman/internal/model"
	"gopkg.in/yaml.v3"
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

var ErrStrictWarnings = errors.New("insomnia export: strict mode rejected warnings")

type exportDocument struct {
	Type         string           `json:"_type" yaml:"_type"`
	ExportFormat int              `json:"__export_format" yaml:"__export_format"`
	Resources    []exportResource `json:"resources" yaml:"resources"`
}

type exportResource struct {
	ID             string            `json:"_id" yaml:"_id"`
	Type           string            `json:"_type" yaml:"_type"`
	ParentID       string            `json:"parentId,omitempty" yaml:"parentId,omitempty"`
	Name           string            `json:"name,omitempty" yaml:"name,omitempty"`
	Method         string            `json:"method,omitempty" yaml:"method,omitempty"`
	URL            string            `json:"url,omitempty" yaml:"url,omitempty"`
	Parameters     []exportParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Headers        []exportHeader    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body           *exportBody       `json:"body,omitempty" yaml:"body,omitempty"`
	Authentication map[string]any    `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

type exportParameter struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type exportHeader struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type exportBody struct {
	MimeType string            `json:"mimeType,omitempty" yaml:"mimeType,omitempty"`
	Text     string            `json:"text,omitempty" yaml:"text,omitempty"`
	Params   []exportParameter `json:"params,omitempty" yaml:"params,omitempty"`
}

func ExportWorkspace(_ context.Context, options ExportOptions) (ExportResult, error) {
	workspace, err := postmanexport.LoadWorkspaceForReuse(options.WorkspacePath)
	if err != nil {
		return ExportResult{}, err
	}
	document, warnings := buildDocument(workspace)
	content, err := marshalDocument(options.OutputPath, document)
	if err != nil {
		return ExportResult{}, err
	}
	if err := writeWarningReport(options.ReportPath, warnings); err != nil {
		return ExportResult{}, err
	}
	if options.Strict && len(warnings) > 0 {
		return ExportResult{ExportedCount: len(workspace.Requests), OutputPath: options.OutputPath, Warnings: warnings, ReportPath: options.ReportPath}, ErrStrictWarnings
	}
	if err := os.MkdirAll(filepath.Dir(options.OutputPath), 0o755); err != nil {
		return ExportResult{}, fmt.Errorf("create insomnia export directory for %s: %w", options.OutputPath, err)
	}
	if err := os.WriteFile(options.OutputPath, content, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("write insomnia export %s: %w", options.OutputPath, err)
	}
	return ExportResult{ExportedCount: len(workspace.Requests), OutputPath: options.OutputPath, Warnings: warnings, ReportPath: options.ReportPath}, nil
}

func buildDocument(workspace model.Workspace) (exportDocument, []model.ImportWarning) {
	resources := []exportResource{{ID: "wrk_1", Type: "workspace", Name: workspace.Project.Name}}
	folderIDs := map[string]string{}
	for _, path := range workspace.Project.Requests {
		folders := requestFolders(path)
		parentID := "wrk_1"
		for index := range folders {
			key := strings.Join(folders[:index+1], "/")
			identifier, ok := folderIDs[key]
			if !ok {
				identifier = "fld_" + strings.ReplaceAll(strings.ToLower(key), "/", "_")
				folderIDs[key] = identifier
				resources = append(resources, exportResource{ID: identifier, Type: "request_group", ParentID: parentID, Name: folders[index]})
			}
			parentID = identifier
		}
	}
	for index, request := range workspace.Requests {
		resources = append(resources, buildRequestResource(workspace.Project.Requests[index], index, request))
	}
	return exportDocument{Type: "export", ExportFormat: 4, Resources: resources}, flowWarnings(workspace)
}

func buildRequestResource(requestPath string, index int, request model.Request) exportResource {
	parentID := "wrk_1"
	folders := requestFolders(requestPath)
	if len(folders) > 0 {
		parentID = "fld_" + strings.ReplaceAll(strings.ToLower(strings.Join(folders, "/")), "/", "_")
	}
	headers, auth := exportHeaders(request.Headers)
	resource := exportResource{
		ID:         "req_" + strconv.Itoa(index),
		Type:       "request",
		ParentID:   parentID,
		Name:       request.Name,
		Method:     normalizeMethod(request.Method),
		URL:        exportURL(request),
		Parameters: exportParameters(request.Query),
		Headers:    headers,
		Body:       exportBodyShape(request.Body),
	}
	if auth != nil {
		resource.Authentication = auth
	}
	return resource
}

func requestFolders(path string) []string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) <= 2 {
		return nil
	}
	return append([]string(nil), parts[1:len(parts)-1]...)
}

func exportHeaders(headers []model.Header) ([]exportHeader, map[string]any) {
	result := make([]exportHeader, 0, len(headers))
	var auth map[string]any
	for _, header := range sortedHeaders(headers) {
		if header.Env != "" && strings.EqualFold(header.Name, "Authorization") {
			auth = map[string]any{"type": "bearer", "token": "{{ " + header.Env + " }}"}
			continue
		}
		value := header.Value
		if header.Env != "" {
			value = "{{ " + header.Env + " }}"
		}
		result = append(result, exportHeader{Name: header.Name, Value: value})
	}
	return result, auth
}

func exportParameters(parameters []model.Parameter) []exportParameter {
	result := make([]exportParameter, 0, len(parameters))
	for _, parameter := range sortedParameters(parameters) {
		result = append(result, exportParameter{Name: parameter.Name, Value: parameter.Value})
	}
	return result
}

func exportBodyShape(body model.RequestBody) *exportBody {
	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	switch mode {
	case "", "raw":
		if strings.TrimSpace(body.Raw) == "" {
			return nil
		}
		return &exportBody{Text: body.Raw}
	case "json":
		return &exportBody{MimeType: "application/json", Text: body.Raw}
	case "form":
		return &exportBody{MimeType: "application/x-www-form-urlencoded", Params: exportParameters(body.Form)}
	default:
		return nil
	}
}

func exportURL(request model.Request) string {
	if strings.TrimSpace(request.URL) != "" {
		return request.URL
	}
	return request.Path
}

func sortedParameters(parameters []model.Parameter) []model.Parameter {
	sorted := append([]model.Parameter(nil), parameters...)
	sort.Slice(sorted, func(left int, right int) bool {
		if sorted[left].Name == sorted[right].Name {
			return sorted[left].Value < sorted[right].Value
		}
		return sorted[left].Name < sorted[right].Name
	})
	return sorted
}

func sortedHeaders(headers []model.Header) []model.Header {
	sorted := append([]model.Header(nil), headers...)
	sort.Slice(sorted, func(left int, right int) bool {
		if sorted[left].Name == sorted[right].Name {
			return sorted[left].Value < sorted[right].Value
		}
		return sorted[left].Name < sorted[right].Name
	})
	return sorted
}

func normalizeMethod(method string) string {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "GET"
	}
	return strings.ToUpper(trimmed)
}

func flowWarnings(workspace model.Workspace) []model.ImportWarning {
	warnings := make([]model.ImportWarning, 0)
	requestPaths := map[string]bool{}
	for _, path := range workspace.Project.Requests {
		requestPaths[path] = true
	}
	for flowIndex, flow := range workspace.Flows {
		if len(flow.Steps) > 1 {
			warnings = append(warnings, model.ImportWarning{Code: "insomnia-flow-multistep-unsupported", Path: fmt.Sprintf("$.flows[%d].steps", flowIndex), Message: "Insomnia export keeps request groups but cannot represent multi-step Flowman flow sequencing losslessly."})
		}
		for stepIndex, step := range flow.Steps {
			if !requestPaths[step.Request] {
				warnings = append(warnings, model.ImportWarning{Code: "insomnia-flow-request-missing", Path: fmt.Sprintf("$.flows[%d].steps[%d].request", flowIndex, stepIndex), Message: "Flow step references a request path that is not part of the workspace request list."})
			}
		}
	}
	return warnings
}

func marshalDocument(path string, document exportDocument) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		content, err := yaml.Marshal(document)
		if err != nil {
			return nil, fmt.Errorf("marshal insomnia yaml export: %w", err)
		}
		return content, nil
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal insomnia json export: %w", err)
	}
	return append(content, '\n'), nil
}

func writeWarningReport(path string, warnings []model.ImportWarning) error {
	if path == "" {
		return nil
	}
	payload := struct {
		Warnings []model.ImportWarning `json:"warnings"`
	}{Warnings: warnings}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal insomnia warning report: %w", err)
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create warning report directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write warning report %s: %w", path, err)
	}
	return nil
}
