package tui

import (
	"context"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/oracle"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

type RunOptions struct {
	ConfigPath     string
	EnvName        string
	Output         io.Writer
	Program        bool
	Services       WorkflowServices
	ImportURL      string
	ImportEndpoint string
	ImportPath     string
	Close          io.Closer
}

func Run(ctx context.Context, options RunOptions) error {
	preview := LoadPreview(options.ConfigPath, options.EnvName)
	if !options.Program {
		_, err := fmt.Fprintln(options.Output, RenderPreview(preview, RenderSize{Width: 100, Height: 32}))
		return err
	}
	services := options.Services
	if services == nil {
		services = defaultServices(ctx, preview)
	}
	program := tea.NewProgram(NewWorkflowModel(preview.Workspace, WorkflowOptions{EnvName: preview.SelectedEnv, Services: services, ImportURL: options.ImportURL, ImportEndpoint: options.ImportEndpoint, ImportPath: options.ImportPath}), tea.WithContext(ctx))
	_, err := program.Run()
	if options.Close != nil {
		if closeErr := options.Close.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}
	return err
}

func NewServices(requestRunner interface {
	Run(context.Context, runner.Execution) (runner.Response, error)
}, traceStore flowtrace.ProcessStateStore, importer func(context.Context, safestore.ImportOptions) (safestore.ImportResult, error)) WorkflowServices {
	return RealWorkflowServices{Runner: requestRunner, Trace: traceStore, Importer: importer}
}

func defaultServices(ctx context.Context, preview Preview) WorkflowServices {
	services, _ := defaultServicesWithClose(ctx, preview)
	return services
}

func defaultServicesWithClose(ctx context.Context, preview Preview) (WorkflowServices, io.Closer) {
	environment, _ := selectedEnvironment(preview.Workspace, preview.SelectedEnv)
	requestRunner := runner.New(runner.Options{Timeout: workflowTimeout(environment)})
	traceStore, closeTrace, err := oracle.OpenProcessStateStoreFromEnv(ctx, oracleEnvConfig(environment))
	if err != nil {
		traceStore = errorTraceStore{err: err}
		closeTrace = nil
	}
	return RealWorkflowServices{Runner: requestRunner, Trace: traceStore, Importer: importSafestoreWithOracle}, closeFunc(closeTrace)
}

func importSafestoreWithOracle(ctx context.Context, options safestore.ImportOptions) (safestore.ImportResult, error) {
	store, closeStore, err := oracle.OpenSafestoreStoreFromEnv(ctx, oracleEnvConfig(options.Environment))
	if err != nil {
		return safestore.ImportResult{}, err
	}
	if closeStore != nil {
		defer closeStore()
	}
	return safestore.ImportReplayRequest(ctx, store, options)
}

func oracleEnvConfig(environment model.Environment) oracle.EnvConfig {
	return oracle.EnvConfig{DSNEnv: environment.Oracle.DSNEnv, UserEnv: environment.Oracle.UserEnv, PasswordEnv: environment.Oracle.PasswordEnv}
}

type errorTraceStore struct {
	err error
}

func (store errorTraceStore) Journey(context.Context, flowtrace.ProcessStateRequest) (flowtrace.Journey, error) {
	return flowtrace.Journey{}, store.err
}

type closeFunc func() error

func (fn closeFunc) Close() error {
	if fn == nil {
		return nil
	}
	return fn()
}

func LoadPreview(configPath string, envName string) Preview {
	workspace, err := config.LoadWorkspaceFromConfig(configPath)
	if err != nil {
		return PreviewError(envName, err)
	}
	selectedEnv := envName
	if selectedEnv == "" {
		selectedEnv = workspace.Project.DefaultEnv
	}
	if _, ok := selectedEnvironment(workspace, selectedEnv); !ok {
		return PreviewError(selectedEnv, fmt.Errorf("environment %q not found in config %s", selectedEnv, configPath))
	}
	return PreviewFromWorkspace(workspace, PreviewOptions{EnvName: selectedEnv, Journey: sampleJourney(workspace)})
}

func sampleJourney(workspace model.Workspace) flowtrace.Journey {
	transactionID := "preview-transaction"
	requestName := "selected request"
	if request, ok := selectedRequest(workspace, 0); ok {
		requestName = request.Name
	}
	rows := []flowtrace.TraceRow{
		{TransactionID: transactionID, Step: 0, Service: "runner", State: "READY", Outcome: "PENDING", Message: "prepared " + requestName},
		{TransactionID: transactionID, Step: 1, Service: "oracle", State: "WAITING", Outcome: "PENDING", Message: "trace placeholder"},
	}
	return flowtrace.MapProcessStateJourney(transactionID, rows, flowtrace.DefaultTerminalStateClassification())
}
