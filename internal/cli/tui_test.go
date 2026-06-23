package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func Test_TUICommand_help_exits_zero(t *testing.T) {
	// Given
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// When
	err := Execute(ctx, []string{"tui", "--help"}, stdout, stderr)

	// Then
	if err != nil {
		t.Fatalf("expected tui help to succeed: %v", err)
	}
	help := stdout.String()
	for _, expected := range []string{"Launch the Flowman preview TUI", "--config", "--env"} {
		if !strings.Contains(help, expected) {
			t.Fatalf("expected tui help to contain %q, got:\n%s", expected, help)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}
