package cli

import (
	"fmt"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/importer/postman"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/spf13/cobra"
)

var importPostmanCollection = postman.ImportCollection

type importPostmanFlags struct {
	outputDir  string
	reportPath string
	strict     bool
}

type postmanImportReport struct {
	CollectionPath string                `json:"collection_path"`
	OutputDir      string                `json:"output_dir"`
	ImportedCount  int                   `json:"imported_count"`
	RequestPaths   []string              `json:"request_paths,omitempty"`
	FlowPaths      []string              `json:"flow_paths,omitempty"`
	Warnings       []model.ImportWarning `json:"warnings,omitempty"`
	ReportPath     string                `json:"report_path,omitempty"`
	Strict         bool                  `json:"strict"`
}

func newImportPostmanCommand() *cobra.Command {
	flags := importPostmanFlags{}
	cmd := &cobra.Command{
		Use:   "postman <file>",
		Short: "Import a supported subset of a Postman collection",
		Long:  "Import folders, requests, URL/query/header/body fields, and basic/bearer/api-key auth from a Postman collection into canonical Flowman YAML. Unsupported fields are reported with JSON pointer paths; --strict exits non-zero on warnings.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			collectionPath := args[0]
			result, err := importPostmanCollection(cmd.Context(), postman.ImportOptions{
				CollectionPath: collectionPath,
				OutputDir:      flags.outputDir,
				ReportPath:     flags.reportPath,
				Strict:         flags.strict,
			})
			report := postmanImportReport{
				CollectionPath: collectionPath,
				OutputDir:      flags.outputDir,
				ImportedCount:  result.ImportedCount,
				RequestPaths:   result.RequestPaths,
				FlowPaths:      result.FlowPaths,
				Warnings:       result.Warnings,
				ReportPath:     result.ReportPath,
				Strict:         flags.strict,
			}
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
	cmd.Flags().StringVar(&flags.reportPath, "report", filepath.Join(".flowman", "generated", "postman-import-report.json"), "path to write JSON warning report")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "exit non-zero when unsupported Postman fields produce warnings")
	return cmd
}

func errorsJoin(left error, right error) error {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	return fmt.Errorf("%w; %v", left, right)
}
