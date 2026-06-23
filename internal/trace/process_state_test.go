package trace_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/billhuanglofi/flowman/internal/trace"
)

func TestProcessStateQueryIsParameterized(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("create sqlmock db: %v", err)
	}
	defer db.Close()

	maliciousTransactionID := "TX123' OR '1'='1"
	mock.ExpectQuery(trace.ProcessStateSQL).
		WithArgs(sql.Named("transaction_id", maliciousTransactionID)).
		WillReturnRows(processStateSQLRows())

	source := trace.NewSQLProcessStateRowSource(db)

	// When
	_, err = source.QueryProcessStateRows(context.Background(), maliciousTransactionID)

	// Then
	if err != nil {
		t.Fatalf("expected process state query to succeed: %v", err)
	}
	if strings.Contains(trace.ProcessStateSQL, maliciousTransactionID) {
		t.Fatalf("process state SQL interpolated transaction id: %s", trace.ProcessStateSQL)
	}
	if !strings.Contains(trace.ProcessStateSQL, "WHERE transaction_id = :transaction_id") {
		t.Fatalf("process state SQL must use named bind parameter, got %s", trace.ProcessStateSQL)
	}
	if !strings.Contains(trace.ProcessStateSQL, "ORDER BY timestamp, step") {
		t.Fatalf("process state SQL must order by timestamp and step, got %s", trace.ProcessStateSQL)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected parameterized query shape to be used: %v", err)
	}
}

func TestTraceRowsMapJourney(t *testing.T) {
	// Given
	startedAt := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	store := trace.NewProcessStateStore(fakeProcessStateRowSource{rows: []trace.TraceRow{
		{TransactionID: "TX123", Timestamp: startedAt, Step: 0, Service: "gateway", State: "DONE", Outcome: "OK", Message: "accepted"},
		{TransactionID: "TX123", Timestamp: startedAt.Add(time.Second), Step: 1, Service: "payment", State: "RUNNING", Outcome: "PENDING", Message: "waiting"},
	}})

	// When
	journey, err := store.Journey(context.Background(), trace.ProcessStateRequest{
		TransactionID:  "TX123",
		Classification: trace.TerminalStateClassification{TerminalStates: []string{"DONE"}, FailedOutcomes: []string{"FAILED"}},
	})

	// Then
	if err != nil {
		t.Fatalf("expected journey mapping to succeed: %v", err)
	}
	if journey.TransactionID != "TX123" {
		t.Fatalf("expected transaction id TX123, got %q", journey.TransactionID)
	}
	if len(journey.Rows) != 2 {
		t.Fatalf("expected two journey rows, got %#v", journey.Rows)
	}
	assertJourneyRow(t, journey.Rows[0], trace.JourneyRow{TransactionID: "TX123", Timestamp: startedAt, Step: 0, Service: "gateway", State: "DONE", Outcome: "OK", Message: "accepted", DisplayLabel: trace.DisplayLabelTerminal})
	assertJourneyRow(t, journey.Rows[1], trace.JourneyRow{TransactionID: "TX123", Timestamp: startedAt.Add(time.Second), Step: 1, Service: "payment", State: "RUNNING", Outcome: "PENDING", Message: "waiting", DisplayLabel: trace.DisplayLabelStuck})
	if journey.DisplayStuckRow == nil || journey.DisplayStuckRow.Service != "payment" {
		t.Fatalf("expected payment row to be display-only stuck row, got %#v", journey.DisplayStuckRow)
	}
}

func TestEmptyTraceIsWarning(t *testing.T) {
	// Given
	store := trace.NewProcessStateStore(fakeProcessStateRowSource{})

	// When
	journey, err := store.Journey(context.Background(), trace.ProcessStateRequest{TransactionID: "TX404"})

	// Then
	if err != nil {
		t.Fatalf("expected empty trace to return warning without DB failure: %v", err)
	}
	if len(journey.Rows) != 0 {
		t.Fatalf("expected no journey rows, got %#v", journey.Rows)
	}
	if len(journey.Warnings) != 1 || journey.Warnings[0].Code != trace.WarningProcessStateEmpty {
		t.Fatalf("expected empty PROCESS_STATE warning, got %#v", journey.Warnings)
	}
}

