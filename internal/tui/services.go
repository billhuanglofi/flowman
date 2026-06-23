package tui

import (
	"context"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

type RealWorkflowServices struct {
	Runner interface {
		Run(context.Context, runner.Execution) (runner.Response, error)
	}
	Trace    flowtrace.ProcessStateStore
	Importer func(context.Context, safestore.ImportOptions) (safestore.ImportResult, error)
}

func (services RealWorkflowServices) RunRequest(ctx context.Context, execution runner.Execution) (runner.Response, error) {
	return services.Runner.Run(ctx, execution)
}

func (services RealWorkflowServices) TraceJourney(ctx context.Context, request flowtrace.ProcessStateRequest) (flowtrace.Journey, error) {
	return services.Trace.Journey(ctx, request)
}

func (services RealWorkflowServices) ImportSafestore(ctx context.Context, options safestore.ImportOptions) (safestore.ImportResult, error) {
	return services.Importer(ctx, options)
}

func workflowTimeout(environment model.Environment) time.Duration {
	if timeout := environment.Trace.Timeout.Duration(); timeout > 0 {
		return timeout
	}
	return 30 * time.Second
}
