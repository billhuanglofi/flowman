package runner

import (
	"net/http"
	"time"

	"github.com/billhuanglofi/flowman/internal/secrets"
)

type Response struct {
	StatusCode     int
	Headers        http.Header
	DisplayHeaders http.Header
	Body           []byte
	DisplayBody    []byte // Redacted body safe for display/logs
	Duration       time.Duration
}

func cloneHeaders(headers http.Header) http.Header {
	cloned := make(http.Header, len(headers))
	for name, values := range headers {
		cloned[name] = append([]string(nil), values...)
	}
	return cloned
}

func redactedHeaders(headers http.Header) http.Header {
	redacted := make(http.Header, len(headers))
	for name, values := range headers {
		if secrets.IsSensitiveHeader(name) {
			redacted[name] = []string{secrets.RedactedValue}
			continue
		}
		redacted[name] = append([]string(nil), values...)
	}
	return redacted
}
