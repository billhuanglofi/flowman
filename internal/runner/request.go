package runner

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

var ErrCannotResolveRequestURL = errors.New("cannot resolve request URL")

var ErrUnsupportedBodyMode = errors.New("runner: unsupported body mode")

var ErrMissingHeaderEnv = errors.New("runner: missing header environment variable")

type Execution struct {
	Request     model.Request
	Environment model.Environment
	URL         string
}

type preparedRequest struct {
	method string
	url    string
	body   io.Reader
}

func prepareRequest(execution Execution) (preparedRequest, error) {
	resolvedURL, err := resolveRequestURL(execution)
	if err != nil {
		return preparedRequest{}, err
	}
	body, err := requestBody(execution.Request.Body)
	if err != nil {
		return preparedRequest{}, fmt.Errorf("request %q body: %w", execution.Request.Name, err)
	}
	method := strings.TrimSpace(execution.Request.Method)
	return preparedRequest{method: method, url: resolvedURL, body: body}, nil
}

func resolveRequestURL(execution Execution) (string, error) {
	explicitURL := strings.TrimSpace(execution.URL)
	if explicitURL != "" {
		return explicitURL, nil
	}
	requestURL := strings.TrimSpace(execution.Request.URL)
	if requestURL != "" {
		return requestURL, nil
	}
	baseURL := strings.TrimSpace(execution.Environment.BaseURL)
	if baseURL == "" {
		return "", cannotResolveURL(execution, "environment base_url is empty")
	}
	path, err := resolveEnvironmentPath(execution)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("request %q environment base_url %q: %w", execution.Request.Name, baseURL, err)
	}
	base.Path = joinURLPath(base.Path, path)
	return base.String(), nil
}

func resolveEnvironmentPath(execution Execution) (string, error) {
	requestPath := strings.TrimSpace(execution.Request.Path)
	endpointName := strings.TrimSpace(execution.Request.Endpoint)
	if endpointName == "" {
		if requestPath == "" {
			return "", cannotResolveURL(execution, "request url, endpoint, and path are empty")
		}
		return requestPath, nil
	}
	for _, endpoint := range execution.Environment.Endpoints {
		if endpoint.Name == endpointName {
			return joinURLPath(endpoint.Path, requestPath), nil
		}
	}
	return "", cannotResolveURL(execution, fmt.Sprintf("endpoint %q is not defined in environment %q", endpointName, execution.Environment.Name))
}

func applyQuery(rawURL string, parameters []model.Parameter) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse request URL %q: %w", rawURL, err)
	}
	query := parsedURL.Query()
	for _, parameter := range parameters {
		query.Add(parameter.Name, parameter.Value)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func applyHeaders(httpRequest *http.Request, headers []model.Header) error {
	for _, header := range headers {
		value, err := headerValue(header)
		if err != nil {
			return err
		}
		httpRequest.Header.Add(header.Name, value)
	}
	return nil
}

func headerValue(header model.Header) (string, error) {
	envName := strings.TrimSpace(header.Env)
	if envName == "" {
		return header.Value, nil
	}
	value, ok := os.LookupEnv(envName)
	if !ok {
		return "", fmt.Errorf("header %q env %q: %w", header.Name, envName, ErrMissingHeaderEnv)
	}
	return value, nil
}

func requestBody(body model.RequestBody) (io.Reader, error) {
	mode := strings.TrimSpace(strings.ToLower(body.Mode))
	switch mode {
	case "":
		return nil, nil
	case "raw", "json":
		return strings.NewReader(body.Raw), nil
	case "form":
		values := url.Values{}
		for _, parameter := range body.Form {
			values.Add(parameter.Name, parameter.Value)
		}
		return strings.NewReader(values.Encode()), nil
	default:
		return nil, fmt.Errorf("%q: %w", body.Mode, ErrUnsupportedBodyMode)
	}
}

func cannotResolveURL(execution Execution, reason string) error {
	return fmt.Errorf("request %q: cannot resolve request URL: %s: %w", execution.Request.Name, reason, ErrCannotResolveRequestURL)
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
	if joined == "" {
		return ""
	}
	return joined
}
