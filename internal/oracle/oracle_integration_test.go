//go:build oracle_integration

package oracle_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/billhuanglofi/flowman/internal/oracle"
	"github.com/billhuanglofi/flowman/internal/trace"
)

func TestOracleIntegration_ProcessStateStore_queriesByBoundTransactionID(t *testing.T) {
	// Given
	config := oracleIntegrationConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, closeStore, err := oracle.OpenProcessStateStore(ctx, config)
	if err != nil {
		t.Fatalf("open oracle process state store: %v", err)
	}
	t.Cleanup(func() {
		if err := closeStore(); err != nil {
			t.Fatalf("close oracle process state store: %v", err)
		}
	})

	// When
	journey, err := store.Journey(ctx, trace.ProcessStateRequest{TransactionID: "FLOWMAN_INTEGRATION_PROBE", Timeout: 30 * time.Second})

	// Then
	if err != nil {
		t.Fatalf("query PROCESS_STATE by bound transaction id: %v", err)
	}
	if journey.TransactionID != "FLOWMAN_INTEGRATION_PROBE" {
		t.Fatalf("expected journey transaction id FLOWMAN_INTEGRATION_PROBE, got %q", journey.TransactionID)
	}
}

func TestOracleIntegration_SafestoreStore_queriesRequestsByBoundTransactionID(t *testing.T) {
	// Given
	config := oracleIntegrationConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, closeStore, err := oracle.OpenSafestoreStore(ctx, config)
	if err != nil {
		t.Fatalf("open oracle safestore store: %v", err)
	}
	t.Cleanup(func() {
		if err := closeStore(); err != nil {
			t.Fatalf("close oracle safestore store: %v", err)
		}
	})

	// When
	rows, err := store.RequestRows(ctx, "FLOWMAN_INTEGRATION_PROBE")

	// Then
	if err != nil {
		t.Fatalf("query safestore by bound transaction id: %v", err)
	}
	for _, row := range rows {
		if row.TransactionID != "FLOWMAN_INTEGRATION_PROBE" {
			t.Fatalf("expected bound transaction id only, got row %#v", row)
		}
	}
}

func oracleIntegrationConfig(t *testing.T) oracle.Config {
	t.Helper()
	dsn, dsnOK := os.LookupEnv("FLOWMAN_ORACLE_DSN")
	user, userOK := os.LookupEnv("FLOWMAN_ORACLE_USER")
	password, passwordOK := os.LookupEnv("FLOWMAN_ORACLE_PASSWORD")
	if !dsnOK || !userOK || !passwordOK {
		t.Skip("set FLOWMAN_ORACLE_DSN, FLOWMAN_ORACLE_USER, and FLOWMAN_ORACLE_PASSWORD to run Oracle integration tests")
	}
	return oracle.Config{DSN: dsn, User: user, Password: password}
}
