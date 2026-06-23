package tui

import (
	"context"
	"errors"
	"regexp"

	tea "charm.land/bubbletea/v2"
	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
	uv "github.com/charmbracelet/ultraviolet"
)

func press(text string) tea.KeyPressMsg {
	char := []rune(text)[0]
	return tea.KeyPressMsg(uv.KeyPressEvent(uv.Key{Text: text, Code: char}))
}

func pressSpecial(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(uv.KeyPressEvent(uv.Key{Code: code}))
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

func tuiTestWorkspace() model.Workspace {
	return model.Workspace{
		Project: model.Project{Name: "flowman-example", DefaultEnv: "uat"},
		Environments: []model.Environment{{
			Name:      "uat",
			BaseURL:   "https://uat.api.example.test",
			Endpoints: []model.EndpointAlias{{Name: "payments", Path: "/payments"}},
			Trace:     model.TraceConfig{TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"}},
		}, {Name: "ppd", BaseURL: "https://ppd.api.example.test"}},
		Requests: []model.Request{{
			Name: "create payment", Method: "POST", Endpoint: "payments", Path: "/v1/payments",
			Trace: model.TraceConfig{TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"}},
		}},
		Flows: []model.Flow{
			{Name: "create payment flow", Steps: []model.FlowStep{{Name: "create payment", Request: "requests/payment/create.request.yaml", Trace: true}}},
			{Name: "create refund flow", Steps: []model.FlowStep{{Name: "create refund", Request: "requests/payment/refund.request.yaml", Trace: true}}},
		},
	}
}

type fakeRequestRunner struct{}

func (fakeRequestRunner) Run(context.Context, runner.Execution) (runner.Response, error) {
	return runner.Response{}, nil
}

type closeTrackingTraceStore struct {
	journey flowtrace.Journey
	closed  bool
}

func (store *closeTrackingTraceStore) Journey(context.Context, flowtrace.ProcessStateRequest) (flowtrace.Journey, error) {
	if store.closed {
		return flowtrace.Journey{}, errors.New("trace store closed before use")
	}
	return store.journey, nil
}

func (store *closeTrackingTraceStore) Close() error {
	store.closed = true
	return nil
}

type fakeWorkflowServices struct {
	response           runner.Response
	runErr             error
	journey            flowtrace.Journey
	traceErr           error
	importResult       safestore.ImportResult
	importErr          error
	runCalls           int
	traceCalls         int
	importCalls        int
	traceTransactionID string
	importURL          string
	importEndpoint     string
	importPath         string
}

func (services *fakeWorkflowServices) RunRequest(_ context.Context, execution runner.Execution) (runner.Response, error) {
	services.runCalls++
	if execution.Request.Name == "" || execution.Environment.Name == "" {
		return runner.Response{}, errors.New("fake runner received incomplete execution")
	}
	return services.response, services.runErr
}

func (services *fakeWorkflowServices) TraceJourney(_ context.Context, request flowtrace.ProcessStateRequest) (flowtrace.Journey, error) {
	services.traceCalls++
	services.traceTransactionID = request.TransactionID
	if services.traceErr != nil {
		return flowtrace.Journey{}, services.traceErr
	}
	return services.journey, nil
}

func (services *fakeWorkflowServices) ImportSafestore(_ context.Context, options safestore.ImportOptions) (safestore.ImportResult, error) {
	services.importCalls++
	services.importURL = options.URL
	services.importEndpoint = options.Endpoint
	services.importPath = options.Path
	if services.importErr != nil {
		return safestore.ImportResult{}, services.importErr
	}
	if options.URL == "" && options.Endpoint == "" && options.Path == "" {
		return safestore.ImportResult{}, safestore.ErrCannotResolveURLNonInteractive
	}
	return services.importResult, nil
}
