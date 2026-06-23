package insomnia_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/billhuanglofi/flowman/internal/importer/insomnia"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestInsomniaSupportedSubsetAndStrictWarnings(t *testing.T) {
	// Given
	root := t.TempDir()
	documentPath := filepath.Join(root, "payments.insomnia.json")
	if err := os.WriteFile(documentPath, []byte(insomniaFixture()), 0o600); err != nil {
		t.Fatalf("expected insomnia fixture write to succeed: %v", err)
	}
	reportPath := filepath.Join(root, "report.json")
	defaultOutputDir := filepath.Join(root, "default")
	strictOutputDir := filepath.Join(root, "strict")

	// When
	defaultResult, defaultErr := insomnia.ImportWorkspace(context.Background(), insomnia.ImportOptions{
		DocumentPath: documentPath,
		OutputDir:    defaultOutputDir,
		ReportPath:   reportPath,
	})
	strictResult, strictErr := insomnia.ImportWorkspace(context.Background(), insomnia.ImportOptions{
		DocumentPath: documentPath,
		OutputDir:    strictOutputDir,
		ReportPath:   reportPath,
		Strict:       true,
	})

	// Then
	if defaultErr != nil {
		t.Fatalf("expected insomnia supported subset import to succeed: %v", defaultErr)
	}
	if !errors.Is(strictErr, insomnia.ErrStrictWarnings) {
		t.Fatalf("expected strict warning failure, got %v", strictErr)
	}
	if strictResult.ImportedCount != 0 {
		t.Fatalf("expected strict import to avoid writes on warnings, got %#v", strictResult)
	}
	if defaultResult.ImportedCount != 2 {
		t.Fatalf("expected two requests imported, got %#v", defaultResult)
	}
	assertContainsAll(t, defaultResult.WarningText(), []string{
		"$.resources[3].body.params[0].fileName",
		"$.resources[3].clientCertificates[0]",
		"$.resources[3].authentication",
		"$.resources[4].cookieJar",
	})
	request, err := storage.LoadRequest(filepath.Join(defaultOutputDir, "requests", "payments", "create-payment.request.yaml"))
	if err != nil {
		t.Fatalf("expected imported request to load: %v", err)
	}
	if request.Name != "Create Payment" || request.Method != "POST" || request.Path != "/payments" {
		t.Fatalf("expected supported request metadata, got %#v", request)
	}
	if request.Body.Mode != "json" || !strings.Contains(request.Body.Raw, "amount") {
		t.Fatalf("expected JSON body preserved, got %#v", request.Body)
	}
	if _, err := os.Stat(filepath.Join(strictOutputDir, "requests")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected strict import to avoid request writes, stat err=%v", err)
	}
	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected warning report to be written: %v", err)
	}
	assertContainsAll(t, string(reportContent), []string{
		"$.resources[3].clientCertificates[0]",
		"$.resources[4].cookieJar",
	})
	if strings.Contains(string(reportContent), "Bearer secret-token") {
		t.Fatalf("expected report to stay redacted, got %s", string(reportContent))
	}
}

func assertContainsAll(t *testing.T, content string, want []string) {
	t.Helper()
	for _, needle := range want {
		if !strings.Contains(content, needle) {
			t.Fatalf("expected %q in content, got %s", needle, content)
		}
	}
}

func insomniaFixture() string {
	return `
{
  "_type": "export",
  "__export_format": 4,
  "resources": [
    {
      "_id": "wrk_1",
      "_type": "workspace",
      "name": "Payments API"
    },
    {
      "_id": "fld_1",
      "_type": "request_group",
      "parentId": "wrk_1",
      "name": "Payments"
    },
    {
      "_id": "req_1",
      "_type": "request",
      "parentId": "fld_1",
      "name": "Create Payment",
      "method": "POST",
      "url": "https://api.example.test/payments",
      "parameters": [
        {"name": "dry_run", "value": "false"},
        {"name": "source", "value": "insomnia"}
      ],
      "headers": [
        {"name": "Content-Type", "value": "application/json"},
        {"name": "Authorization", "value": "Bearer secret-token"}
      ],
      "body": {
        "mimeType": "application/json",
        "text": "{\n  \"amount\": \"12.34\"\n}"
      }
    },
    {
      "_id": "req_2",
      "_type": "request",
      "parentId": "fld_1",
      "name": "Get Status",
      "method": "GET",
      "url": "https://api.example.test/status",
      "headers": [
        {"name": "X-API-Key", "value": "top-secret-key"}
      ],
      "body": {
        "mimeType": "multipart/form-data",
        "params": [
          {"name": "file", "fileName": "/tmp/secret.txt"}
        ]
      },
      "clientCertificates": [
        {"certificate": "client.pem"}
      ],
      "authentication": {
        "type": "bearer"
      }
    },
    {
      "_id": "req_3",
      "_type": "request",
      "parentId": "fld_1",
      "name": "Cookie Jar Request",
      "method": "GET",
      "url": "https://api.example.test/cookies",
      "cookieJar": {
        "cookies": []
      }
    }
  ]
}
`
}
