package workflow

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveFilesystemPath converts relative paths to absolute paths based on
// the current working directory. Absolute paths are returned unchanged.
// This ensures filesystem operations work correctly regardless of whether
// the client provides relative or absolute paths.
func ResolveFilesystemPath(location string) (string, error) {
	if location == "" {
		return "", fmt.Errorf("location cannot be empty")
	}

	// Clean the path first to normalize it
	location = filepath.Clean(location)

	// If already absolute, return as-is
	if filepath.IsAbs(location) {
		return location, nil
	}

	// Resolve relative path against current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}

	absPath := filepath.Join(cwd, location)
	return absPath, nil
}
