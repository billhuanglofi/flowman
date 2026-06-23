package generate

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func resolveRequestURL(request model.Request, environment model.Environment) (string, error) {
	if explicitURL := strings.TrimSpace(request.URL); explicitURL != "" {
		return appendQuery(explicitURL, request.Query)
	}
	baseURL := strings.TrimSpace(environment.BaseURL)
	if baseURL == "" {
		return "", fmt.Errorf("request %q requires url or env base_url: %w", request.Name, ErrCannotResolveRequestURL)
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("request %q base_url %q: %w", request.Name, baseURL, err)
	}
	resolvedPath, err := resolvePath(request, environment)
	if err != nil {
		return "", err
	}
	base.Path = joinURLPath(base.Path, resolvedPath)
	return appendQuery(base.String(), request.Query)
}

func resolvePath(request model.Request, environment model.Environment) (string, error) {
	requestPath := strings.TrimSpace(request.Path)
	endpointName := strings.TrimSpace(request.Endpoint)
	if endpointName == "" {
		if requestPath == "" {
			return "", fmt.Errorf("request %q requires endpoint or path when url is absent: %w", request.Name, ErrCannotResolveRequestURL)
		}
		return requestPath, nil
	}
	for _, endpoint := range environment.Endpoints {
		if endpoint.Name == endpointName {
			return joinURLPath(endpoint.Path, requestPath), nil
		}
	}
	return "", fmt.Errorf("request %q endpoint %q is not defined in env %q: %w", request.Name, endpointName, environment.Name, ErrCannotResolveRequestURL)
}

func appendQuery(rawURL string, parameters []model.Parameter) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse request URL %q: %w", rawURL, err)
	}
	queryValues := parsedURL.Query()
	for _, parameter := range sortedParameters(parameters) {
		queryValues.Add(parameter.Name, parameter.Value)
	}
	parsedURL.RawQuery = queryValues.Encode()
	return parsedURL.String(), nil
}

func sortedParameters(parameters []model.Parameter) []model.Parameter {
	sorted := append([]model.Parameter(nil), parameters...)
	sort.Slice(sorted, func(left int, right int) bool {
		if sorted[left].Name == sorted[right].Name {
			return sorted[left].Value < sorted[right].Value
		}
		return sorted[left].Name < sorted[right].Name
	})
	return sorted
}

func joinURLPath(parts ...string) string {
	joined := ""
	for _, part := range parts {
		trimmed := strings.Trim(part, "/")
		if trimmed == "" {
			continue
		}
		joined += "/" + trimmed
	}
	return joined
}
