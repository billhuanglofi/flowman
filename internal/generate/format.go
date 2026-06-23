package generate

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func formatJSON(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func renderFormBody(parameters []model.Parameter) string {
	values := url.Values{}
	for _, parameter := range sortedParameters(parameters) {
		values.Add(parameter.Name, parameter.Value)
	}
	return values.Encode()
}

func shellEscapeSingleQuoted(value string) string {
	var builder bytes.Buffer
	for _, char := range value {
		if char == '\'' {
			builder.WriteString(`'"'"'`)
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}
