package postman

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func exportRequest(request model.Request) *requestShape {
	headers, auth := exportHeaders(request.Headers)
	shape := &requestShape{Method: normalizeMethod(request.Method), Header: headers, URL: exportURL(request)}
	if body := exportBody(request.Body); body != nil {
		shape.Body = body
	}
	if auth != nil {
		shape.Auth = auth
	}
	return shape
}

func exportHeaders(headers []model.Header) ([]headerShape, *authShape) {
	result := make([]headerShape, 0, len(headers))
	var auth *authShape
	for _, header := range headers {
		name := strings.TrimSpace(header.Name)
		if header.Env != "" {
			value := postmanVariable(header.Env)
			if strings.EqualFold(name, "Authorization") {
				auth = &authShape{Type: "bearer", Bearer: []authEntry{{Key: "token", Value: value}}}
				continue
			}
			if strings.EqualFold(name, "X-API-Key") {
				auth = &authShape{Type: "apikey", APIKey: []authEntry{{Key: "key", Value: name}, {Key: "value", Value: value}, {Key: "in", Value: "header"}}}
				continue
			}
			if secrets.IsSensitiveHeader(name) {
				result = append(result, headerShape{Key: name, Value: value})
				continue
			}
		}
		result = append(result, headerShape{Key: name, Value: header.Value})
	}
	sort.Slice(result, func(left int, right int) bool { return result[left].Key < result[right].Key })
	return result, auth
}

func exportBody(body model.RequestBody) *bodyShape {
	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	switch mode {
	case "", "raw":
		if strings.TrimSpace(body.Raw) == "" {
			return nil
		}
		return &bodyShape{Mode: "raw", Raw: body.Raw}
	case "json":
		return &bodyShape{Mode: "raw", Raw: body.Raw, Options: &bodyOptions{Raw: map[string]string{"language": "json"}}}
	case "form":
		fields := make([]headerShape, 0, len(body.Form))
		for _, parameter := range sortedParameters(body.Form) {
			fields = append(fields, headerShape{Key: parameter.Name, Value: parameter.Value})
		}
		return &bodyShape{Mode: "urlencoded", URLEnc: fields}
	default:
		return nil
	}
}

func exportURL(request model.Request) any {
	if strings.TrimSpace(request.URL) != "" {
		resolved, err := appendQuery(request.URL, request.Query)
		if err == nil {
			return resolved
		}
		return request.URL
	}
	query := make([]map[string]string, 0, len(request.Query))
	for _, parameter := range sortedParameters(request.Query) {
		query = append(query, map[string]string{"key": parameter.Name, "value": parameter.Value})
	}
	return map[string]any{"raw": request.Path, "path": splitPath(request.Path), "query": query}
}

func appendQuery(rawURL string, parameters []model.Parameter) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	values := parsedURL.Query()
	for _, parameter := range sortedParameters(parameters) {
		values.Add(parameter.Name, parameter.Value)
	}
	parsedURL.RawQuery = values.Encode()
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

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func normalizeMethod(method string) string {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "GET"
	}
	return strings.ToUpper(trimmed)
}

func postmanVariable(name string) string {
	return fmt.Sprintf("{{%s}}", name)
}
