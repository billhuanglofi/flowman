package cli

import (
	"fmt"

	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/spf13/cobra"
)

type configValidateFlags struct {
	configPath   string
	envName      string
	checkSecrets bool
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Validate and inspect Flowman configuration",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newConfigValidateCommand())
	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	flags := configValidateFlags{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Flowman project configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := config.LoadWorkspaceFromConfig(flags.configPath)
			if err != nil {
				return err
			}
			result, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: flags.envName, CheckSecrets: flags.checkSecrets})
			if err != nil {
				return err
			}
			for _, message := range result.Messages {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), message); err != nil {
					return fmt.Errorf("write config validation output: %w", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flags.configPath, "config", "flowman.yaml", "path to flowman.yaml")
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to validate")
	cmd.Flags().BoolVar(&flags.checkSecrets, "check-secrets", false, "check required secret environment variables are present without printing values")
	return cmd
}
