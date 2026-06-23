//go:build oracle || oracle_integration

package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/billhuanglofi/flowman/internal/trace"
	"github.com/godror/godror"
)

func OpenProcessStateStore(ctx context.Context, config Config) (trace.ProcessStateStore, func() error, error) {
	db, err := openDB(ctx, config)
	if err != nil {
		return nil, nil, err
	}
	store := trace.NewProcessStateStore(trace.NewSQLProcessStateRowSource(db))
	return store, db.Close, nil
}

func OpenProcessStateStoreFromEnv(ctx context.Context, envConfig EnvConfig) (trace.ProcessStateStore, func() error, error) {
	config, err := envConfig.Resolve()
	if err != nil {
		return nil, nil, err
	}
	return OpenProcessStateStore(ctx, config)
}

func openDB(ctx context.Context, config Config) (*sql.DB, error) {
	params, err := godror.ParseDSN(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse oracle dsn: %w", err)
	}
	params.Username = config.User
	params.Password = godror.NewPassword(config.Password)
	db := sql.OpenDB(godror.NewConnector(params))
	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping oracle: %w; close oracle connection: %w", err, closeErr)
		}
		return nil, fmt.Errorf("ping oracle: %w", err)
	}
	return db, nil
}
