package postman_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/billhuanglofi/flowman/internal/importer/postman"
	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestPostmanSupportedSubsetAndStrictWarnings(t *testing.T) {
	// Given
	root := t.TempDir()
	collectionPath := filepath.Join(root, "payments.postman_collection.json")
	if err := os.WriteFile(collectionPath, []byte(postmanCollectionFixture()), 0o600); err != nil {
		t.Fatalf("expected collection fixture write to succeed: %v", err)
	}
	reportPath := filepath.Join(root, "report.json")
	defaultOutputDir := filepath.Join(root, "default")
	strictOutputDir := filepath.Join(root, "strict")

	// When
	defaultResult, defaultErr := postman.ImportCollection(context.Background(), postman.ImportOptions{
		CollectionPath: collectionPath,
		OutputDir:      defaultOutputDir,
		ReportPath:     reportPath,
	})
	strictResult, strictErr := postman.ImportCollection(context.Background(), postman.ImportOptions{
		CollectionPath: collectionPath,
		OutputDir:      strictOutputDir,
		ReportPath:     reportPath,
		Strict:         true,
	})

	// Then
	if defaultErr != nil {
		t.Fatalf("expected supported subset import to succeed: %v", defaultErr)
	}
	if strictResult.ImportedCount != 0 {
		t.Fatalf("expected strict import to avoid writes on warnings, got %#v", strictResult)
	}
	if !errors.Is(strictErr, postman.ErrStrictWarnings) {
		t.Fatalf("expected strict warning failure, got %v", strictErr)
	}
	if defaultResult.ImportedCount != 2 {
		t.Fatalf("expected two requests imported, got %#v", defaultResult)
	}
	if len(defaultResult.Warnings) < 5 {
		t.Fatalf("expected unsupported-field warnings, got %#v", defaultResult.Warnings)
	}
	assertWarningsContain(t, defaultResult.WarningText(), []string{
		"$.variable[0]",
		"$.item[0].item[0].event[0].script.exec",
		"$.item[0].item[0].request.body.graphql",
		"$.item[0].item[1].request.certificate",
		"$.item[0].item[1].request.body.formdata[0].src",
	})

	createRequestPath := filepath.Join(defaultOutputDir, "requests", "payments", "create-payment.request.yaml")
	createFlowPath := filepath.Join(defaultOutputDir, "flows", "payments", "create-payment.flow.yaml")
	createRequest, err := storage.LoadRequest(createRequestPath)
	if err != nil {
		t.Fatalf("expected canonical request to load: %v", err)
	}
	if createRequest.Name != "Create Payment" || createRequest.Method != "POST" || createRequest.Path != "/payments" {
		t.Fatalf("expected request metadata imported, got %#v", createRequest)
	}
	if len(createRequest.Query) != 2 || createRequest.Query[0].Name != "dry_run" || createRequest.Query[0].Value != "false" {
		t.Fatalf("expected ordered query params, got %#v", createRequest.Query)
	}
	if createRequest.Body.Mode != "json" || !strings.Contains(createRequest.Body.Raw, "amount") {
		t.Fatalf("expected JSON raw body preserved, got %#v", createRequest.Body)
	}
	if len(createRequest.ImportWarnings) == 0 {
		t.Fatalf("expected request warnings to be preserved in canonical request, got %#v", createRequest)
	}
	assertHeaderEnv(t, createRequestPath, createRequest.Headers, "Authorization", "FLOWMAN_POSTMAN_PAYMENTS_CREATE_PAYMENT_AUTHORIZATION")

	createFlow, err := storage.LoadFlow(createFlowPath)
	if err != nil {
		t.Fatalf("expected canonical flow to load: %v", err)
	}
	if createFlow.Name != "Create Payment" || len(createFlow.Steps) != 1 || createFlow.Steps[0].Request != "requests/payments/create-payment.request.yaml" {
		t.Fatalf("expected request-backed flow output, got %#v", createFlow)
	}

	statusRequestPath := filepath.Join(defaultOutputDir, "requests", "payments", "get-status.request.yaml")
	statusRequest, err := storage.LoadRequest(statusRequestPath)
	if err != nil {
		t.Fatalf("expected second canonical request to load: %v", err)
	}
	if statusRequest.Method != "GET" || statusRequest.URL != "https://api.example.test/status" {
		t.Fatalf("expected GET request URL import, got %#v", statusRequest)
	}
	assertHeaderEnv(t, statusRequestPath, statusRequest.Headers, "X-API-Key", "FLOWMAN_POSTMAN_PAYMENTS_GET_STATUS_X_API_KEY")

	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected warning report to be written: %v", err)
	}
	assertWarningsContain(t, string(reportContent), []string{
		"$.variable[0]",
		"$.item[0].item[0].event[0].script.exec",
		"$.item[0].item[1].request.certificate",
	})
	if strings.Contains(string(reportContent), "Bearer secret-token") || strings.Contains(string(reportContent), "top-secret") {
		t.Fatalf("expected report to stay redacted, got %s", string(reportContent))
	}
	if _, err := os.Stat(filepath.Join(strictOutputDir, "requests")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected strict import to avoid request writes, stat err=%v", err)
	}
}

