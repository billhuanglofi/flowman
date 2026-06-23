package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func renderHeaders(headers []model.Header, allowUnsafe bool) ([]renderedHeader, error) {
	rendered := make([]renderedHeader, 0, len(headers))
	for _, header := range headers {
		renderedHeaderValue, err := renderHeader(header, allowUnsafe)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, renderedHeaderValue)
	}
	sort.Slice(rendered, func(left int, right int) bool {
		if rendered[left].sortKey == rendered[right].sortKey {
			return rendered[left].name < rendered[right].name
		}
		return rendered[left].sortKey < rendered[right].sortKey
	})
	return rendered, nil
}

func renderHeader(header model.Header, allowUnsafe bool) (renderedHeader, error) {
	name := strings.TrimSpace(header.Name)
	if envName := strings.TrimSpace(header.Env); envName != "" {
		return renderedHeader{
			name:      name,
			curlValue: fmt.Sprintf("%s: ${%s}", name, envName),
			httpValue: fmt.Sprintf("%s: {{$processEnv %s}}", name, envName),
			sortKey:   strings.ToLower(name),
		}, nil
	}
	if secrets.IsSensitiveHeader(name) && strings.TrimSpace(header.Value) != "" && !allowUnsafe {
		return renderedHeader{}, fmt.Errorf("header %q must use env reference or unsafe flag: %w", name, ErrRawSensitiveHeader)
	}
	value := header.Value
	return renderedHeader{
		name:      name,
		curlValue: name + ": " + value,
		httpValue: name + ": " + value,
		sortKey:   strings.ToLower(name),
	}, nil
}
