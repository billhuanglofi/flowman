package postman

import (
	"path/filepath"
	"strings"
)

func buildOutputPath(prefix string, folders []string, name string, suffix string) string {
	parts := []string{prefix}
	for _, folder := range folders {
		parts = append(parts, slug(folder))
	}
	parts = append(parts, slug(name)+suffix)
	return filepath.ToSlash(filepath.Join(parts...))
}

func envNameFor(folders []string, requestName string, fieldName string) string {
	parts := []string{"FLOWMAN", "POSTMAN"}
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
