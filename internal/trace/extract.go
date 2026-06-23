package trace

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
)

var ErrTransactionIDNotFound = errors.New("trace: transaction id not found")

var ErrInvalidJSONPath = errors.New("trace: invalid JSON path")

var ErrInvalidRegex = errors.New("trace: invalid transaction id regex")

type ExtractionInput struct {
	Override string
	Config   model.TraceConfig
	Response runner.Response
}

// Supported JSON path syntax is a small deterministic subset: root `$`, dot
// object keys like `$.data.transactionId`, and quoted bracket object keys like
// `$["transaction-id"]` or `$.data['transaction-id']`. Arrays, wildcards,
// recursive descent, filters, and unquoted bracket keys are intentionally not
// supported.
func ExtractTransactionID(input ExtractionInput) (string, error) {
	attempts := []string{"cli override --tx"}
	if tx := strings.TrimSpace(input.Override); tx != "" {
		return tx, nil
	}
	if tx, source := extractHeader(input.Response.Headers, input.Config.TransactionID.Header); tx != "" {
		return tx, nil
	} else {
		attempts = append(attempts, source)
	}
	for _, path := range input.Config.TransactionID.JSONPaths {
		tx, err := extractJSONPath(input.Response.Body, path)
		if err != nil {
			return "", err
		}
		attempts = append(attempts, "json path "+path)
		if tx != "" {
			return tx, nil
		}
	}
	if len(input.Config.TransactionID.JSONPaths) == 0 {
		attempts = append(attempts, "no configured JSON path")
	}
	if tx, source, err := extractRegex(input.Response.Body, input.Config.TransactionID.Regex); err != nil {
		return "", err
	} else if tx != "" {
		return tx, nil
	} else {
		attempts = append(attempts, source)
	}
	return "", fmt.Errorf("transaction id extraction failed after attempting %s: %w", strings.Join(attempts, ", "), ErrTransactionIDNotFound)
}

func extractHeader(headers http.Header, headerName string) (string, string) {
	name := strings.TrimSpace(headerName)
	if name == "" {
		return "", "no configured response header"
	}
	value := strings.TrimSpace(headers.Get(name))
	if value != "" {
		return value, "header " + name
	}
	for responseName, values := range headers {
		if !strings.EqualFold(responseName, name) || len(values) == 0 {
			continue
		}
		return strings.TrimSpace(values[0]), "header " + name
	}
	return "", "header " + name
}

func extractJSONPath(body []byte, path string) (string, error) {
	segments, err := parseJSONPath(path)
	if err != nil {
		return "", err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return "", nil
	}
	current := value
	for _, segment := range segments {
		object, ok := current.(map[string]any)
		if !ok {
			return "", nil
		}
		child, ok := object[segment]
		if !ok {
			return "", nil
		}
		current = child
	}
	text, ok := current.(string)
	if !ok {
		return "", nil
	}
	return strings.TrimSpace(text), nil
}

func parseJSONPath(path string) ([]string, error) {
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("json path %q must start with $: %w", path, ErrInvalidJSONPath)
	}
	segments := make([]string, 0)
	for index := 1; index < len(path); {
		switch path[index] {
		case '.':
			segment, next, err := parseDotSegment(path, index+1)
			if err != nil {
				return nil, err
			}
			segments = append(segments, segment)
			index = next
		case '[':
			segment, next, err := parseBracketSegment(path, index+1)
			if err != nil {
				return nil, err
			}
			segments = append(segments, segment)
			index = next
		default:
			return nil, fmt.Errorf("json path %q contains unsupported token at byte %d: %w", path, index, ErrInvalidJSONPath)
		}
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("json path %q must select an object key: %w", path, ErrInvalidJSONPath)
	}
	return segments, nil
}

func parseDotSegment(path string, start int) (string, int, error) {
	if start >= len(path) || !isIdentifierStart(rune(path[start])) {
		return "", 0, fmt.Errorf("json path %q has invalid dot key at byte %d: %w", path, start, ErrInvalidJSONPath)
	}
	end := start + 1
	for end < len(path) && isIdentifierPart(rune(path[end])) {
		end++
	}
	return path[start:end], end, nil
}

func parseBracketSegment(path string, start int) (string, int, error) {
	if start >= len(path) || (path[start] != '\'' && path[start] != '"') {
		return "", 0, fmt.Errorf("json path %q bracket key must be quoted at byte %d: %w", path, start, ErrInvalidJSONPath)
	}
	quote := path[start]
	end := start + 1
	for end < len(path) && path[end] != quote {
		end++
	}
	if end >= len(path) || end+1 >= len(path) || path[end+1] != ']' {
		return "", 0, fmt.Errorf("json path %q has unterminated bracket key: %w", path, ErrInvalidJSONPath)
	}
	segment := path[start+1 : end]
	if segment == "" {
		return "", 0, fmt.Errorf("json path %q has empty bracket key: %w", path, ErrInvalidJSONPath)
	}
	return segment, end + 2, nil
}

func isIdentifierStart(char rune) bool {
	return char == '_' || unicode.IsLetter(char)
}

func isIdentifierPart(char rune) bool {
	return isIdentifierStart(char) || unicode.IsDigit(char)
}

func extractRegex(body []byte, pattern string) (string, string, error) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return "", "no configured body regex", nil
	}
	expression, err := regexp.Compile(trimmed)
	if err != nil {
		return "", "", fmt.Errorf("body regex %q: %w: %w", trimmed, err, ErrInvalidRegex)
	}
	matches := expression.FindSubmatch(body)
	if len(matches) == 0 {
		return "", "body regex " + trimmed, nil
	}
	if len(matches) > 1 {
		return strings.TrimSpace(string(matches[1])), "body regex " + trimmed, nil
	}
	return strings.TrimSpace(string(matches[0])), "body regex " + trimmed, nil
}
