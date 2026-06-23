package postman

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func scriptWarnings(pointer string, events []eventObject) []model.ImportWarning {
	warnings := make([]model.ImportWarning, 0)
	for index, event := range events {
		if len(event.Script.Exec) == 0 {
			continue
		}
		code := "postman-script-unsupported"
		message := "Postman scripts are not executable Flowman features and were reported instead of discarded."
		if strings.EqualFold(event.Listen, "prerequest") {
			code = "postman-prerequest-script-unsupported"
			message = "Pre-request scripts are unsupported and were reported instead of discarded."
		}
		if strings.EqualFold(event.Listen, "test") {
			code = "postman-test-script-unsupported"
			message = "Test scripts are unsupported and were reported instead of discarded."
		}
		warnings = append(warnings, warning(code, fmt.Sprintf("%s.event[%d].script.exec", pointer, index), message))
	}
	return warnings
}

func appendExtraWarnings(warnings *[]model.ImportWarning, pointer string, extra map[string]json.RawMessage) {
	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		*warnings = append(*warnings, warning("postman-field-unmapped", pointer+"."+key, fmt.Sprintf("Field %q is not mapped into canonical Flowman YAML.", key)))
	}
}

func warning(code string, path string, message string) model.ImportWarning {
	return model.ImportWarning{Code: code, Path: path, Message: message}
}

func marshalWarningReport(warnings []model.ImportWarning) ([]byte, error) {
	payload := struct {
		Warnings []model.ImportWarning `json:"warnings"`
	}{Warnings: warnings}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal postman warning report: %w", err)
	}
	return append(content, '\n'), nil
}

func warningsText(warnings []model.ImportWarning) string {
	var builder strings.Builder
	for _, item := range warnings {
		builder.WriteString(item.Code)
		builder.WriteString(item.Path)
		builder.WriteString(item.Message)
	}
	return builder.String()
}
