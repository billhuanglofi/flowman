//go:build !oracle && !oracle_integration

package oracle

import (
	"context"

	"github.com/billhuanglofi/flowman/internal/trace"
)

func OpenProcessStateStore(context.Context, Config) (trace.ProcessStateStore, func() error, error) {
	return nil, nil, ErrSupportNotEnabled
}

func OpenProcessStateStoreFromEnv(context.Context, EnvConfig) (trace.ProcessStateStore, func() error, error) {
	return nil, nil, ErrSupportNotEnabled
}
