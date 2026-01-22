package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

func TestAdminUsageListPathMismatch(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/usage/tenants/tenant-a/extra",
		nil,
	)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminUsageTenantPathMismatch(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)

	request := httptest.NewRequest(http.MethodGet, "/admin/usage/tenants/", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminUsageMissingWorkspaceManager(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:     "127.0.0.1:0",
			AuthManager: authManager,
			TenantStore: tenant.NewInMemoryStore(),
			UsageStore:  tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)
	server.workspaceManager = nil

	request := httptest.NewRequest(http.MethodGet, "/admin/usage/tenants", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminUsageTenantMissing(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/usage/tenants/tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminUsageStoreMissing(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	server.usageStore = nil
	request := httptest.NewRequest(http.MethodGet, "/admin/usage/tenants", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminUsageMethodNotAllowed(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodPost, "/admin/usage/tenants", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestAdminUsageInvalidTenantID(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/usage/tenants/bad%20id",
		nil,
	)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminUsageTenantStoreMissing(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
			UsageStore:       tenant.NewInMemoryUsageStore(),
		},
		nil,
		nil,
	)
	server.tenantStore = nil
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/usage/tenants/tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}
