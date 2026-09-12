// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceManager manages per-tenant workspace roots.
type WorkspaceManager struct {
	root string
}

// NewWorkspaceManager initializes a workspace manager for tenants.
func NewWorkspaceManager(root string) (*WorkspaceManager, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("workspace root must not be empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, err
	}
	return &WorkspaceManager{root: absolute}, nil
}

// Workspace ensures a tenant workspace directory exists and returns its path.
func (m *WorkspaceManager) Workspace(tenantID string) (string, error) {
	if m == nil {
		return "", errors.New("workspace manager not configured")
	}
	safeID := sanitizeTenantID(tenantID)
	if safeID == "" {
		return "", errors.New("tenant id is required")
	}
	path := filepath.Join(m.root, safeID)
	rel, err := filepath.Rel(m.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("invalid tenant workspace path")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeTenantID(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '_'
		}
	}, tenantID)
}
