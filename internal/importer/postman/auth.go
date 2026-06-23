package postman

import (
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func importAuth(pointer string, folders []string, requestName string, auth authObject) ([]model.Header, []model.ImportWarning) {
	warnings := make([]model.ImportWarning, 0)
	appendExtraWarnings(&warnings, pointer, auth.Extra)
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "noauth":
		return nil, warnings
	case "bearer":
		token := authValue(auth.Bearer, "token")
		return []model.Header{{Name: "Authorization", Env: envNameFor(folders, requestName, "Authorization")}}, warningsForMissingAuthValue(warnings, pointer, "bearer", token)
	case "basic":
		username := authValue(auth.Basic, "username")
		password := authValue(auth.Basic, "password")
		if username == "" || password == "" {
			return nil, append(warnings, warning("postman-auth-unmapped", pointer, "Basic auth is missing username or password and was not imported."))
		}
		return []model.Header{{Name: "Authorization", Env: envNameFor(folders, requestName, "Authorization")}}, warnings
	case "apikey":
		keyName := authValue(auth.APIKey, "key")
		location := strings.ToLower(authValue(auth.APIKey, "in"))
		if location != "header" {
			return nil, append(warnings, warning("postman-auth-unmapped", pointer, "API key auth outside headers is unsupported and was not imported."))
		}
		if keyName == "" {
			return nil, append(warnings, warning("postman-auth-unmapped", pointer, "API key auth missing header name and was not imported."))
		}
		return []model.Header{{Name: keyName, Env: envNameFor(folders, requestName, keyName)}}, warnings
	default:
		return nil, append(warnings, warning("postman-auth-unmapped", pointer+".type", fmt.Sprintf("Auth type %q is unsupported and was not imported.", auth.Type)))
	}
}

func warningsForMissingAuthValue(warnings []model.ImportWarning, pointer string, authType string, value string) []model.ImportWarning {
	if value != "" {
		return warnings
	}
	return append(warnings, warning("postman-auth-unmapped", pointer, fmt.Sprintf("%s auth missing required value and was not imported.", authType)))
}

func authValue(entries []authEntry, key string) string {
	for _, entry := range entries {
		if entry.Key == key {
			return entry.Value
		}
	}
	return ""
}

func importHeader(name string, value string, envName string) model.Header {
	if secrets.IsSensitiveHeader(name) {
		return model.Header{Name: name, Env: envName}
	}
	return model.Header{Name: name, Value: value}
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
