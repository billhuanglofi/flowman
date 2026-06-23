package cli

import (
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/importer/insomnia"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/spf13/cobra"
)

var importInsomniaWorkspace = insomnia.ImportWorkspace

type importInsomniaFlags struct {
	outputDir  string
	reportPath string
	strict     bool
}

type insomniaImportReport struct {
	DocumentPath  string                `json:"document_path"`
	OutputDir     string                `json:"output_dir"`
	ImportedCount int                   `json:"imported_count"`
	RequestPaths  []string              `json:"request_paths,omitempty"`
	FlowPaths     []string              `json:"flow_paths,omitempty"`
	Warnings      []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath    string                `json:"report_path,omitempty"`
	Strict        bool                  `json:"strict"`
}

func newImportInsomniaCommand() *cobra.Command {
	flags := importInsomniaFlags{}
	cmd := &cobra.Command{
		Use:   "insomnia <file>",
		Short: "Import a supported subset of an Insomnia export",
		Long:  "Import request groups, requests, URL/query/header/body fields, and env-backed sensitive headers from an Insomnia export into canonical Flowman YAML. Unsupported fields are reported with JSON pointer paths; --strict exits non-zero on warnings.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			documentPath := args[0]
			result, err := importInsomniaWorkspace(cmd.Context(), insomnia.ImportOptions{DocumentPath: documentPath, OutputDir: flags.outputDir, ReportPath: flags.reportPath, Strict: flags.strict})
			report := insomniaImportReport{DocumentPath: documentPath, OutputDir: flags.outputDir, ImportedCount: result.ImportedCount, RequestPaths: result.RequestPaths, FlowPaths: result.FlowPaths, Warnings: result.Warnings, ReportPath: result.ReportPath, Strict: flags.strict}
			if err != nil {
				if writeErr := writeJSONReport(cmd.OutOrStdout(), report); writeErr != nil {
					return errorsJoin(err, writeErr)
				}
				return err
			}
			return writeJSONReport(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&flags.outputDir, "out", ".", "directory to write canonical requests/ and flows/")
	cmd.Flags().StringVar(&flags.reportPath, "report", filepath.Join(".flowman", "generated", "insomnia-import-report.json"), "path to write JSON warning report")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "exit non-zero when unsupported Insomnia fields produce warnings")
	return cmd
}
