package oracle

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/billhuanglofi/flowman/internal/importer/safestore"
)

const SupportNotEnabledMessage = "oracle support not enabled; rebuild with -tags oracle or run integration with -tags oracle_integration"

var ErrSupportNotEnabled = errors.New(SupportNotEnabledMessage)

type Config struct {
	DSN      string
	User     string
	Password string
}

type EnvConfig struct {
	DSNEnv      string
	UserEnv     string
	PasswordEnv string
}

func (config EnvConfig) Resolve() (Config, error) {
	dsn, err := lookupRequiredEnv(config.DSNEnv)
	if err != nil {
		return Config{}, err
	}
	user, err := lookupRequiredEnv(config.UserEnv)
	if err != nil {
		return Config{}, err
	}
	password, err := lookupRequiredEnv(config.PasswordEnv)
	if err != nil {
		return Config{}, err
	}
	return Config{DSN: dsn, User: user, Password: password}, nil
}

func lookupRequiredEnv(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("oracle env name is empty")
	}
	value, ok := os.LookupEnv(trimmed)
	if !ok {
		return "", fmt.Errorf("oracle env %q is not set", trimmed)
	}
	return value, nil
}

type SafestoreRequestRow = safestore.RequestRow

type SafestoreStore = safestore.RequestStore
