// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"errors"
	"os"
	"path/filepath"
)

// WorkspaceUsage returns the total size in bytes for a tenant workspace.
func WorkspaceUsage(path string) (int64, error) {
	if path == "" {
		return 0, errors.New("workspace path required")
	}
	var total int64
	err := filepath.WalkDir(path, func(entryPath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
