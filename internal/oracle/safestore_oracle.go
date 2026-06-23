//go:build oracle || oracle_integration

package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
)

const SafestoreRequestSQL = safestore.RequestSQL

type sqlSafestoreStore struct {
	db *sql.DB
}

func OpenSafestoreStore(ctx context.Context, config Config) (SafestoreStore, func() error, error) {
	db, err := openDB(ctx, config)
	if err != nil {
		return nil, nil, err
	}
	return sqlSafestoreStore{db: db}, db.Close, nil
}

func OpenSafestoreStoreFromEnv(ctx context.Context, envConfig EnvConfig) (SafestoreStore, func() error, error) {
	config, err := envConfig.Resolve()
	if err != nil {
		return nil, nil, err
	}
	return OpenSafestoreStore(ctx, config)
}

func (store sqlSafestoreStore) RequestRows(ctx context.Context, transactionID string) ([]SafestoreRequestRow, error) {
	rows, err := store.db.QueryContext(ctx, SafestoreRequestSQL, sql.Named("transaction_id", transactionID))
	if err != nil {
		return nil, fmt.Errorf("execute safestore request query: %w", err)
	}
	defer rows.Close()

	requestRows := make([]SafestoreRequestRow, 0)
	for rows.Next() {
		var row SafestoreRequestRow
		if err := rows.Scan(&row.TransactionID, &row.Timestamp, &row.Body, &row.Headers); err != nil {
			return nil, fmt.Errorf("scan safestore request row: %w", err)
		}
		requestRows = append(requestRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read safestore request rows: %w", err)
	}
	return requestRows, nil
}
