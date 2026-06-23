package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	"github.com/billhuanglofi/flowman/internal/storage"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
	"github.com/spf13/cobra"
)

type requestExecutor interface {
	Run(context.Context, runner.Execution) (runner.Response, error)
}

var newRequestExecutor = func(timeout time.Duration) requestExecutor {
	return runner.New(runner.Options{Timeout: timeout})
}

type requestRunFlags struct {
	envName    string
	txID       string
	configPath string
}

type requestRunReport struct {
	RequestName    string              `json:"request_name"`
	Environment    string              `json:"environment"`
	TransactionID  string              `json:"transaction_id"`
	StatusCode     int                 `json:"status_code"`
	Duration       string              `json:"duration"`
	BodyBytes      int                 `json:"body_bytes"`
	DisplayHeaders map[string][]string `json:"display_headers"`
}

func newRequestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request",
		Short: "Run configured HTTP requests",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newRequestRunCommand())
	return cmd
}

func newRequestRunCommand() *cobra.Command {
	flags := requestRunFlags{}
	cmd := &cobra.Command{
		Use:   "run <request.yaml>",
		Short: "Run a canonical request and print a redacted JSON report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requestPath := args[0]
			request, environment, err := loadCommandRequestInputs(flags.configPath, requestPath, flags.envName)
			if err != nil {
				return err
			}
			executor := newRequestExecutor(requestTimeout(request, environment))
			response, err := executor.Run(cmd.Context(), runner.Execution{Request: request, Environment: environment})
			if err != nil {
				return err
			}
			transactionID, err := flowtrace.ExtractTransactionID(flowtrace.ExtractionInput{Override: flags.txID, Config: requestTraceConfig(request, environment), Response: response})
			if err != nil {
				return err
			}
			report := requestRunReport{
				RequestName:    request.Name,
				Environment:    environment.Name,
				TransactionID:  transactionID,
				StatusCode:     response.StatusCode,
				Duration:       response.Duration.String(),
				BodyBytes:      len(response.Body),
				DisplayHeaders: response.DisplayHeaders,
			}
			return writeJSONReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to use")
	cmd.Flags().StringVar(&flags.txID, "tx", "", "transaction id override")
	cmd.Flags().StringVar(&flags.configPath, "config", "", "path to flowman.yaml (defaults to nearest workspace above the request file)")
	_ = cmd.MarkFlagRequired("env")
	return cmd
}

func loadCommandRequestInputs(configPath string, requestPath string, envName string) (model.Request, model.Environment, error) {
	workspacePath := configPath
	if workspacePath == "" {
		resolved, err := findWorkspaceConfig(requestPath)
		if err != nil {
			return model.Request{}, model.Environment{}, err
		}
		workspacePath = resolved
	}
	workspace, err := config.LoadWorkspaceFromConfig(workspacePath)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	request, err := loadRequestFromPath(requestPath)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	environment, err := selectEnvironment(workspace, envName)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	return request, environment, nil
}

func loadWorkspaceForCommand(configPath string) (model.Workspace, error) {
	return config.LoadWorkspaceFromConfig(configPath)
}

func loadRequestFromPath(requestPath string) (model.Request, error) {
	return storage.LoadRequest(requestPath)
}

func requestTraceConfig(request model.Request, environment model.Environment) model.TraceConfig {
	if len(request.Trace.TransactionID.JSONPaths) > 0 || request.Trace.TransactionID.Header != "" || request.Trace.TransactionID.Regex != "" {
		return request.Trace
	}
	return environment.Trace
}

func requestTimeout(request model.Request, environment model.Environment) time.Duration {
	traceConfig := requestTraceConfig(request, environment)
	if timeout := traceConfig.Timeout.Duration(); timeout > 0 {
		return timeout
	}
	return 30 * time.Second
}

func writeJSONReport(output interface{ Write([]byte) (int, error) }, report any) error {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json report: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := output.Write(encoded); err != nil {
		return fmt.Errorf("write json report: %w", err)
	}
	return nil
}
