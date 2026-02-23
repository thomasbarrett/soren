package builtin

import (
	"errors"
	"fmt"
	"path/filepath"
)

func validatePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return absPath, nil
}
