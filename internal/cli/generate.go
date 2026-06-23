package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/billhuanglofi/flowman/internal/generate"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
	"github.com/spf13/cobra"
)

type generateFlags struct {
	envName                     string
	outputPath                  string
	unsafeAllowSensitiveHeaders bool
}

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate derived request artifacts",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newGenerateProjectionCommand("curl"))
	cmd.AddCommand(newGenerateProjectionCommand("http"))
	return cmd
}

func newGenerateProjectionCommand(kind string) *cobra.Command {
	flags := generateFlags{}
	cmd := &cobra.Command{
		Use:   kind + " <request.yaml>",
		Short: projectionShort(kind),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requestPath := args[0]
			request, environment, err := loadProjectionInputs(requestPath, flags.envName)
			if err != nil {
				return err
			}
			content, err := renderProjection(kind, request, environment, flags.unsafeAllowSensitiveHeaders)
			if err != nil {
				return err
			}
			if err := writeProjection(flags.outputPath, content); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "generated %s projection at %s\n", kind, flags.outputPath)
			return err
		},
	}
	cmd.Flags().StringVar(&flags.envName, "env", "", "environment name to use for URL resolution")
	cmd.Flags().StringVar(&flags.outputPath, "out", defaultProjectionPath(kind), "path to write the generated projection")
	cmd.Flags().BoolVar(&flags.unsafeAllowSensitiveHeaders, "unsafe-allow-sensitive-headers", false, "allow raw sensitive header values to be written to projections")
	return cmd
}

func projectionShort(kind string) string {
	if kind == "curl" {
		return "Generate deterministic cURL projection from canonical request YAML"
	}
	return "Generate deterministic .http projection from canonical request YAML"
}

func defaultProjectionPath(kind string) string {
	if kind == "curl" {
		return filepath.Join(".flowman", "generated", "request.sh")
	}
	return filepath.Join(".flowman", "generated", "request.http")
}

func loadProjectionInputs(requestPath string, envName string) (model.Request, model.Environment, error) {
	workspacePath, err := findWorkspaceConfig(requestPath)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	workspace, err := config.LoadWorkspaceFromConfig(workspacePath)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	request, err := storage.LoadRequest(requestPath)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	environment, err := selectEnvironment(workspace, envName)
	if err != nil {
		return model.Request{}, model.Environment{}, err
	}
	return request, environment, nil
}

func findWorkspaceConfig(requestPath string) (string, error) {
	absPath, err := filepath.Abs(requestPath)
	if err != nil {
		return "", fmt.Errorf("resolve request path %s: %w", requestPath, err)
	}
	current := filepath.Dir(absPath)
	for {
		candidate := filepath.Join(current, "flowman.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return "", fmt.Errorf("cannot find flowman.yaml above %s", requestPath)
}

func selectEnvironment(workspace model.Workspace, envName string) (model.Environment, error) {
	selectedName := strings.TrimSpace(envName)
	if selectedName == "" {
		selectedName = workspace.Project.DefaultEnv
	}
	for _, environment := range workspace.Environments {
		if environment.Name == selectedName {
			return environment, nil
		}
	}
	return model.Environment{}, fmt.Errorf("environment %q: %w", selectedName, config.ErrEnvironmentNotFound)
}

func renderProjection(kind string, request model.Request, environment model.Environment, allowUnsafe bool) ([]byte, error) {
	options := generate.Options{Environment: environment, UnsafeAllowSensitiveHeaders: allowUnsafe}
	if kind == "curl" {
		return generate.RenderCurl(request, options)
	}
	return generate.RenderHTTP(request, options)
}

func writeProjection(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create projection output directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write projection %s: %w", path, err)
	}
	return nil
}
