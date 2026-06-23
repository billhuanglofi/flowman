package secrets_test

import (
	"strings"
	"testing"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func Test_RedactHeaderValue_redacts_sensitive_header_case_variants(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "authorization", header: "Authorization"},
		{name: "authorization lowercase", header: "authorization"},
		{name: "cookie mixed case", header: "CoOkIe"},
		{name: "set cookie", header: "Set-Cookie"},
		{name: "api key", header: "X-API-Key"},
		{name: "proxy authorization", header: "Proxy-Authorization"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			value := strings.Repeat("x", 8)

			// When
			got := secrets.RedactHeaderValue(tt.header, value)

			// Then
			if got != "[REDACTED]" {
				t.Fatalf("expected sensitive header value redacted, got %q", got)
			}
		})
	}
}

func Test_RedactHeaderValue_preserves_non_sensitive_header(t *testing.T) {
	// Given
	value := "flowman-example-correlation"

	// When
	got := secrets.RedactHeaderValue("X-Correlation-ID", value)

	// Then
	if got != value {
		t.Fatalf("expected non-sensitive header preserved, got %q", got)
	}
}

func Test_RedactHeaders_redacts_sensitive_values_without_mutating_input(t *testing.T) {
	// Given
	value := strings.Repeat("x", 8)
	headers := []model.Header{
		{Name: "Authorization", Value: value},
		{Name: "X-Correlation-ID", Value: "flowman-example-correlation"},
	}

	// When
	got := secrets.RedactHeaders(headers)

	// Then
	if got[0].Value != "[REDACTED]" {
		t.Fatalf("expected sensitive header redacted, got %#v", got[0])
	}
	if got[1].Value != "flowman-example-correlation" {
		t.Fatalf("expected non-sensitive header preserved, got %#v", got[1])
	}
	if strings.Contains(got[0].Value, value) {
		t.Fatalf("expected redacted output without sensitive value, got %#v", got[0])
	}
	if headers[0].Value != value {
		t.Fatalf("expected input headers not to be mutated, got %#v", headers[0])
	}
}

func Test_IsSensitiveHeader_matches_required_header_names_only(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "Authorization", want: true},
		{name: "Cookie", want: true},
		{name: "Set-Cookie", want: true},
		{name: "X-API-Key", want: true},
		{name: "Proxy-Authorization", want: true},
		{name: "X-Correlation-ID", want: false},
		{name: "Content-Type", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := secrets.IsSensitiveHeader(tt.name)

			// Then
			if got != tt.want {
				t.Fatalf("expected %q sensitivity %t, got %t", tt.name, tt.want, got)
			}
		})
	}
}

func Test_RedactResponseBody_redacts_sensitive_JSON_fields(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantRedact  []string
		wantKeep    []string
	}{
		{
			name:        "simple api_key field",
			body:        `{"api_key":"secret123","user":"john"}`,
			contentType: "application/json",
			wantRedact:  []string{"secret123"},
			wantKeep:    []string{"john"},
		},
		{
			name:        "access_token field",
			body:        `{"access_token":"Bearer xyz","status":"ok"}`,
			contentType: "application/json",
			wantRedact:  []string{"Bearer xyz"},
			wantKeep:    []string{"ok"},
		},
		{
			name:        "nested password field",
			body:        `{"user":{"name":"alice","password":"secret"},"id":123}`,
			contentType: "application/json",
			wantRedact:  []string{"secret"},
			wantKeep:    []string{"alice"},
		},
		{
			name:        "array with sensitive data",
			body:        `{"tokens":[{"token":"abc"},{"token":"def"}]}`,
			contentType: "application/json",
			wantRedact:  []string{"abc", "def"},
			wantKeep:    []string{},
		},
		{
			name:        "non-JSON content type returns unchanged",
			body:        `{"password":"secret"}`,
			contentType: "text/plain",
			wantRedact:  []string{},
			wantKeep:    []string{"secret"},
		},
		{
			name:        "invalid JSON returns unchanged",
			body:        `{invalid json}`,
			contentType: "application/json",
			wantRedact:  []string{},
			wantKeep:    []string{"{invalid json}"},
		},
		{
			name:        "multiple sensitive fields",
			body:        `{"password":"p1","api_key":"k1","client_secret":"s1","data":"safe"}`,
			contentType: "application/json",
			wantRedact:  []string{"p1", "k1", "s1"},
			wantKeep:    []string{"safe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := secrets.RedactResponseBody([]byte(tt.body), tt.contentType)
			result := string(got)

			// Then - check redacted values are gone
			for _, sensitive := range tt.wantRedact {
				if strings.Contains(result, sensitive) {
					t.Errorf("expected %q to be redacted, but found in: %s", sensitive, result)
				}
			}

			// Then - check safe values are kept
			for _, safe := range tt.wantKeep {
				if !strings.Contains(result, safe) {
					t.Errorf("expected %q to be kept, but not found in: %s", safe, result)
				}
			}

			// Then - check [REDACTED] marker present for JSON with sensitive fields
			if len(tt.wantRedact) > 0 && tt.contentType == "application/json" {
				if !strings.Contains(result, "[REDACTED]") {
					t.Errorf("expected [REDACTED] marker in result: %s", result)
				}
			}
		})
	}
}

func Test_IsSensitiveJSONField_matches_common_sensitive_field_names(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{field: "password", want: true},
		{field: "Password", want: true},
		{field: "PASSWORD", want: true},
		{field: "api_key", want: true},
		{field: "apiKey", want: true},
		{field: "api-key", want: true},
		{field: "access_token", want: true},
		{field: "accessToken", want: true},
		{field: "refresh_token", want: true},
		{field: "token", want: true},
		{field: "secret", want: true},
		{field: "client_secret", want: true},
		{field: "private_key", want: true},
		{field: "bearer", want: true},
		{field: "authorization", want: true},
		{field: "credentials", want: true},
		{field: "username", want: false},
		{field: "email", want: false},
		{field: "id", want: false},
		{field: "status", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			// When
			got := secrets.IsSensitiveJSONField(tt.field)

			// Then
			if got != tt.want {
				t.Errorf("IsSensitiveJSONField(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}
