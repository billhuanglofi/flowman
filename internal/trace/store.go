package trace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const ProcessStateSQL = "SELECT transaction_id, timestamp, step, state, outcome, REGEXP_SUBSTR(sequence, '[^,]+', 1, step + 1) AS service, message FROM PROCESS_STATE WHERE transaction_id = :transaction_id ORDER BY timestamp, step"

var ErrProcessStateQuery = errors.New("trace: process state query failed")

type ProcessStateStore interface {
	Journey(ctx context.Context, request ProcessStateRequest) (Journey, error)
}

type ProcessStateRowSource interface {
	QueryProcessStateRows(ctx context.Context, transactionID string) ([]TraceRow, error)
}

type ProcessStateRequest struct {
	TransactionID  string
	Timeout        time.Duration
	Classification TerminalStateClassification
}

type Store struct {
	source ProcessStateRowSource
}

func NewProcessStateStore(source ProcessStateRowSource) Store {
	return Store{source: source}
}

func (store Store) Journey(ctx context.Context, request ProcessStateRequest) (Journey, error) {
	queryCtx, cancel := processStateQueryContext(ctx, request.Timeout)
	defer cancel()
	rows, err := store.source.QueryProcessStateRows(queryCtx, request.TransactionID)
	if err != nil {
		return Journey{}, fmt.Errorf("query PROCESS_STATE for transaction %q: %w: %w", request.TransactionID, ErrProcessStateQuery, err)
	}
	return MapProcessStateJourney(request.TransactionID, rows, request.Classification), nil
}

func processStateQueryContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

type SQLProcessStateRowSource struct {
	db *sql.DB
}

func NewSQLProcessStateRowSource(db *sql.DB) SQLProcessStateRowSource {
	return SQLProcessStateRowSource{db: db}
}

func (source SQLProcessStateRowSource) QueryProcessStateRows(ctx context.Context, transactionID string) ([]TraceRow, error) {
	rows, err := source.db.QueryContext(ctx, ProcessStateSQL, sql.Named("transaction_id", transactionID))
	if err != nil {
		return nil, fmt.Errorf("execute PROCESS_STATE query: %w", err)
	}
	defer rows.Close()

	traceRows := make([]TraceRow, 0)
	for rows.Next() {
		var row TraceRow
		if err := rows.Scan(&row.TransactionID, &row.Timestamp, &row.Step, &row.State, &row.Outcome, &row.Service, &row.Message); err != nil {
			return nil, fmt.Errorf("scan PROCESS_STATE row: %w", err)
		}
		traceRows = append(traceRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read PROCESS_STATE rows: %w", err)
	}
	return traceRows, nil
}
