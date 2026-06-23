package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

const (
	statusReady     = "ready"
	statusRunning   = "running"
	statusTracing   = "tracing"
	statusTypingTX  = "typing transaction id"
	statusImporting = "importing safestore"
)

type WorkflowServices interface {
	RunRequest(ctx context.Context, execution runner.Execution) (runner.Response, error)
	TraceJourney(ctx context.Context, request flowtrace.ProcessStateRequest) (flowtrace.Journey, error)
	ImportSafestore(ctx context.Context, options safestore.ImportOptions) (safestore.ImportResult, error)
}

type WorkflowOptions struct {
	EnvName        string
	Services       WorkflowServices
	ImportURL      string
	ImportEndpoint string
	ImportPath     string
	ImportOutput   string
}

type Model struct {
	preview        Preview
	size           RenderSize
	services       WorkflowServices
	environment    model.Environment
	status         string
	message        string
	inputMode      inputMode
	inputBuffer    string
	importURL      string
	importEndpoint string
	importPath     string
	importOutput   string
}

type inputMode string

const (
	inputModeNone   inputMode = "none"
	inputModeTrace  inputMode = "trace"
	inputModeImport inputMode = "import"
)

type runCompleteMsg struct {
	response      runner.Response
	transactionID string
	err           error
}

type traceCompleteMsg struct {
	journey flowtrace.Journey
	err     error
}

type importCompleteMsg struct {
	result safestore.ImportResult
	err    error
}

func NewModel(preview Preview) Model {
	environment, _ := selectedEnvironment(preview.Workspace, preview.SelectedEnv)
	return Model{preview: preview, size: RenderSize{Width: 100, Height: 32}, environment: environment, status: statusReady, inputMode: inputModeNone}
}

func NewWorkflowModel(workspace model.Workspace, options WorkflowOptions) Model {
	preview := PreviewFromWorkspace(workspace, PreviewOptions{EnvName: options.EnvName})
	environment, _ := selectedEnvironment(workspace, preview.SelectedEnv)
	return Model{
		preview:        preview,
		size:           RenderSize{Width: 100, Height: 32},
		services:       options.Services,
		environment:    environment,
		status:         statusReady,
		message:        "r run | t trace tx | i import safestore | up/down select | q quit",
		inputMode:      inputModeNone,
		importURL:      options.ImportURL,
		importEndpoint: options.ImportEndpoint,
		importPath:     options.ImportPath,
		importOutput:   defaultImportOutput(options.ImportOutput),
	}
}

func (model Model) Init() tea.Cmd {
	return nil
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.size = RenderSize{Width: msg.Width, Height: msg.Height}
	case tea.KeyPressMsg:
		return model.handleKey(msg)
	case runCompleteMsg:
		return model.handleRunComplete(msg)
	case traceCompleteMsg:
		return model.handleTraceComplete(msg)
	case importCompleteMsg:
		return model.handleImportComplete(msg)
	}
	return model, nil
}

func (model Model) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.SetContent(RenderPreview(model.preview, model.size, model.viewState()))
	return view
}

func (model Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if model.inputMode != inputModeNone {
		return model.handleInputKey(msg)
	}
	key := msg.String()
	if msg.Key().Text != "" {
		key = msg.Key().Text
	}
	code := msg.Key().Code
	if code == 0 && len(msg.Key().Text) == 1 {
		code = []rune(msg.Key().Text)[0]
	}
	switch code {
	case 'r':
		key = "r"
	case 't':
		key = "t"
	case 'i':
		key = "i"
	}
	switch key {
	case "q", "ctrl+c":
		return model, tea.Quit
	case "up":
		model.moveSelection(-1)
	case "down":
		model.moveSelection(1)
	case "r":
		return model.startRun()
	case "t":
		model.inputMode = inputModeTrace
		model.inputBuffer = ""
		model.status = statusTypingTX
		model.message = "Enter transaction id, then press enter to trace"
	case "i":
		model.inputMode = inputModeImport
		model.inputBuffer = ""
		model.status = statusImporting
		model.message = "Enter transaction id, then press enter to import Safestore request"
	}
	return model, nil
}

func (model Model) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		model.inputMode = inputModeNone
		model.inputBuffer = ""
		model.status = statusReady
		model.message = "Input cancelled"
		return model, nil
	case "backspace":
		if model.inputBuffer != "" {
			model.inputBuffer = model.inputBuffer[:len(model.inputBuffer)-1]
		}
		return model, nil
	case "enter":
		value := strings.TrimSpace(model.inputBuffer)
		mode := model.inputMode
		model.inputMode = inputModeNone
		model.inputBuffer = ""
		if value == "" {
			model.status = statusReady
			model.message = "Transaction id is required"
			return model, nil
		}
		if mode == inputModeTrace {
			return model.startTrace(value)
		}
		return model.startImport(value)
	default:
		if msg.Key().Text != "" {
			model.inputBuffer += msg.Key().Text
		}
		return model, nil
	}
}

func (model *Model) moveSelection(delta int) {
	if len(model.preview.Workspace.Requests) == 0 {
		return
	}
	next := model.preview.Selected + delta
	if next < 0 {
		next = len(model.preview.Workspace.Requests) - 1
	}
	if next >= len(model.preview.Workspace.Requests) {
		next = 0
	}
	model.preview.Selected = next
	model.message = "Selected request changed"
}

func (model Model) startRun() (tea.Model, tea.Cmd) {
	return startRun(model)
}

func (model Model) handleRunComplete(msg runCompleteMsg) (tea.Model, tea.Cmd) {
	return handleRunComplete(model, msg)
}

func (model Model) startTrace(transactionID string) (tea.Model, tea.Cmd) {
	return startTrace(model, transactionID)
}

func (model Model) handleTraceComplete(msg traceCompleteMsg) (tea.Model, tea.Cmd) {
	return handleTraceComplete(model, msg)
}

func (model Model) startImport(transactionID string) (tea.Model, tea.Cmd) {
	return startImport(model, transactionID)
}

func (model Model) handleImportComplete(msg importCompleteMsg) (tea.Model, tea.Cmd) {
	return handleImportComplete(model, msg)
}

func (model Model) viewState() ViewState {
	return ViewState{
		Status:         model.status,
		Message:        model.message,
		InputMode:      string(model.inputMode),
		InputValue:     model.inputBuffer,
		ImportURL:      model.importURL,
		ImportEndpoint: model.importEndpoint,
		ImportPath:     model.importPath,
	}
}

func requestTraceConfig(request model.Request, environment model.Environment) model.TraceConfig {
	if request.Trace.TransactionID.Header != "" || len(request.Trace.TransactionID.JSONPaths) > 0 || request.Trace.TransactionID.Regex != "" {
		return request.Trace
	}
	return environment.Trace
}

func defaultImportOutput(path string) string {
	if strings.TrimSpace(path) == "" {
		return "requests/replay.request.yaml"
	}
	return path
}
