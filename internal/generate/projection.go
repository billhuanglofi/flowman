package generate

import (
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func newProjection(request model.Request, options Options) (projection, error) {
	resolvedURL, err := resolveRequestURL(request, options.Environment)
	if err != nil {
		return projection{}, err
	}
	headers, err := renderHeaders(request.Headers, options.UnsafeAllowSensitiveHeaders)
	if err != nil {
		return projection{}, err
	}
	body, err := renderBody(request.Body)
	if err != nil {
		return projection{}, err
	}
	return projection{
		environmentName: strings.TrimSpace(options.Environment.Name),
		method:          normalizeMethod(request.Method),
		url:             resolvedURL,
		headers:         headerLines(headers, curlHeaderLine),
		httpHeaders:     headerLines(headers, httpHeaderLine),
		body:            body,
	}, nil
}

func normalizeMethod(method string) string {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "GET"
	}
	return strings.ToUpper(trimmed)
}

func headerLines(headers []renderedHeader, line func(renderedHeader) string) []string {
	lines := make([]string, 0, len(headers))
	for _, header := range headers {
		lines = append(lines, line(header))
	}
	return lines
}

func curlHeaderLine(header renderedHeader) string {
	return header.curlValue
}

func httpHeaderLine(header renderedHeader) string {
	return header.httpValue
}

func renderBody(body model.RequestBody) (string, error) {
	mode := strings.TrimSpace(strings.ToLower(body.Mode))
	switch mode {
	case "", "raw":
		return strings.TrimSpace(body.Raw), nil
	case "json":
		return formatJSON(body.Raw)
	case "form":
		return renderFormBody(body.Form), nil
	default:
		return "", fmt.Errorf("body mode %q is unsupported for generation", body.Mode)
	}
}
