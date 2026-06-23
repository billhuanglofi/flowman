package generate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/billhuanglofi/flowman/internal/model"
)

func TestGeneratedCurlAndHTTPAreDeterministic(t *testing.T) {
	// Given
	request := model.Request{
		Version:  "v1",
		Name:     "create payment",
		Method:   "POST",
		Endpoint: "payments",
		Path:     "/v1/payments",
		Query: []model.Parameter{
			{Name: "source", Value: "flowman-fixture"},
			{Name: "dry_run", Value: "false"},
		},
		Headers: []model.Header{
			{Name: "X-Correlation-ID", Value: "flowman-example-correlation"},
			{Name: "Authorization", Env: "FLOWMAN_PAYMENT_AUTH"},
			{Name: "Content-Type", Value: "application/json"},
		},
		Body: model.RequestBody{Mode: "json", Raw: `{"currency":"USD","amount":"12.34","reference":"example-payment"}`},
	}
	environment := model.Environment{
		Name:    "uat",
		BaseURL: "https://uat.api.example.test",
		Endpoints: []model.EndpointAlias{{
			Name: "payments",
			Path: "/payments",
		}},
	}
	options := Options{Environment: environment}

	// When
	firstCurl, err := RenderCurl(request, options)
	if err != nil {
		t.Fatalf("expected curl projection to render: %v", err)
	}
	secondCurl, err := RenderCurl(request, options)
	if err != nil {
		t.Fatalf("expected second curl projection to render: %v", err)
	}
	firstHTTP, err := RenderHTTP(request, options)
	if err != nil {
		t.Fatalf("expected http projection to render: %v", err)
	}
	secondHTTP, err := RenderHTTP(request, options)
	if err != nil {
		t.Fatalf("expected second http projection to render: %v", err)
	}

	// Then
	if string(firstCurl) != string(secondCurl) {
		t.Fatalf("expected byte-identical curl output")
	}
	if string(firstHTTP) != string(secondHTTP) {
		t.Fatalf("expected byte-identical http output")
	}
	assertMatchesGolden(t, "curl.golden", firstCurl)
	assertMatchesGolden(t, "request.http.golden", firstHTTP)
	for _, output := range []string{string(firstCurl), string(firstHTTP)} {
		if !strings.Contains(output, DoNotEditMarker) {
			t.Fatalf("expected do-not-edit marker in output:\n%s", output)
		}
		if strings.Contains(output, "raw-secret") {
			t.Fatalf("expected output to omit raw secrets:\n%s", output)
		}
		if strings.Contains(output, "Generated at") {
			t.Fatalf("expected deterministic output without timestamps:\n%s", output)
		}
	}
	assertOrdering(t, string(firstCurl), []string{
		"dry_run=false&source=flowman-fixture",
		"Authorization: ${FLOWMAN_PAYMENT_AUTH}",
		"Content-Type: application/json",
		"X-Correlation-ID: flowman-example-correlation",
	})
	assertOrdering(t, string(firstHTTP), []string{
		"dry_run=false&source=flowman-fixture",
		"Authorization: {{$processEnv FLOWMAN_PAYMENT_AUTH}}",
		"Content-Type: application/json",
		"X-Correlation-ID: flowman-example-correlation",
	})
}

func TestRenderCurlAndHTTPRejectRawSensitiveHeaderByDefault(t *testing.T) {
	// Given
	request := model.Request{
		Version: "v1",
		Name:    "raw secret request",
		Method:  "GET",
		URL:     "https://api.example.test/status",
		Headers: []model.Header{{Name: "Authorization", Value: "Bearer raw-secret"}},
	}

	// When
	_, curlErr := RenderCurl(request, Options{})
	_, httpErr := RenderHTTP(request, Options{})

	// Then
	for _, err := range []error{curlErr, httpErr} {
		if !errors.Is(err, ErrRawSensitiveHeader) {
			t.Fatalf("expected raw sensitive header error, got %v", err)
		}
		if strings.Contains(err.Error(), "raw-secret") {
			t.Fatalf("expected error not to echo raw secret, got %v", err)
		}
	}
}

func TestRenderCurlAndHTTPAllowRawSensitiveHeaderWhenUnsafeEnabled(t *testing.T) {
	// Given
	request := model.Request{
		Version: "v1",
		Name:    "raw secret request",
		Method:  "GET",
		URL:     "https://api.example.test/status",
		Headers: []model.Header{{Name: "Authorization", Value: "Bearer raw-secret"}},
	}
	options := Options{UnsafeAllowSensitiveHeaders: true}

	// When
	curlOutput, curlErr := RenderCurl(request, options)
	httpOutput, httpErr := RenderHTTP(request, options)

	// Then
	if curlErr != nil {
		t.Fatalf("expected unsafe curl generation to succeed: %v", curlErr)
	}
	if httpErr != nil {
		t.Fatalf("expected unsafe http generation to succeed: %v", httpErr)
	}
	if !strings.Contains(string(curlOutput), "Bearer raw-secret") {
		t.Fatalf("expected unsafe curl output to include raw secret value")
	}
	if !strings.Contains(string(httpOutput), "Bearer raw-secret") {
		t.Fatalf("expected unsafe http output to include raw secret value")
	}
}

func assertMatchesGolden(t *testing.T, fileName string, got []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", fileName)
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("expected golden file %s to be readable: %v", goldenPath, err)
	}
	if string(got) != string(want) {
		t.Fatalf("expected output to match %s\nwant:\n%s\n\ngot:\n%s", goldenPath, string(want), string(got))
	}
}

func assertOrdering(t *testing.T, output string, orderedSubstrings []string) {
	t.Helper()
	lastIndex := -1
	for _, substring := range orderedSubstrings {
		index := strings.Index(output, substring)
		if index == -1 {
			t.Fatalf("expected %q in output:\n%s", substring, output)
		}
		if index <= lastIndex {
			t.Fatalf("expected %q after previous substring in output:\n%s", substring, output)
		}
		lastIndex = index
	}
}
