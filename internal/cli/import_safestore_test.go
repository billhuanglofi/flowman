package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/oracle"
)

func Test_ImportSafestoreCommand_writes_redacted_json_report(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	outputPath := filepath.Join(t.TempDir(), "replay.request.yaml")
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", "secret-dsn")
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", "secret-user")
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", "secret-password")
	openCapture := safestoreOpenCapture{}
	importCapture := safestoreImportCapture{}
	restoreOpen := stubSafestoreStoreOpener(func(_ context.Context, cfg oracle.EnvConfig) (oracle.SafestoreStore, func() error, error) {
		openCapture.config = cfg
		return fakeSafestoreStore{}, func() error {
			openCapture.closed = true
			return nil
		}, nil
	})
	defer restoreOpen()
	restoreImport := stubSafestoreImporter(func(_ context.Context, _ safestore.RequestStore, options safestore.ImportOptions) (safestore.ImportResult, error) {
		importCapture.options = options
		if err := os.WriteFile(options.OutputPath, []byte("version: v1\nname: replay-TX-123\n"), 0o644); err != nil {
			return safestore.ImportResult{}, err
		}
		return safestore.ImportResult{
			Request: model.Request{
				Name:     "replay-TX-123",
				Method:   "POST",
				Endpoint: "payments",
				Path:     "/v1/payments",
				ImportWarnings: []model.ImportWarning{{
					Code:    "sensitive-header-env-reference",
					Path:    "headers.Authorization",
					Message: "Imported sensitive header \"Authorization\" as env reference FLOWMAN_IMPORTED_TX_123_AUTHORIZATION; raw value was not written.",
				}},
			},
			OutputPath: options.OutputPath,
		}, nil
	})
	defer restoreImport()

	// When
	err := Execute(ctx, []string{"import", "safestore", "--tx", "TX-123", "--env", "uat", "--endpoint", "payments", "--path", "/v1/payments", "--out", outputPath, "--non-interactive", "--config", "../../testdata/flowman/flowman.yaml"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected import safestore to succeed: %v", err)
	}
	if !openCapture.closed {
		t.Fatal("expected Safestore store closer to run")
	}
	if importCapture.options.TransactionID != "TX-123" || !importCapture.options.NonInteractive {
		t.Fatalf("expected CLI flags forwarded to importer, got %#v", importCapture.options)
	}
	wantConfig := oracle.EnvConfig{
		DSNEnv:      "FLOWMAN_UAT_ORACLE_DSN",
		UserEnv:     "FLOWMAN_UAT_ORACLE_USER",
		PasswordEnv: "FLOWMAN_UAT_ORACLE_PASSWORD",
	}
	if !reflect.DeepEqual(openCapture.config, wantConfig) {
		t.Fatalf("expected oracle env-name config, got %#v", openCapture.config)
	}
	var report safestoreImportReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected JSON report, got %q: %v", stdout.String(), err)
	}
	if report.TransactionID != "TX-123" || report.Method != "POST" || report.OutputPath != outputPath {
		t.Fatalf("expected import report metadata, got %#v", report)
	}
	combined := stdout.String() + stderr.String()
	for _, secret := range []string{"secret-dsn", "secret-user", "secret-password"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("expected secret %q to stay out of import output, got %q", secret, combined)
		}
	}
}

func Test_ImportSafestoreCommand_noninteractive_without_url_input_exits_nonzero_without_prompt(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	t.Setenv("FLOWMAN_UAT_ORACLE_DSN", "secret-dsn")
	t.Setenv("FLOWMAN_UAT_ORACLE_USER", "secret-user")
	t.Setenv("FLOWMAN_UAT_ORACLE_PASSWORD", "secret-password")
	restoreOpen := stubSafestoreStoreOpener(func(_ context.Context, _ oracle.EnvConfig) (oracle.SafestoreStore, func() error, error) {
		return fakeSafestoreStore{}, func() error { return nil }, nil
	})
	defer restoreOpen()
	restoreImport := stubSafestoreImporter(func(_ context.Context, _ safestore.RequestStore, options safestore.ImportOptions) (safestore.ImportResult, error) {
		return safestore.ImportReplayRequest(context.Background(), fakeSafestoreStore{rows: []safestore.RequestRow{{TransactionID: options.TransactionID, Headers: `{}`, Body: `{}`}}}, options)
	})
	defer restoreImport()

	// When
	err := Execute(ctx, []string{"import", "safestore", "--tx", "TX-999", "--env", "uat", "--out", filepath.Join(t.TempDir(), "replay.request.yaml"), "--non-interactive", "--config", "../../testdata/flowman/flowman.yaml"}, stdout, stderr)

	// Then
	if err == nil {
		t.Fatal("expected non-interactive import without URL input to fail")
	}
	if !strings.Contains(err.Error(), "--non-interactive") {
		t.Fatalf("expected actionable non-interactive error, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no success report on failure, got %q", stdout.String())
	}
	combined := stdout.String() + stderr.String() + err.Error()
	if strings.Contains(strings.ToLower(combined), "prompt") || strings.Contains(strings.ToLower(combined), "enter ") {
		t.Fatalf("expected failure without prompt/hang wording, got %q", combined)
	}
	if ctx.Err() != nil {
		t.Fatalf("expected command to fail before context timeout, got %v", ctx.Err())
	}
}

type safestoreOpenCapture struct {
	config oracle.EnvConfig
	closed bool
}

type safestoreImportCapture struct {
	options safestore.ImportOptions
}

type fakeSafestoreStore struct {
	rows []safestore.RequestRow
}

func (store fakeSafestoreStore) RequestRows(_ context.Context, _ string) ([]safestore.RequestRow, error) {
	return store.rows, nil
}

func stubSafestoreStoreOpener(opener func(context.Context, oracle.EnvConfig) (oracle.SafestoreStore, func() error, error)) func() {
	previous := openSafestoreStore
	openSafestoreStore = opener
	return func() {
		openSafestoreStore = previous
	}
}

func stubSafestoreImporter(importer func(context.Context, safestore.RequestStore, safestore.ImportOptions) (safestore.ImportResult, error)) func() {
	previous := importSafestoreReplayRequest
	importSafestoreReplayRequest = importer
	return func() {
		importSafestoreReplayRequest = previous
	}
}
