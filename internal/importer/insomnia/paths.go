package insomnia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func explicitURL(raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return ""
}

func inferredPath(raw string) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parts := strings.SplitN(raw, "/", 4)
		if len(parts) < 4 {
			return ""
		}
		return "/" + parts[3]
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}
	return ""
}

func buildOutputPath(prefix string, folders []string, name string, suffix string) string {
	parts := []string{prefix}
	for _, folder := range folders {
		parts = append(parts, slug(folder))
	}
	parts = append(parts, slug(name)+suffix)
	return filepath.ToSlash(filepath.Join(parts...))
}

func envNameFor(folders []string, requestName string, fieldName string) string {
	parts := []string{"FLOWMAN", "INSOMNIA"}
	for _, folder := range folders {
		parts = append(parts, envToken(folder))
	}
	parts = append(parts, envToken(requestName), envToken(fieldName))
	return strings.Join(parts, "_")
}

func envToken(value string) string {
	replacer := strings.NewReplacer("-", "_", " ", "_", "/", "_", ".", "_", ":", "_")
	return strings.ToUpper(strings.Trim(replacer.Replace(value), "_"))
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-", ":", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func dedupeHeaders(headers []model.Header) []model.Header {
	seen := map[string]int{}
	result := make([]model.Header, 0, len(headers))
	for _, header := range headers {
		key := strings.ToLower(strings.TrimSpace(header.Name))
		if index, ok := seen[key]; ok {
			result[index] = header
			continue
		}
		seen[key] = len(result)
		result = append(result, header)
	}
	return result
}

func writeWarningReport(path string, warnings []model.ImportWarning) error {
	if path == "" {
		return nil
	}
	payload := struct {
		Warnings []model.ImportWarning `json:"warnings"`
	}{Warnings: warnings}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal insomnia warning report: %w", err)
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create warning report directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write warning report %s: %w", path, err)
	}
	return nil
}