func assertWarningsContain(t *testing.T, content string, want []string) {
	t.Helper()
	for _, needle := range want {
		if !strings.Contains(content, needle) {
			t.Fatalf("expected %q in warnings/report, got %s", needle, content)
		}
	}
}

func assertHeaderEnv(t *testing.T, path string, headers []model.Header, name string, want string) {
	t.Helper()
	for _, header := range headers {
		if header.Name == name {
			if header.Env != want || header.Value != "" {
				t.Fatalf("expected %s env reference %q in %s, got %#v", name, want, path, header)
			}
			return
		}
	}
	t.Fatalf("expected header %s in %s", name, path)
}

func postmanCollectionFixture() string {
	return `
{
  "info": {
    "name": "Payments API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {
      "key": "baseUrl",
      "value": "https://api.example.test"
    }
  ],
  "item": [
    {
      "name": "Payments",
      "item": [
        {
          "name": "Create Payment",
          "event": [
            {
              "listen": "prerequest",
              "script": {
                "exec": ["pm.environment.set('x', 'y');"]
              }
            }
          ],
          "request": {
            "method": "POST",
            "header": [
              {"key": "Content-Type", "value": "application/json"},
              {"key": "Authorization", "value": "Bearer secret-token"}
            ],
            "auth": {
              "type": "bearer",
              "bearer": [{"key": "token", "value": "top-secret"}]
            },
            "url": {
              "raw": "https://api.example.test/payments?dry_run=false&source=postman",
              "protocol": "https",
              "host": ["api", "example", "test"],
              "path": ["payments"],
              "query": [
                {"key": "dry_run", "value": "false"},
                {"key": "source", "value": "postman"}
              ]
            },
            "body": {
              "mode": "raw",
              "raw": "{\n  \"amount\": \"12.34\"\n}",
              "options": {
                "raw": {"language": "json"}
              },
              "graphql": {
                "query": "query Unsupported { payments }"
              }
            }
          }
        },
        {
          "name": "Get Status",
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": ["pm.test('status', function () {});"]
              }
            }
          ],
          "request": {
            "method": "GET",
            "certificate": {"name": "client.pem"},
            "auth": {
              "type": "apikey",
              "apikey": [
                {"key": "key", "value": "X-API-Key"},
                {"key": "value", "value": "top-secret-key"},
                {"key": "in", "value": "header"}
              ]
            },
            "url": "https://api.example.test/status",
            "body": {
              "mode": "formdata",
              "formdata": [
                {"key": "file", "type": "file", "src": "/tmp/secret.txt"}
              ]
            }
          }
        }
      ]
    }
  ]
}
`
}
