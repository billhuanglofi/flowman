package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/config"
	"github.com/billhuanglofi/flowman/internal/model"
)

func Test_ValidateWorkspace_accepts_fixture_env_without_reading_secret_values(t *testing.T) {
	// Given
	root := filepath.Join("..", "..", "testdata", "flowman")
	workspace, err := config.LoadWorkspaceFromConfig(filepath.Join(root, "flowman.yaml"))
	if err != nil {
		t.Fatalf("expected workspace load to succeed: %v", err)
	}
	secretValue := strings.Repeat("x", 8)
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", secretValue)

	// When
	result, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "uat"})

	// Then
	if err != nil {
		t.Fatalf("expected validation to succeed without checking secret presence: %v", err)
	}
	if result.Environment.Name != "uat" {
		t.Fatalf("expected selected uat env, got %q", result.Environment.Name)
	}
	if len(result.SecretEnvNames) != 3 {
		t.Fatalf("expected oracle secret env names, got %#v", result.SecretEnvNames)
	}
	if strings.Contains(strings.Join(result.Messages, "\n"), secretValue) {
		t.Fatal("expected validation messages not to include secret values")
	}
}

func Test_ValidateWorkspace_rejects_missing_env_selection(t *testing.T) {
	// Given
	root := filepath.Join("..", "..", "testdata", "flowman")
	workspace, err := config.LoadWorkspaceFromConfig(filepath.Join(root, "flowman.yaml"))
	if err != nil {
		t.Fatalf("expected workspace load to succeed: %v", err)
	}

	// When
	_, err = config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "missing"})

	// Then
	if !errors.Is(err, config.ErrEnvironmentNotFound) {
		t.Fatalf("expected environment not found error, got %v", err)
	}
	if strings.Contains(err.Error(), strings.Repeat("x", 8)) {
		t.Fatalf("expected error not to include secret values, got %v", err)
	}
}

func Test_ValidateWorkspace_rejects_raw_sensitive_header_and_redacts_error(t *testing.T) {
	// Given
	workspace := validWorkspace()
	secretValue := strings.Repeat("x", 8)
	workspace.Requests[0].Headers = append(workspace.Requests[0].Headers, model.Header{Name: "authorization", Value: secretValue})

	// When
	_, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "uat"})

	// Then
	if !errors.Is(err, config.ErrRawSensitiveHeader) {
		t.Fatalf("expected raw sensitive header error, got %v", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatalf("expected redacted error without raw header value, got %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("expected redacted marker in error, got %v", err)
	}
}

func Test_ValidateWorkspace_accepts_sensitive_header_env_reference(t *testing.T) {
	// Given
	workspace := validWorkspace()
	workspace.Requests[0].Headers = append(workspace.Requests[0].Headers, model.Header{Name: "X-API-Key", Env: "FLOWMAN_UAT_API_KEY"})

	// When
	result, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "uat"})

	// Then
	if err != nil {
		t.Fatalf("expected sensitive header env reference to validate: %v", err)
	}
	if !containsString(result.SecretEnvNames, "FLOWMAN_UAT_API_KEY") {
		t.Fatalf("expected sensitive header env name recorded, got %#v", result.SecretEnvNames)
	}
}

func Test_ValidateWorkspace_check_secrets_reports_missing_env_without_values(t *testing.T) {
	// Given
	workspace := validWorkspace()
	workspace.Requests[0].Headers = append(workspace.Requests[0].Headers, model.Header{Name: "Cookie", Env: "FLOWMAN_UAT_COOKIE"})
	secretValue := strings.Repeat("x", 8)
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", secretValue)

	// When
	_, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "uat", CheckSecrets: true})

	// Then
	if !errors.Is(err, config.ErrMissingSecretEnv) {
		t.Fatalf("expected missing secret env error, got %v", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatalf("expected check-secrets error not to print secret value, got %v", err)
	}
	if !strings.Contains(err.Error(), "FLOWMAN_UAT_COOKIE") {
		t.Fatalf("expected missing env var name in error, got %v", err)
	}
}

func Test_ValidateWorkspace_check_secrets_accepts_present_env_names_without_values(t *testing.T) {
	// Given
	workspace := validWorkspace()
	workspace.Requests[0].Headers = append(workspace.Requests[0].Headers, model.Header{Name: "Set-Cookie", Env: "FLOWMAN_UAT_SET_COOKIE"})
	secretValue := strings.Repeat("x", 8)
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", secretValue)
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", secretValue)
	t.Setenv("FLOWMAN_UAT_SET_COOKIE", secretValue)

	// When
	result, err := config.ValidateWorkspace(workspace, config.ValidationOptions{EnvName: "uat", CheckSecrets: true})

	// Then
	if err != nil {
		t.Fatalf("expected present secret env vars to validate: %v", err)
	}
	for _, message := range result.Messages {
		if strings.Contains(message, secretValue) {
			t.Fatalf("expected messages not to print secret values, got %q", message)
		}
	}
}

