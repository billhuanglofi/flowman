package safestore

import (
	"context"
	"time"
)

const RequestSQL = "SELECT transaction_id, timestamp, body, headers FROM safestore WHERE transaction_id = :transaction_id AND direction = 'Request'"

type RequestRow struct {
	TransactionID string
	Timestamp     time.Time
	Body          string
	Headers       string
}

type RequestStore interface {
	RequestRows(ctx context.Context, transactionID string) ([]RequestRow, error)
}