func TestProcessStateStoreReturnsContextTimeoutAndQueryErrors(t *testing.T) {
	t.Run("context timeout", func(t *testing.T) {
		// Given
		store := trace.NewProcessStateStore(blockingProcessStateRowSource{})

		// When
		_, err := store.Journey(context.Background(), trace.ProcessStateRequest{TransactionID: "TX123", Timeout: time.Nanosecond})

		// Then
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded from query context, got %v", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		// Given
		queryErr := errors.New("database unavailable")
		store := trace.NewProcessStateStore(fakeProcessStateRowSource{err: queryErr})

		// When
		_, err := store.Journey(context.Background(), trace.ProcessStateRequest{TransactionID: "TX123"})

		// Then
		if !errors.Is(err, queryErr) || !errors.Is(err, trace.ErrProcessStateQuery) {
			t.Fatalf("expected wrapped query error and sentinel, got %v", err)
		}
	})
}

func TestStuckDisplaySelectsLatestNonTerminalOrFailedRowByConfigurableLabels(t *testing.T) {
	// Given
	startedAt := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	store := trace.NewProcessStateStore(fakeProcessStateRowSource{rows: []trace.TraceRow{
		{TransactionID: "TX123", Timestamp: startedAt, Step: 0, Service: "gateway", State: "CUSTOM_DONE", Outcome: "OK", Message: "accepted"},
		{TransactionID: "TX123", Timestamp: startedAt.Add(time.Second), Step: 1, Service: "inventory", State: "CUSTOM_WAIT", Outcome: "PENDING", Message: "waiting"},
		{TransactionID: "TX123", Timestamp: startedAt.Add(2 * time.Second), Step: 2, Service: "payment", State: "CUSTOM_DONE", Outcome: "CUSTOM_FAIL", Message: "declined"},
		{TransactionID: "TX123", Timestamp: startedAt.Add(3 * time.Second), Step: 3, Service: "notifier", State: "CUSTOM_DONE", Outcome: "OK", Message: "sent"},
	}})

	// When
	journey, err := store.Journey(context.Background(), trace.ProcessStateRequest{
		TransactionID: "TX123",
		Classification: trace.TerminalStateClassification{
			TerminalStates: []string{"CUSTOM_DONE"},
			FailedOutcomes: []string{"CUSTOM_FAIL"},
		},
	})

	// Then
	if err != nil {
		t.Fatalf("expected journey mapping to succeed: %v", err)
	}
	if journey.DisplayStuckRow == nil {
		t.Fatalf("expected display-only stuck row")
	}
	if journey.DisplayStuckRow.Service != "payment" || journey.DisplayStuckRow.DisplayLabel != trace.DisplayLabelFailed {
		t.Fatalf("expected latest configured failed row as display stuck row, got %#v", journey.DisplayStuckRow)
	}
	if journey.Rows[3].DisplayLabel != trace.DisplayLabelTerminal {
		t.Fatalf("expected later configured terminal success row to remain terminal display only, got %#v", journey.Rows[3])
	}
}

func processStateSQLRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"transaction_id", "timestamp", "step", "state", "outcome", "service", "message"}).
		AddRow("TX123", time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC), 0, "DONE", "OK", "gateway", "accepted")
}

func assertJourneyRow(t *testing.T, got trace.JourneyRow, want trace.JourneyRow) {
	t.Helper()
	if got != want {
		t.Fatalf("expected journey row %#v, got %#v", want, got)
	}
}

type fakeProcessStateRowSource struct {
	rows []trace.TraceRow
	err  error
}

func (source fakeProcessStateRowSource) QueryProcessStateRows(_ context.Context, _ string) ([]trace.TraceRow, error) {
	return source.rows, source.err
}

type blockingProcessStateRowSource struct{}

func (source blockingProcessStateRowSource) QueryProcessStateRows(ctx context.Context, _ string) ([]trace.TraceRow, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
