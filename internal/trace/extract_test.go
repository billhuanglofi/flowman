package trace_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/runner"
	"github.com/billhuanglofi/flowman/internal/trace"
)

func TestTransactionIDExtractionPrecedence(t *testing.T) {
	// Given
	response := runner.Response{
		Headers: http.Header{"X-Transaction-ID": []string{"header-tx"}},
		Body:    []byte(`{"transaction_id":"json-tx","message":"transaction_id=regex-tx"}`),
	}
	config := model.TraceConfig{
		TransactionID: model.TransactionIDExtraction{
			Header:    "X-Transaction-ID",
			JSONPaths: []string{"$.transaction_id"},
			Regex:     `transaction_id=([A-Za-z0-9_-]+)`,
		},
		PollInterval: model.NewDuration(2 * time.Second),
		Timeout:      model.NewDuration(30 * time.Second),
	}

	// When
	got, err := trace.ExtractTransactionID(trace.ExtractionInput{Override: "override-tx", Config: config, Response: response})

	// Then
	if err != nil {
		t.Fatalf("expected override extraction to succeed: %v", err)
	}
	if got != "override-tx" {
		t.Fatalf("expected CLI override to win over header, JSON, and regex, got %q", got)
	}

	// When
	got, err = trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if err != nil {
		t.Fatalf("expected header extraction to succeed: %v", err)
	}
	if got != "header-tx" {
		t.Fatalf("expected configured header to win over JSON and regex, got %q", got)
	}

	// When
	jsonResponse := runner.Response{Body: response.Body}
	got, err = trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: jsonResponse})

	// Then
	if err != nil {
		t.Fatalf("expected JSON path extraction to succeed: %v", err)
	}
	if got != "json-tx" {
		t.Fatalf("expected configured JSON path to win over regex, got %q", got)
	}

	// When
	regexResponse := runner.Response{Body: []byte(`{"message":"transaction_id=regex-tx"}`)}
	got, err = trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: regexResponse})

	// Then
	if err != nil {
		t.Fatalf("expected regex extraction to succeed: %v", err)
	}
	if got != "regex-tx" {
		t.Fatalf("expected configured regex fallback, got %q", got)
	}
}

func TestTransactionIDExtractionHeader_uses_configured_response_header(t *testing.T) {
	// Given
	response := runner.Response{Headers: http.Header{"X-Flowman-Tx": []string{"  header-tx  "}}}
	config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{Header: "X-Flowman-Tx"}}

	// When
	got, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if err != nil {
		t.Fatalf("expected configured response header extraction to succeed: %v", err)
	}
	if got != "header-tx" {
		t.Fatalf("expected trimmed header transaction id, got %q", got)
	}
}

func TestTransactionIDExtractionJSONPath_supports_root_and_nested_dot_paths(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want string
	}{
		{name: "root snake case", path: "$.transaction_id", body: `{"transaction_id":"root-tx"}`, want: "root-tx"},
		{name: "nested camel case", path: "$.data.transactionId", body: `{"data":{"transactionId":"nested-tx"}}`, want: "nested-tx"},
		{name: "bracket quoted key", path: `$.data["transaction-id"]`, body: `{"data":{"transaction-id":"bracket-tx"}}`, want: "bracket-tx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			response := runner.Response{Body: []byte(tt.body)}
			config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{JSONPaths: []string{tt.path}}}

			// When
			got, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

			// Then
			if err != nil {
				t.Fatalf("expected JSON path %s extraction to succeed: %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("expected JSON path transaction id %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTransactionIDExtractionRegex_uses_first_capture_group(t *testing.T) {
	// Given
	response := runner.Response{Body: []byte(`accepted transaction_id=regex-tx status=ok`)}
	config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{Regex: `transaction_id=([A-Za-z0-9_-]+)`}}

	// When
	got, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if err != nil {
		t.Fatalf("expected regex extraction to succeed: %v", err)
	}
	if got != "regex-tx" {
		t.Fatalf("expected first capture group transaction id, got %q", got)
	}
}

func TestTransactionIDExtractionFailure_names_attempted_sources_when_empty_or_missing(t *testing.T) {
	// Given
	response := runner.Response{
		Headers: http.Header{"X-Transaction-ID": []string{"   "}},
		Body:    []byte(`{"data":{"other":"value"}}`),
	}
	config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{
		Header:    "X-Transaction-ID",
		JSONPaths: []string{"$.transaction_id", "$.data.transactionId"},
		Regex:     `transaction_id=([A-Za-z0-9_-]+)`,
	}}

	// When
	_, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if !errors.Is(err, trace.ErrTransactionIDNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
	message := err.Error()
	for _, want := range []string{"cli override --tx", "header X-Transaction-ID", "json path $.transaction_id", "json path $.data.transactionId", "body regex transaction_id=([A-Za-z0-9_-]+)"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected failure to name attempted source %q, got %v", want, err)
		}
	}
}

func TestTransactionIDExtractionFailure_without_config_names_default_attempted_source(t *testing.T) {
	// Given
	response := runner.Response{Body: []byte(`{"transaction_id":"ignored-without-config"}`)}

	// When
	_, err := trace.ExtractTransactionID(trace.ExtractionInput{Response: response})

	// Then
	if !errors.Is(err, trace.ErrTransactionIDNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
	if !strings.Contains(err.Error(), "cli override --tx") || !strings.Contains(err.Error(), "no configured response header") || !strings.Contains(err.Error(), "no configured JSON path") || !strings.Contains(err.Error(), "no configured body regex") {
		t.Fatalf("expected no-config failure to name attempted sources, got %v", err)
	}
}

func TestTransactionIDExtractionFailure_reports_invalid_JSON_path_before_later_sources(t *testing.T) {
	// Given
	response := runner.Response{Body: []byte(`{"message":"transaction_id=regex-tx"}`)}
	config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{
		JSONPaths: []string{"data.transactionId"},
		Regex:     `transaction_id=([A-Za-z0-9_-]+)`,
	}}

	// When
	_, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if !errors.Is(err, trace.ErrInvalidJSONPath) {
		t.Fatalf("expected invalid JSON path error before regex fallback, got %v", err)
	}
	if !strings.Contains(err.Error(), "data.transactionId") {
		t.Fatalf("expected invalid path in error, got %v", err)
	}
}

func TestTransactionIDExtractionFailure_reports_malformed_regex(t *testing.T) {
	// Given
	response := runner.Response{Body: []byte(`transaction_id=regex-tx`)}
	config := model.TraceConfig{TransactionID: model.TransactionIDExtraction{Regex: `transaction_id=([`}}

	// When
	_, err := trace.ExtractTransactionID(trace.ExtractionInput{Config: config, Response: response})

	// Then
	if !errors.Is(err, trace.ErrInvalidRegex) {
		t.Fatalf("expected invalid regex error, got %v", err)
	}
}
