//go:build !oracle && !oracle_integration

package oracle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/billhuanglofi/flowman/internal/oracle"
)

func TestOpenSafestoreStore_returnsActionableError_whenOracleSupportDisabled(t *testing.T) {
	// Given
	config := oracle.Config{DSN: "example", User: "user", Password: "secret"}

	// When
	store, closeStore, err := oracle.OpenSafestoreStore(context.Background(), config)

	// Then
	if store != nil {
		t.Fatalf("expected no safestore store without Oracle support, got %#v", store)
	}
	if closeStore != nil {
		t.Fatalf("expected no close function without Oracle support")
	}
	if !errors.Is(err, oracle.ErrSupportNotEnabled) {
		t.Fatalf("expected oracle support error, got %v", err)
	}
	if err.Error() != oracle.SupportNotEnabledMessage {
		t.Fatalf("expected actionable error %q, got %q", oracle.SupportNotEnabledMessage, err.Error())
	}
}