func Test_LoadWorkspaceFromConfig_uses_config_file_parent_directory(t *testing.T) {
	// Given
	root := filepath.Join("..", "..", "testdata", "flowman")

	// When
	workspace, err := config.LoadWorkspaceFromConfig(filepath.Join(root, "flowman.yaml"))

	// Then
	if err != nil {
		t.Fatalf("expected workspace load to succeed: %v", err)
	}
	if workspace.Project.Name != "flowman-example" || len(workspace.Environments) != 2 {
		t.Fatalf("expected fixture workspace, got %#v", workspace.Project)
	}
}

func Test_LoadWorkspaceFromConfig_returns_error_when_exact_config_path_missing(t *testing.T) {
	// Given
	root := filepath.Join("..", "..", "testdata", "flowman")
	path := filepath.Join(root, "not-flowman.yaml")

	// When
	_, err := config.LoadWorkspaceFromConfig(path)

	// Then
	if err == nil {
		t.Fatal("expected missing exact config path to fail")
	}
	if !strings.Contains(err.Error(), "not-flowman.yaml") {
		t.Fatalf("expected error to name exact missing config path, got %v", err)
	}
}

func Test_LoadWorkspaceFromConfig_loads_custom_config_basename(t *testing.T) {
	// Given
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "custom.yaml"), "version: v1\nname: custom\ndefault_env: uat\nenvironments:\n  - envs/uat.yaml\nrequests:\n  - requests/status.request.yaml\nflows:\n  - flows/status.flow.yaml\n")
	writeFile(t, filepath.Join(root, "envs", "uat.yaml"), "version: v1\nname: uat\nbase_url: https://uat.api.example.test\noracle:\n  dsn_env: FLOWMAN_UAT_ORACLE_DSN\n  user_env: FLOWMAN_UAT_ORACLE_USER\n  password_env: FLOWMAN_UAT_ORACLE_PASSWORD\ntrace:\n  transaction_id:\n    header: X-Transaction-ID\n  poll_interval: 2s\n  timeout: 30s\n")
	writeFile(t, filepath.Join(root, "requests", "status.request.yaml"), "version: v1\nname: status\nmethod: GET\nurl: https://api.example.test/status\n")
	writeFile(t, filepath.Join(root, "flows", "status.flow.yaml"), "version: v1\nname: status flow\nsteps:\n  - name: status\n    request: requests/status.request.yaml\n")

	// When
	workspace, err := config.LoadWorkspaceFromConfig(filepath.Join(root, "custom.yaml"))

	// Then
	if err != nil {
		t.Fatalf("expected custom config basename to load: %v", err)
	}
	if workspace.Project.Name != "custom" || workspace.Environments[0].Name != "uat" || workspace.Requests[0].Name != "status" {
		t.Fatalf("expected custom workspace relatives to load, got %#v", workspace)
	}
}

func Test_LoadWorkspaceFromConfig_wraps_malformed_yaml_path(t *testing.T) {
	// Given
	tmp := t.TempDir()
	path := filepath.Join(tmp, "flowman.yaml")
	if err := os.WriteFile(path, []byte("version: [unterminated\n"), 0o600); err != nil {
		t.Fatalf("expected malformed fixture write to succeed: %v", err)
	}

	// When
	_, err := config.LoadWorkspaceFromConfig(path)

	// Then
	if err == nil || !strings.Contains(err.Error(), "flowman.yaml") {
		t.Fatalf("expected actionable malformed YAML error, got %v", err)
	}
}

func validWorkspace() model.Workspace {
	return model.Workspace{
		Project: model.Project{Version: "v1", Name: "fixture", DefaultEnv: "uat", Environments: []string{"envs/uat.yaml"}},
		Environments: []model.Environment{{
			Version: "v1",
			Name:    "uat",
			BaseURL: "https://uat.api.example.test",
			Endpoints: []model.EndpointAlias{{
				Name: "payments",
				Path: "/payments",
			}},
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
		}},
		Requests: []model.Request{{
			Version:  "v1",
			Name:     "create payment",
			Method:   "POST",
			Endpoint: "payments",
			Headers:  []model.Header{{Name: "Content-Type", Value: "application/json"}},
		}},
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("expected parent directory write to succeed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("expected fixture write to succeed: %v", err)
	}
}
