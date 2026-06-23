package secrets

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

const RedactedValue = "[REDACTED]"

var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"proxy-authorization": {},
}

var sensitiveJSONFields = map[string]struct{}{
	"password":       {},
	"api_key":        {},
	"apikey":         {},
	"api-key":        {},
	"access_token":   {},
	"accesstoken":    {},
	"access-token":   {},
	"refresh_token":  {},
	"refreshtoken":   {},
	"refresh-token":  {},
	"token":          {},
	"secret":         {},
	"client_secret":  {},
	"clientsecret":   {},
	"client-secret":  {},
	"private_key":    {},
	"privatekey":     {},
	"private-key":    {},
	"bearer":         {},
	"authorization":  {},
	"auth":           {},
	"credentials":    {},
	"credential":     {},
}

func IsSensitiveHeader(name string) bool {
	_, ok := sensitiveHeaders[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func RedactHeaderValue(name string, value string) string {
	if IsSensitiveHeader(name) {
		return RedactedValue
	}
	return value
}

func RedactHeaders(headers []model.Header) []model.Header {
	redacted := make([]model.Header, 0, len(headers))
	for _, header := range headers {
		redacted = append(redacted, model.Header{
			Name:  header.Name,
			Value: RedactHeaderValue(header.Name, header.Value),
			Env:   header.Env,
		})
	}
	return redacted
}

// IsSensitiveJSONField checks if a JSON field name contains sensitive data
func IsSensitiveJSONField(fieldName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(fieldName))
	_, ok := sensitiveJSONFields[normalized]
	return ok
}

// RedactResponseBody redacts sensitive fields in JSON response bodies
// If the body is not valid JSON, it returns the original body unchanged
func RedactResponseBody(body []byte, contentType string) []byte {
	// Only process JSON content types
	if !strings.Contains(strings.ToLower(contentType), "json") {
		return body
	}

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Not valid JSON, return original
		return body
	}

	// Redact sensitive fields recursively
	redacted := redactJSONValue(data)

	// Marshal back to JSON
	result, err := json.Marshal(redacted)
	if err != nil {
		// If marshaling fails, return original
		return body
	}

	return result
}

// redactJSONValue recursively redacts sensitive fields in JSON structures
func redactJSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			if IsSensitiveJSONField(key) {
				result[key] = RedactedValue
			} else {
				result[key] = redactJSONValue(val)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = redactJSONValue(item)
		}
		return result
	default:
		return v
	}
}

// RedactResponseBodyPretty redacts and pretty-prints JSON for display
func RedactResponseBodyPretty(body []byte, contentType string) []byte {
	redacted := RedactResponseBody(body, contentType)

	// Try to pretty-print JSON
	if strings.Contains(strings.ToLower(contentType), "json") {
		var buf bytes.Buffer
		if err := json.Indent(&buf, redacted, "", "  "); err == nil {
			return buf.Bytes()
		}
	}

	return redacted
}
