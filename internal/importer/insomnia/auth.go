package insomnia

import (
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func importAuthentication(path string, folders []string, requestName string, auth map[string]any) ([]model.Header, []model.ImportWarning) {
	authType := strings.ToLower(strings.TrimSpace(stringValue(auth["type"])))
	switch authType {
	case "", "none":
		return nil, nil
	case "bearer":
		token := strings.TrimSpace(stringValue(auth["token"]))
		header := model.Header{Name: "Authorization", Env: envNameFor(folders, requestName, "Authorization")}
		if token == "" {
			return []model.Header{header}, []model.ImportWarning{{Code: "insomnia-auth-unmapped", Path: path, Message: "Bearer auth is missing a token and was imported as an env-backed Authorization header."}}
		}
		return []model.Header{header}, nil
	case "basic":
		username := strings.TrimSpace(stringValue(auth["username"]))
		password := strings.TrimSpace(stringValue(auth["password"]))
		header := model.Header{Name: "Authorization", Env: envNameFor(folders, requestName, "Authorization")}
		if username == "" || password == "" {
			return []model.Header{header}, []model.ImportWarning{{Code: "insomnia-auth-unmapped", Path: path, Message: "Basic auth is missing username or password and was imported as an env-backed Authorization header."}}
		}
		return []model.Header{header}, nil
	case "apikey":
		keyName := strings.TrimSpace(stringValue(auth["key"]))
		addTo := strings.ToLower(strings.TrimSpace(stringValue(auth["addTo"])))
		if addTo != "header" || keyName == "" {
			return nil, []model.ImportWarning{{Code: "insomnia-auth-unmapped", Path: path, Message: "API key auth outside headers or without a header name is unsupported and was not imported."}}
		}
		return []model.Header{{Name: keyName, Env: envNameFor(folders, requestName, keyName)}}, nil
	default:
		return nil, []model.ImportWarning{{Code: "insomnia-auth-unsupported", Path: path, Message: fmt.Sprintf("Insomnia auth type %q is unsupported and was reported instead of discarded.", authType)}}
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
