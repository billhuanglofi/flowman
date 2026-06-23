package tui

import (
	"context"
	"fmt"
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
	activeTab      responseTab
	historyIndex   int
	history        []historyEntry
}

type responseTab string

const (
	tabBody    responseTab = "body"
	tabHeaders responseTab = "headers"
	tabTrace   responseTab = "trace"
)

type historyEntry struct {
	request       string
	statusCode    int
	duration      string
	transactionID string
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
	return Model{
		preview:      preview,
		size:         RenderSize{Width: 100, Height: 32},
		environment:  environment,
		status:       statusReady,
		inputMode:    inputModeNone,
		activeTab:    tabBody,
		history:      make([]historyEntry, 0, 20),
		historyIndex: -1,
	}
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
		message:        "r run | Ctrl+R replay | tab switch | j/k nav | e env | ? help | q quit",
		inputMode:      inputModeNone,
		importURL:      options.ImportURL,
		importEndpoint: options.ImportEndpoint,
		importPath:     options.ImportPath,
		importOutput:   defaultImportOutput(options.ImportOutput),
		activeTab:      tabBody,
		history:        make([]historyEntry, 0, 20),
		historyIndex:   -1,
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
	case 'j':
		key = "j"
	case 'k':
		key = "k"
	case 'e':
		key = "e"
	case '?':
		key = "?"
	}
	switch key {
	case "q", "ctrl+c":
		return model, tea.Quit
	case "up", "k":
		model.moveSelection(-1)
	case "down", "j":
		model.moveSelection(1)
	case "tab":
		model.nextTab()
	case "shift+tab":
		model.prevTab()
	case "r":
		return model.startRun()
	case "ctrl+r":
		return model.replayLast()
	case "e":
		return model.showEnvSwitcher()
	case "?":
		return model.showHelp()
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

func (model *Model) nextTab() {
	switch model.activeTab {
	case tabBody:
		model.activeTab = tabHeaders
	case tabHeaders:
		model.activeTab = tabTrace
	case tabTrace:
		model.activeTab = tabBody
	}
	model.message = fmt.Sprintf("Switched to %s tab", model.activeTab)
}

func (model *Model) prevTab() {
	switch model.activeTab {
	case tabBody:
		model.activeTab = tabTrace
	case tabHeaders:
		model.activeTab = tabBody
	case tabTrace:
		model.activeTab = tabHeaders
	}
	model.message = fmt.Sprintf("Switched to %s tab", model.activeTab)
}

func (model Model) replayLast() (tea.Model, tea.Cmd) {
	if len(model.history) == 0 {
		model.message = "No history to replay"
		return model, nil
	}
	model.message = "Replaying last request..."
	return model.startRun()
}

func (model Model) showEnvSwitcher() (tea.Model, tea.Cmd) {
	// For now, just cycle through environments
	envs := model.preview.Workspace.Environments
	if len(envs) == 0 {
		model.message = "No environments available"
		return model, nil
	}

	currentIdx := -1
	for i, env := range envs {
		if env.Name == model.preview.SelectedEnv {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(envs)
	model.preview.SelectedEnv = envs[nextIdx].Name
	model.environment = envs[nextIdx]
	model.message = fmt.Sprintf("Switched to environment: %s", envs[nextIdx].Name)
	return model, nil
}

func (model Model) showHelp() (tea.Model, tea.Cmd) {
	model.message = "Help: r=run | Ctrl+R=replay | Tab=switch tabs | j/k=nav | e=env | q=quit"
	return model, nil
}

func (model *Model) addToHistory(request string, statusCode int, duration string, transactionID string) {
	entry := historyEntry{
		request:       request,
		statusCode:    statusCode,
		duration:      duration,
		transactionID: transactionID,
	}

	// Keep last 20 entries
	model.history = append([]historyEntry{entry}, model.history...)
	if len(model.history) > 20 {
		model.history = model.history[:20]
	}
	model.historyIndex = 0
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
		ActiveTab:      string(model.activeTab),
		History:        model.history,
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
