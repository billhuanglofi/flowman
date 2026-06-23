package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/oracle"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
	"github.com/spf13/cobra"
)

var openTraceStore = oracle.OpenProcessStateStoreFromEnv

type traceFlags struct {
	transactionID string
	envName       string
	configPath    string
}

func newTraceCommand() *cobra.Command {
	flags := traceFlags{}
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Trace request journeys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := loadWorkspaceForCommand(flags.configPath)
			if err != nil {
				return err
			}
			environment, err := selectEnvironment(workspace, flags.envName)
			if err != nil {
				return err
			}
			store, closeStore, err := openTraceStore(cmd.Context(), oracleEnvConfig(environment))
			if err != nil {
				return err
			}
			if closeStore != nil {
				defer closeStore()
			}
			journey, err := store.Journey(cmd.Context(), flowtrace.ProcessStateRequest{TransactionID: flags.transactionID, Timeout: environment.Trace.Timeout.Duration(), Classification: flowtrace.DefaultTerminalStateClassification()})
			if err != nil {
				return err
			}
			return writeTraceReport(cmd.OutOrStdout(), journey)
		},
	}
	cmd.Flags().StringVar(&flags.transactionID, "tx", "", "transaction id to trace")
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to use")
	cmd.Flags().StringVar(&flags.configPath, "config", "flowman.yaml", "path to flowman.yaml")
	_ = cmd.MarkFlagRequired("tx")
	_ = cmd.MarkFlagRequired("env")
	return cmd
}

func oracleEnvConfig(environment model.Environment) oracle.EnvConfig {
	return oracle.EnvConfig{
		DSNEnv:      environment.Oracle.DSNEnv,
		UserEnv:     environment.Oracle.UserEnv,
		PasswordEnv: environment.Oracle.PasswordEnv,
	}
}

func writeTraceReport(output io.Writer, journey flowtrace.Journey) error {
	if _, err := fmt.Fprintf(output, "Transaction: %s\n", journey.TransactionID); err != nil {
		return fmt.Errorf("write trace header: %w", err)
	}
	for _, warning := range journey.Warnings {
		if _, err := fmt.Fprintf(output, "Warning [%s]: %s\n", warning.Code, warning.Message); err != nil {
			return fmt.Errorf("write trace warning: %w", err)
		}
	}
	for _, row := range journey.Rows {
		message := strings.ReplaceAll(row.Message, "\n", " ")
		if _, err := fmt.Fprintf(output, "%s step=%d service=%s state=%s outcome=%s label=%s message=%q\n", row.Timestamp.Format(time.RFC3339), row.Step, row.Service, row.State, row.Outcome, row.DisplayLabel, message); err != nil {
			return fmt.Errorf("write trace row: %w", err)
		}
	}
	return nil
}
