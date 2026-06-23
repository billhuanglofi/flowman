package config

import (
	"fmt"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func LoadWorkspaceFromConfig(path string) (model.Workspace, error) {
	project, err := storage.LoadProject(path)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("load config %s: %w", path, err)
	}
	root := filepath.Dir(path)
	environments, err := loadEnvironments(root, project.Environments)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("load config %s: %w", path, err)
	}
	requests, err := loadRequests(root, project.Requests)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("load config %s: %w", path, err)
	}
	flows, err := loadFlows(root, project.Flows)
	if err != nil {
		return model.Workspace{}, fmt.Errorf("load config %s: %w", path, err)
	}
	return model.Workspace{Project: project, Environments: environments, Requests: requests, Flows: flows}, nil
}

func loadEnvironments(root string, paths []string) ([]model.Environment, error) {
	environments := make([]model.Environment, 0, len(paths))
	for _, relativePath := range paths {
		environment, err := storage.LoadEnvironment(filepath.Join(root, relativePath))
		if err != nil {
			return nil, err
		}
		environments = append(environments, environment)
	}
	return environments, nil
}

func loadRequests(root string, paths []string) ([]model.Request, error) {
	requests := make([]model.Request, 0, len(paths))
	for _, relativePath := range paths {
		request, err := storage.LoadRequest(filepath.Join(root, relativePath))
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func loadFlows(root string, paths []string) ([]model.Flow, error) {
	flows := make([]model.Flow, 0, len(paths))
	for _, relativePath := range paths {
		flow, err := storage.LoadFlow(filepath.Join(root, relativePath))
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}
	return flows, nil
}
