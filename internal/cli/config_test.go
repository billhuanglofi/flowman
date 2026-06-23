package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func Test_ConfigValidateCommand_prints_ok_messages_for_fixture_env(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	configPath := filepath.Join("..", "..", "testdata", "flowman", "flowman.yaml")

	// When
	err := Execute(ctx, []string{"config", "validate", "--config", configPath, "--env", "uat"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected config validate to succeed: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"OK: config valid", "OK: secrets referenced by environment variable name only"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func Test_ConfigValidateCommand_returns_error_for_missing_env(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	configPath := filepath.Join("..", "..", "testdata", "flowman", "flowman.yaml")

	// When
	err := Execute(ctx, []string{"config", "validate", "--config", configPath, "--env", "missing"}, stdout, stderr)

	// Then
	if err == nil {
		t.Fatal("expected config validate to fail for missing env")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing env name in error, got %v", err)
	}
	if strings.Contains(strings.ToLower(stdout.String()+stderr.String()+err.Error()), "panic") {
		t.Fatalf("expected failure without panic, got stdout=%q stderr=%q err=%v", stdout.String(), stderr.String(), err)
	}
}

func Test_ConfigValidateCommand_check_secrets_uses_presence_only(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	configPath := filepath.Join("..", "..", "testdata", "flowman", "flowman.yaml")
	secretValue := strings.Repeat("x", 8)
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", secretValue)

	// When
	err := Execute(ctx, []string{"config", "validate", "--config", configPath, "--env", "uat", "--check-secrets"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected check-secrets to succeed when env vars are present: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "OK: checked secret env var presence") {
		t.Fatalf("expected check-secrets OK output, got:\n%s", output)
	}
	if strings.Contains(output, secretValue) || strings.Contains(stderr.String(), secretValue) {
		t.Fatal("expected command output not to include secret value")
	}
}

func Test_ConfigValidateCommand_redacts_raw_sensitive_header_error(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tmp := t.TempDir()
	secretValue := strings.Repeat("x", 8)
	request := model.Request{
		Version: "v1",
		Name:    "raw secret request",
		Method:  "GET",
		URL:     "https://api.example.test/status",
		Headers: []model.Header{{Name: "Authorization", Value: secretValue}},
	}
	writeWorkspaceFixture(t, tmp, request)

	// When
	err := Execute(ctx, []string{"config", "validate", "--config", filepath.Join(tmp, "flowman.yaml"), "--env", "uat"}, stdout, stderr)

	// Then
	if err == nil {
		t.Fatal("expected raw sensitive header to fail validation")
	}
	combined := stdout.String() + stderr.String() + err.Error()
	if strings.Contains(combined, secretValue) {
		t.Fatalf("expected redacted output without raw sensitive value, got %q", combined)
	}
	if !strings.Contains(combined, "[REDACTED]") {
		t.Fatalf("expected redacted marker in failure, got %q", combined)
	}
}

func writeWorkspaceFixture(t *testing.T, root string, request model.Request) {
	t.Helper()
	project := model.Project{Version: "v1", Name: "fixture", DefaultEnv: "uat", Environments: []string{"envs/uat.yaml"}, Requests: []string{"requests/raw.request.yaml"}}
	environment := model.Environment{
		Version: "v1",
		Name:    "uat",
		BaseURL: "https://uat.api.example.test",
		Oracle: model.OracleDatabase{
			DSNEnv:      "FLOWMAN_UAT_ORACLE_DSN",
			UserEnv:     "FLOWMAN_UAT_ORACLE_USER",
			PasswordEnv: "FLOWMAN_UAT_ORACLE_PASSWORD",
		},
		Trace: model.TraceConfig{
			TransactionID: model.TransactionIDExtraction{Header: "X-Transaction-ID"},
			PollInterval:  model.NewDuration(2 * time.Second),
			Timeout:       model.NewDuration(30 * time.Second),
		},
	}
	if err := storage.WriteCanonical(filepath.Join(root, "flowman.yaml"), project); err != nil {
		t.Fatalf("expected project fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "envs", "uat.yaml"), environment); err != nil {
		t.Fatalf("expected environment fixture write to succeed: %v", err)
	}
	if err := storage.WriteCanonical(filepath.Join(root, "requests", "raw.request.yaml"), request); err != nil {
		t.Fatalf("expected request fixture write to succeed: %v", err)
	}
}
