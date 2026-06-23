package safestore

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func parseHeaders(raw string, transactionID string) ([]model.Header, []model.ImportWarning, error) {
	var headerJSON map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &headerJSON); err != nil {
		return nil, nil, fmt.Errorf("parse Safestore headers JSON object: %w: %w", err, ErrInvalidHeaderJSON)
	}
	names := sortedHeaderNames(headerJSON)
	headers := make([]model.Header, 0, len(headerJSON))
	warnings := make([]model.ImportWarning, 0)
	for _, name := range names {
		values, err := headerValues(headerJSON[name])
		if err != nil {
			return nil, nil, fmt.Errorf("headers.%s: %w", name, err)
		}
		for _, value := range values {
			header, warning := importHeader(name, value, transactionID)
			headers = append(headers, header)
			if warning.Code != "" {
				warnings = append(warnings, warning)
			}
		}
	}
	return headers, warnings, nil
}

func sortedHeaderNames(headers map[string]json.RawMessage) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func headerValues(raw json.RawMessage) ([]string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return []string{value}, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("header value must be string or string array: %w", ErrInvalidHeaderJSON)
	}
	return values, nil
}

func importHeader(name string, value string, transactionID string) (model.Header, model.ImportWarning) {
	if !secrets.IsSensitiveHeader(name) {
		return model.Header{Name: name, Value: value}, model.ImportWarning{}
	}
	envName := secretEnvName(transactionID, name)
	warning := model.ImportWarning{
		Code:    "sensitive-header-env-reference",
		Path:    "headers." + name,
		Message: fmt.Sprintf("Imported sensitive header %q as env reference %s; raw value was not written.", name, envName),
	}
	return model.Header{Name: name, Env: envName}, warning
}

func secretEnvName(transactionID string, headerName string) string {
	parts := []string{"FLOWMAN", "IMPORTED", transactionID, headerName}
	return sanitizeEnvName(strings.Join(parts, "_"))
}

func sanitizeEnvName(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			builder.WriteRune(char)
			continue
		}
		if char >= 'a' && char <= 'z' {
			builder.WriteRune(char - 'a' + 'A')
			continue
		}
		builder.WriteByte('_')
	}
	return strings.Join(strings.FieldsFunc(builder.String(), func(char rune) bool { return char == '_' }), "_")
}
