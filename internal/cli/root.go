package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/billhuanglofi/flowman/internal/app"
	"github.com/spf13/cobra"
)

type commandGroup struct {
	use   string
	short string
}

var placeholderGroups = []commandGroup{
	{use: "config", short: "Validate and inspect Flowman configuration"},
	{use: "request", short: "Run configured HTTP requests"},
	{use: "trace", short: "Trace request journeys"},
	{use: "import", short: "Import external API workflow data"},
	{use: "export", short: "Export Flowman data to supported formats"},
	{use: "generate", short: "Generate derived request artifacts"},
	{use: "tui", short: "Launch the terminal interface"},
}

func Execute(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := NewRootCommand(stdout, stderr)
	if _, _, err := cmd.Find(args); err != nil {
		return errors.Join(err, cmd.Help())
	}
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

func NewRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "flowman",
		Short:         "Flowman traces and replays API request flows",
		Long:          "Flowman traces and replays API request flows from Git-friendly project files.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newRequestCommand())
	rootCmd.AddCommand(newTraceCommand())
	rootCmd.AddCommand(newImportCommand())
	rootCmd.AddCommand(newExportCommand())
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(newTUICommand())
	for _, group := range placeholderGroups {
		if group.use == "config" || group.use == "request" || group.use == "trace" || group.use == "import" || group.use == "export" || group.use == "generate" || group.use == "tui" {
			continue
		}
		rootCmd.AddCommand(newPlaceholderCommand(group))
	}

	return rootCmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Flowman version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "flowman %s\n", app.Version())
			return err
		},
	}
}

func newPlaceholderCommand(group commandGroup) *cobra.Command {
	return &cobra.Command{
		Use:   group.use,
		Short: group.short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s command group is not implemented yet\n", group.use)
			return err
		},
	}
}
