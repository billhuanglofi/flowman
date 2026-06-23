//go:build oracle || oracle_integration

package oracle_test

import (
	"strings"
	"testing"

	"github.com/billhuanglofi/flowman/internal/oracle"
	"github.com/billhuanglofi/flowman/internal/trace"
)

func TestOracleSQL_usesNamedBindParameters_whenOracleAdapterEnabled(t *testing.T) {
	// Given
	maliciousTransactionID := "TX123' OR '1'='1"

	// When
	processStateSQL := trace.ProcessStateSQL
	safestoreSQL := oracle.SafestoreRequestSQL

	// Then
	if strings.Contains(processStateSQL, maliciousTransactionID) {
		t.Fatalf("PROCESS_STATE SQL interpolated transaction id: %s", processStateSQL)
	}
	if strings.Contains(safestoreSQL, maliciousTransactionID) {
		t.Fatalf("safestore SQL interpolated transaction id: %s", safestoreSQL)
	}
	if !strings.Contains(processStateSQL, "WHERE transaction_id = :transaction_id") {
		t.Fatalf("PROCESS_STATE SQL must use named bind parameter, got %s", processStateSQL)
	}
	if !strings.Contains(safestoreSQL, "WHERE transaction_id = :transaction_id") {
		t.Fatalf("safestore SQL must use named bind parameter, got %s", safestoreSQL)
	}
	if !strings.Contains(safestoreSQL, "direction = 'Request'") {
		t.Fatalf("safestore SQL must constrain request direction, got %s", safestoreSQL)
	}
}
