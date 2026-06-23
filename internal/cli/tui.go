package cli

import (
	"github.com/billhuanglofi/flowman/internal/tui"
	"github.com/spf13/cobra"
)

type tuiFlags struct {
	configPath     string
	envName        string
	preview        bool
	importURL      string
	importEndpoint string
	importPath     string
}

func newTUICommand() *cobra.Command {
	flags := tuiFlags{}
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the Flowman preview TUI",
		Long:  "Launch the Flowman preview TUI using canonical config, requests, runner response, extraction, and trace journey models.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(cmd.Context(), tui.RunOptions{ConfigPath: flags.configPath, EnvName: flags.envName, Output: cmd.OutOrStdout(), Program: !flags.preview, ImportURL: flags.importURL, ImportEndpoint: flags.importEndpoint, ImportPath: flags.importPath})
		},
	}
	cmd.Flags().StringVar(&flags.configPath, "config", "flowman.yaml", "path to flowman.yaml")
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to preview")
	cmd.Flags().StringVar(&flags.importURL, "import-url", "", "Safestore import full request URL resolution")
	cmd.Flags().StringVar(&flags.importEndpoint, "import-endpoint", "", "Safestore import endpoint alias resolution")
	cmd.Flags().StringVar(&flags.importPath, "import-path", "", "Safestore import path resolution")
	cmd.Flags().BoolVar(&flags.preview, "preview", false, "render deterministic preview text and exit")
	return cmd
}
