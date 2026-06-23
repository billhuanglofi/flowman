package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func Test_RootCommand_help_lists_placeholder_groups(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	err := Execute(ctx, []string{"--help"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected help to succeed: %v", err)
	}
	help := stdout.String()
	for _, commandName := range []string{"config", "request", "trace", "import", "export", "generate", "tui"} {
		if !strings.Contains(help, commandName) {
			t.Fatalf("expected help to list %q, got:\n%s", commandName, help)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func Test_RootCommand_version_prints_non_empty_version(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	err := Execute(ctx, []string{"version"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected version to succeed: %v", err)
	}
	output := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(output, "flowman ") || output == "flowman" {
		t.Fatalf("expected non-empty version output, got %q", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func Test_RootCommand_unknown_returns_error_and_help_without_panic(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	err := Execute(ctx, []string{"unknown"}, stdout, stderr)

	// Then
	if err == nil {
		t.Fatal("expected unknown command to return an error")
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "Usage:") {
		t.Fatalf("expected help usage for unknown command, got:\n%s", combined)
	}
	if strings.Contains(strings.ToLower(combined), "panic") {
		t.Fatalf("expected output without panic, got:\n%s", combined)
	}
}
