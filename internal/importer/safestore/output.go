package safestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func outputPaths(basePath string, count int) []string {
	paths := make([]string, 0, count)
	if count == 1 {
		return append(paths, basePath)
	}
	extension := filepath.Ext(basePath)
	stem := strings.TrimSuffix(basePath, extension)
	for index := range count {
		paths = append(paths, fmt.Sprintf("%s-%d%s", stem, index, extension))
	}
	return paths
}

func optionalOutputPaths(basePath string, count int) []string {
	if strings.TrimSpace(basePath) == "" {
		return nil
	}
	return outputPaths(basePath, count)
}

func ensureWritable(paths []string, overwrite bool) error {
	if overwrite {
		return nil
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; pass --overwrite to replace it: %w", path, ErrOutputExists)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat output %s: %w", path, err)
		}
	}
	return nil
}
