package cli

import (
	"path/filepath"

	insomniaexport "github.com/billhuanglofi/flowman/internal/exporter/insomnia"
	postmanexport "github.com/billhuanglofi/flowman/internal/exporter/postman"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/spf13/cobra"
)

var exportPostmanWorkspace = postmanexport.ExportWorkspace
var exportInsomniaWorkspace = insomniaexport.ExportWorkspace

type exportFlags struct {
	outputPath string
	reportPath string
	strict     bool
}

type exportReport struct {
	WorkspacePath string                `json:"workspace_path"`
	OutputPath    string                `json:"output_path"`
	ExportedCount int                   `json:"exported_count"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath    string                `json:"report_path,omitempty"`
	Strict        bool                  `json:"strict"`
}

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "export", Short: "Export Flowman data to supported formats", Args: cobra.NoArgs}
	cmd.AddCommand(newExportPostmanCommand())
	cmd.AddCommand(newExportInsomniaCommand())
	return cmd
}

func newExportPostmanCommand() *cobra.Command {
	flags := exportFlags{}
	cmd := &cobra.Command{
		Use:   "postman <flowman-root-or-dir>",
		Short: "Export canonical Flowman YAML to a Postman collection subset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspacePath := args[0]
			result, err := exportPostmanWorkspace(cmd.Context(), postmanexport.ExportOptions{WorkspacePath: workspacePath, OutputPath: flags.outputPath, ReportPath: flags.reportPath, Strict: flags.strict})
			report := exportReport{WorkspacePath: workspacePath, OutputPath: result.OutputPath, ExportedCount: result.ExportedCount, Warnings: result.Warnings, ReportPath: result.ReportPath, Strict: flags.strict}
			if err != nil {
				if writeErr := writeJSONReport(cmd.OutOrStdout(), report); writeErr != nil {
					return errorsJoin(err, writeErr)
				}
				return err
			}
			return writeJSONReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&flags.outputPath, "out", filepath.Join(".flowman", "generated", "collection.postman_collection.json"), "path to write Postman collection JSON")
	cmd.Flags().StringVar(&flags.reportPath, "report", "", "optional path to write JSON warning report")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "exit non-zero when export warnings are produced")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}

func newExportInsomniaCommand() *cobra.Command {
	flags := exportFlags{}
	cmd := &cobra.Command{
		Use:   "insomnia <flowman-root-or-dir>",
		Short: "Export canonical Flowman YAML to an Insomnia subset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspacePath := args[0]
			result, err := exportInsomniaWorkspace(cmd.Context(), insomniaexport.ExportOptions{WorkspacePath: workspacePath, OutputPath: flags.outputPath, ReportPath: flags.reportPath, Strict: flags.strict})
			report := exportReport{WorkspacePath: workspacePath, OutputPath: result.OutputPath, ExportedCount: result.ExportedCount, Warnings: result.Warnings, ReportPath: result.ReportPath, Strict: flags.strict}
			if err != nil {
				if writeErr := writeJSONReport(cmd.OutOrStdout(), report); writeErr != nil {
					return errorsJoin(err, writeErr)
				}
				return err
			}
			return writeJSONReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&flags.outputPath, "out", filepath.Join(".flowman", "generated", "collection.insomnia.json"), "path to write Insomnia JSON or YAML")
	cmd.Flags().StringVar(&flags.reportPath, "report", "", "optional path to write JSON warning report")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "exit non-zero when export warnings are produced")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
