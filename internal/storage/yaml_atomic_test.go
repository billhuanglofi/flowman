package storage_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func Test_WriteCanonical_uses_atomic_write_rename(t *testing.T) {
	// Given
	tmp := t.TempDir()
	path := filepath.Join(tmp, "atomic-test.yaml")

	request := model.Request{
		Version: "v1",
		Name:    "test-request",
		Method:  "GET",
		URL:     "https://api.example.test/status",
	}

	// When
	err := storage.WriteCanonical(path, request)

	// Then
	if err != nil {
		t.Fatalf("expected atomic write to succeed: %v", err)
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "atomic-test.yaml" {
			t.Errorf("expected no temp files, found: %s", entry.Name())
		}
	}

	// Verify content is correct
	loaded, err := storage.LoadRequest(path)
	if err != nil {
		t.Fatalf("expected to load written file: %v", err)
	}
	if loaded.Name != "test-request" {
		t.Errorf("expected loaded name to match, got %q", loaded.Name)
	}
}

func Test_WriteCanonical_handles_concurrent_writes_safely(t *testing.T) {
	// Given
	tmp := t.TempDir()
	path := filepath.Join(tmp, "concurrent.yaml")

	// When - multiple goroutines write concurrently
	const writers = 10
	var wg sync.WaitGroup
	wg.Add(writers)

	successCount := 0
	var mu sync.Mutex

	for i := 0; i < writers; i++ {
		i := i
		go func() {
			defer wg.Done()
			request := model.Request{
				Version: "v1",
				Name:    string(rune('a' + i)), // Different names to detect corruption
				Method:  "GET",
				URL:     "https://api.example.test/status",
			}
			if err := storage.WriteCanonical(path, request); err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Then - at least some writes should succeed (race is expected, not all will win)
	if successCount == 0 {
		t.Fatal("expected at least one write to succeed")
	}

	// Then - final file is valid (not corrupted by partial writes)
	loaded, err := storage.LoadRequest(path)
	if err != nil {
		t.Fatalf("expected final file to be valid after concurrent writes: %v", err)
	}

	// Final name should be one of the writers (a-j)
	if len(loaded.Name) != 1 || loaded.Name[0] < 'a' || loaded.Name[0] > 'j' {
		t.Errorf("expected valid name from one of the writers, got %q", loaded.Name)
	}

	// Verify no temp files leaked
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "concurrent.yaml" {
			t.Errorf("expected no temp files after concurrent writes, found: %s", entry.Name())
		}
	}
}

func Test_WriteCanonical_cleans_up_temp_file_on_rename_failure(t *testing.T) {
	// Given
	tmp := t.TempDir()

	// Create a directory where the file should be (causes rename to fail)
	badPath := filepath.Join(tmp, "subdir", "file.yaml")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("failed to create conflicting directory: %v", err)
	}

	request := model.Request{
		Version: "v1",
		Name:    "test",
		Method:  "GET",
		URL:     "https://api.example.test/status",
	}

	// When
	err := storage.WriteCanonical(badPath, request)

	// Then - operation should fail
	if err == nil {
		t.Fatal("expected write to fail when target is a directory")
	}

	// Then - no temp files should be left behind
	entries, err := os.ReadDir(filepath.Join(tmp, "subdir"))
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("expected temp file to be cleaned up, found: %s", entry.Name())
		}
	}
}
