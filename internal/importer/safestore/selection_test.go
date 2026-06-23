package safestore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func TestImportReplayRequestSelectsFirstWhenRequested(t *testing.T) {
	// Given
	store := fakeStore{rows: multiRowRequests("TX130")}
	options := validOptions(t, "TX130")
	options.Selection = safestore.SelectFirst

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected first selection to succeed: %v", err)
	}
	if result.Request.Body.Raw != "first" {
		t.Fatalf("expected first body, got %q", result.Request.Body.Raw)
	}
}

func TestImportReplayRequestSelectsIndexWhenRequested(t *testing.T) {
	// Given
	store := fakeStore{rows: []safestore.RequestRow{
		{TransactionID: "TX131", Timestamp: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), Headers: `{}`, Body: `first`},
		{TransactionID: "TX131", Timestamp: time.Date(2026, 6, 21, 10, 30, 0, 0, time.UTC), Headers: `{}`, Body: `middle`},
		{TransactionID: "TX131", Timestamp: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC), Headers: `{}`, Body: `latest`},
	}}
	options := validOptions(t, "TX131")
	options.Selection = safestore.Selection("index:1")

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected index selection to succeed: %v", err)
	}
	if result.Request.Body.Raw != "middle" {
		t.Fatalf("expected indexed body, got %q", result.Request.Body.Raw)
	}
}

func TestImportReplayRequestWritesMultipleFilesWhenAllRequested(t *testing.T) {
	// Given
	root := t.TempDir()
	requestPath := filepath.Join(root, "requests", "payment", "replay.request.yaml")
	flowPath := filepath.Join(root, "flows", "payment", "replay.flow.yaml")
	store := fakeStore{rows: multiRowRequests("TX132")}
	options := validOptions(t, "TX132")
	options.All = true
	options.OutputPath = requestPath
	options.FlowOutputPath = flowPath

	// When
	result, err := safestore.ImportReplayRequest(context.Background(), &store, options)

	// Then
	if err != nil {
		t.Fatalf("expected all selection to succeed: %v", err)
	}
	if len(result.Requests) != 2 || len(result.OutputPaths) != 2 || len(result.Flows) != 2 || len(result.FlowPaths) != 2 {
		t.Fatalf("expected multi-file import result, got %#v", result)
	}
	assertExistingFile(t, result.OutputPaths[0])
	assertExistingFile(t, result.OutputPaths[1])
	assertExistingFile(t, result.FlowPaths[0])
	assertExistingFile(t, result.FlowPaths[1])

	firstRequest, err := storage.LoadRequest(result.OutputPaths[0])
	if err != nil {
		t.Fatalf("expected first canonical request to load: %v", err)
	}
	secondRequest, err := storage.LoadRequest(result.OutputPaths[1])
	if err != nil {
		t.Fatalf("expected second canonical request to load: %v", err)
	}
	if firstRequest.Body.Raw != "first" || secondRequest.Body.Raw != "latest" {
		t.Fatalf("expected all selection to preserve row ordering, got first=%q second=%q", firstRequest.Body.Raw, secondRequest.Body.Raw)
	}
}

func multiRowRequests(transactionID string) []safestore.RequestRow {
	return []safestore.RequestRow{
		{TransactionID: transactionID, Timestamp: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), Headers: `{}`, Body: `first`},
		{TransactionID: transactionID, Timestamp: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC), Headers: `{}`, Body: `latest`},
	}
}

func assertExistingFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}
