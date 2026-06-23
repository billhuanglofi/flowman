package cli

import (
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/oracle"
	"github.com/spf13/cobra"
)

var openSafestoreStore = oracle.OpenSafestoreStoreFromEnv
var importSafestoreReplayRequest = safestore.ImportReplayRequest

type importSafestoreFlags struct {
	transactionID  string
	envName        string
	configPath     string
	url            string
	endpoint       string
	path           string
	method         string
	outputPath     string
	nonInteractive bool
	selection      string
	all            bool
	overwrite      bool
}

type safestoreImportReport struct {
	TransactionID string                `json:"transaction_id"`
	Environment   string                `json:"environment"`
	Method        string                `json:"method"`
	OutputPath    string                `json:"output_path"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
}

func newImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import external API workflow data",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newImportPostmanCommand())
	cmd.AddCommand(newImportInsomniaCommand())
	cmd.AddCommand(newImportSafestoreCommand())
	return cmd
}

func newImportSafestoreCommand() *cobra.Command {
	flags := importSafestoreFlags{}
	cmd := &cobra.Command{
		Use:   "safestore",
		Short: "Import a Safestore request as canonical Flowman YAML",
		Long:  "Import a Safestore request by transaction id, resolve its URL from --url or environment endpoint/path inputs, default the method to POST, and emit a redacted JSON report.",
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
			store, closeStore, err := openSafestoreStore(cmd.Context(), oracleEnvConfig(environment))
			if err != nil {
				return err
			}
			if closeStore != nil {
				defer closeStore()
			}
			result, err := importSafestoreReplayRequest(cmd.Context(), store, safestore.ImportOptions{
				TransactionID:  flags.transactionID,
				Environment:    environment,
				URL:            flags.url,
				Endpoint:       flags.endpoint,
				Path:           flags.path,
				Method:         flags.method,
				OutputPath:     flags.outputPath,
				Selection:      safestore.Selection(flags.selection),
				All:            flags.all,
				Overwrite:      flags.overwrite,
				NonInteractive: flags.nonInteractive,
			})
			if err != nil {
				return err
			}
			report := safestoreImportReport{
				TransactionID: flags.transactionID,
				Environment:   environment.Name,
				Method:        result.Request.Method,
				OutputPath:    result.OutputPath,
				Warnings:      result.Request.ImportWarnings,
			}
			return writeJSONReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&flags.transactionID, "tx", "", "transaction id to import")
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to use")
	cmd.Flags().StringVar(&flags.configPath, "config", "flowman.yaml", "path to flowman.yaml")
	cmd.Flags().StringVar(&flags.url, "url", "", "full request URL override")
	cmd.Flags().StringVar(&flags.endpoint, "endpoint", "", "environment endpoint alias for URL resolution")
	cmd.Flags().StringVar(&flags.path, "path", "", "request path to join with the environment base URL or endpoint alias")
	cmd.Flags().StringVar(&flags.method, "method", "POST", "request method override; defaults to POST for Safestore imports")
	cmd.Flags().StringVar(&flags.outputPath, "out", filepath.Join("requests", "replay.request.yaml"), "path to write canonical request YAML")
	cmd.Flags().BoolVar(&flags.nonInteractive, "non-interactive", false, "fail fast instead of prompting when URL resolution input is incomplete")
	cmd.Flags().StringVar(&flags.selection, "select", "", "select one Safestore row with first, latest, or index:N")
	cmd.Flags().BoolVar(&flags.all, "all", false, "import all Safestore request rows for the transaction")
	cmd.Flags().BoolVar(&flags.overwrite, "overwrite", false, "overwrite existing output file")
	_ = cmd.MarkFlagRequired("tx")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
