//go:build !oracle && !oracle_integration

package oracle

import "context"

func OpenSafestoreStore(context.Context, Config) (SafestoreStore, func() error, error) {
	return nil, nil, ErrSupportNotEnabled
}

func OpenSafestoreStoreFromEnv(context.Context, EnvConfig) (SafestoreStore, func() error, error) {
	return nil, nil, ErrSupportNotEnabled
}
