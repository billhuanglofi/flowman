package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

type RenderSize struct {
	Width  int
	Height int
}

type ResponsePreview struct {
	State         string
	Status        string
	TransactionID string
	Duration      string
}

type Preview struct {
	Workspace   model.Workspace
	SelectedEnv string
	Selected    int
	Response    ResponsePreview
	RunState    string
	TraceState  string
	Journey     flowtrace.Journey
	LoadError   error
}

type PreviewOptions struct {
	EnvName  string
	Response runner.Response
	Journey  flowtrace.Journey
}

func PreviewFromWorkspace(workspace model.Workspace, options PreviewOptions) Preview {
	envName := strings.TrimSpace(options.EnvName)
	if envName == "" {
		envName = workspace.Project.DefaultEnv
	}
	preview := Preview{
		Workspace:   workspace,
		SelectedEnv: envName,
		Response:    placeholderResponse(),
		RunState:    "ready to run selected request",
		TraceState:  "trace waiting for transaction id",
		Journey:     options.Journey,
	}
	if len(workspace.Requests) == 0 {
		preview.RunState = "no requests found in config"
	}
	if options.Response.StatusCode > 0 || len(options.Response.Body) > 0 || len(options.Response.Headers) > 0 {
		preview.Response = responseFromCore(workspace, preview.Selected, options.Response)
		preview.TraceState = "trace preview loaded from in-memory journey"
	}
	return preview
}

func PreviewError(envName string, err error) Preview {
	return Preview{SelectedEnv: envName, LoadError: err, Response: placeholderResponse()}
}

func selectedEnvironment(workspace model.Workspace, envName string) (model.Environment, bool) {
	for _, environment := range workspace.Environments {
		if environment.Name == envName {
			return environment, true
		}
	}
	return model.Environment{}, false
}

func selectedRequest(workspace model.Workspace, index int) (model.Request, bool) {
	if index < 0 || index >= len(workspace.Requests) {
		return model.Request{}, false
	}
	return workspace.Requests[index], true
}

func responseFromCore(workspace model.Workspace, selected int, response runner.Response) ResponsePreview {
	request, ok := selectedRequest(workspace, selected)
	if !ok {
		return placeholderResponse()
	}
	tx, err := flowtrace.ExtractTransactionID(flowtrace.ExtractionInput{Config: request.Trace, Response: response})
	if err != nil {
		tx = "pending"
	}
	return ResponsePreview{
		State:         "complete",
		Status:        fmt.Sprintf("Status: %d", response.StatusCode),
		TransactionID: tx,
		Duration:      response.Duration.String(),
	}
}

func placeholderResponse() ResponsePreview {
	return ResponsePreview{State: "not run", Status: "response pending", TransactionID: "pending", Duration: "pending"}
}

func safeText(value string) string {
	return strings.Map(func(char rune) rune {
		if char == '\n' || char == '\t' {
			return ' '
		}
		if unicode.IsControl(char) {
			return -1
		}
		return char
	}, value)
}
