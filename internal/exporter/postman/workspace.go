package postman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/billhuanglofi/flowman/internal/model"
)

func LoadWorkspaceForReuse(path string) (model.Workspace, error) {
	return loadWorkspace(path)
}

func loadWorkspace(path string) (model.Workspace, error) {
	configPath, err := resolveWorkspaceConfig(path)
	if err != nil {
		return model.Workspace{}, err
	}
	workspace, err := config.LoadWorkspaceFromConfig(configPath)
	if err != nil {
		return model.Workspace{}, err
	}
	return workspace, nil
}

func resolveWorkspaceConfig(path string) (string, error) {
	if filepath.Base(path) == "flowman.yaml" {
		return path, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat workspace path %s: %w", path, err)
	}
	current := path
	if !info.IsDir() {
		current = filepath.Dir(path)
	}
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
	return "", fmt.Errorf("cannot find flowman.yaml above %s", path)
}
