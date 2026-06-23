package safestore

import (
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func resolveURL(options ImportOptions) (resolvedURL, error) {
	if explicitURL := strings.TrimSpace(options.URL); explicitURL != "" {
		return resolvedURL{url: explicitURL}, nil
	}
	if strings.TrimSpace(options.Endpoint) != "" {
		if _, err := endpointPath(options.Environment, options.Endpoint); err != nil {
			return resolvedURL{}, err
		}
		return resolvedURL{endpoint: options.Endpoint, path: strings.TrimSpace(options.Path)}, nil
	}
	if requestPath := strings.TrimSpace(options.Path); requestPath != "" && strings.TrimSpace(options.Environment.BaseURL) != "" {
		return resolvedURL{path: requestPath}, nil
	}
	if options.NonInteractive {
		return resolvedURL{}, fmt.Errorf("transaction %q needs --url, --endpoint, or env base_url + --path when --non-interactive is set: %w: %w", options.TransactionID, ErrCannotResolveURLNonInteractive, ErrCannotResolveURL)
	}
	return resolvedURL{}, fmt.Errorf("transaction %q needs URL resolution input before interactive prompting can be added to a caller: %w: %w", options.TransactionID, ErrInteractiveURLResolutionNotImplemented, ErrCannotResolveURL)
}

func endpointPath(environment model.Environment, endpointName string) (string, error) {
	for _, endpoint := range environment.Endpoints {
		if endpoint.Name == endpointName {
			return endpoint.Path, nil
		}
	}
	return "", fmt.Errorf("endpoint %q is not defined in environment %q: %w", endpointName, environment.Name, ErrCannotResolveURL)
}
