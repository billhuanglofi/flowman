package config

import (
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

func secretEnvNames(environment model.Environment, requests []model.Request) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, 3)
	for _, name := range []string{environment.Oracle.DSNEnv, environment.Oracle.UserEnv, environment.Oracle.PasswordEnv} {
		names = appendSecretEnvName(names, seen, name)
	}
	for _, request := range requests {
		for _, header := range request.Headers {
			if secrets.IsSensitiveHeader(header.Name) {
				names = appendSecretEnvName(names, seen, header.Env)
			}
		}
	}
	return names
}

func appendSecretEnvName(names []string, seen map[string]struct{}, name string) []string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return names
	}
	if _, ok := seen[trimmed]; ok {
		return names
	}
	seen[trimmed] = struct{}{}
	return append(names, trimmed)
}
