package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

var ErrEnvironmentNotFound = errors.New("config: environment not found")

var ErrRawSensitiveHeader = errors.New("config: raw sensitive header value")

var ErrMissingSecretEnv = errors.New("config: missing secret environment variable")

var ErrInvalidConfig = errors.New("config: invalid")

type ValidationOptions struct {
	EnvName      string
	CheckSecrets bool
}

type ValidationResult struct {
	Environment    model.Environment
	SecretEnvNames []string
	Messages       []string
}

func ValidateWorkspace(workspace model.Workspace, options ValidationOptions) (ValidationResult, error) {
	environment, err := selectEnvironment(workspace, options.EnvName)
	if err != nil {
		return ValidationResult{}, err
	}
	secretNames := secretEnvNames(environment, workspace.Requests)
	result := ValidationResult{
		Environment:    environment,
		SecretEnvNames: secretNames,
		Messages: []string{
			"OK: config valid",
			"OK: secrets referenced by environment variable name only",
		},
	}
	if err := validateEnvironmentShape(environment); err != nil {
		return ValidationResult{}, err
	}
	if err := validateRequestReferences(environment, workspace.Requests); err != nil {
		return ValidationResult{}, err
	}
	if err := validateSecretReferences(workspace.Requests); err != nil {
		return ValidationResult{}, err
	}
	if options.CheckSecrets {
		if err := validateSecretPresence(secretNames); err != nil {
			return ValidationResult{}, err
		}
		result.Messages = append(result.Messages, "OK: checked secret env var presence")
	}
	return result, nil
}

func selectEnvironment(workspace model.Workspace, name string) (model.Environment, error) {
	selectedName := strings.TrimSpace(name)
	if selectedName == "" {
		selectedName = workspace.Project.DefaultEnv
	}
	for _, environment := range workspace.Environments {
		if environment.Name == selectedName {
			return environment, nil
		}
	}
	return model.Environment{}, fmt.Errorf("environment %q: %w", selectedName, ErrEnvironmentNotFound)
}

func validateEnvironmentShape(environment model.Environment) error {
	baseURL, err := url.Parse(environment.BaseURL)
	if err != nil {
		return fmt.Errorf("environment %q base_url: %w", environment.Name, err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return fmt.Errorf("environment %q base_url %q must include scheme and host: %w", environment.Name, environment.BaseURL, ErrInvalidConfig)
	}
	return nil
}

func validateRequestReferences(environment model.Environment, requests []model.Request) error {
	aliases := endpointAliases(environment)
	var errs []error
	for _, request := range requests {
		endpoint := strings.TrimSpace(request.Endpoint)
		if endpoint == "" {
			continue
		}
		if _, ok := aliases[endpoint]; !ok {
			errs = append(errs, fmt.Errorf("request %q endpoint %q is not defined in env %q: %w", request.Name, endpoint, environment.Name, ErrInvalidConfig))
		}
	}
	return errors.Join(errs...)
}

func validateSecretReferences(requests []model.Request) error {
	var errs []error
	for _, request := range requests {
		for index, header := range request.Headers {
			if !secrets.IsSensitiveHeader(header.Name) {
				continue
			}
			if strings.TrimSpace(header.Value) != "" {
				errs = append(errs, fmt.Errorf("request %q headers[%d] %s=%s must use env reference: %w", request.Name, index, header.Name, secrets.RedactHeaderValue(header.Name, header.Value), ErrRawSensitiveHeader))
				continue
			}
			if strings.TrimSpace(header.Env) == "" {
				errs = append(errs, fmt.Errorf("request %q headers[%d] %s missing env reference: %w", request.Name, index, header.Name, ErrRawSensitiveHeader))
			}
		}
	}
	return errors.Join(errs...)
}

func validateSecretPresence(names []string) error {
	var errs []error
	for _, name := range names {
		if _, ok := os.LookupEnv(name); ok {
			continue
		}
		errs = append(errs, fmt.Errorf("%s: %w", name, ErrMissingSecretEnv))
	}
	return errors.Join(errs...)
}

func endpointAliases(environment model.Environment) map[string]string {
	aliases := make(map[string]string, len(environment.Endpoints))
	for _, endpoint := range environment.Endpoints {
		aliases[endpoint.Name] = endpoint.Path
	}
	return aliases
}
