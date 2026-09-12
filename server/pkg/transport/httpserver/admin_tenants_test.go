package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

func TestAdminTenantCreateInvalidID(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)

	body := []byte(`{"tenant_id":"bad id"}`)
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantCreateMissingID(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	body := []byte(`{"display_name":"Tenant"}`)
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantUpdateMissingBody(t *testing.T) {
	store := tenant.NewInMemoryStore()
	if _, err := store.Create(tenant.Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      store,
		},
		nil,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/tenants/tenant-a",
		bytes.NewReader([]byte(`{}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantGetMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants/missing", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantDeleteMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodDelete, "/admin/tenants/missing", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantMethodNotAllowed(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants/tenant-a", nil)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestAdminTenantCreateDuplicate(t *testing.T) {
	store := tenant.NewInMemoryStore()
	if _, err := store.Create(tenant.Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      store,
		},
		nil,
		nil,
	)
	body := []byte(`{"tenant_id":"tenant-a"}`)
	request := httptest.NewRequest(http.MethodPost, "/admin/tenants", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

func TestAdminTenantUpdateInvalidID(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/tenants/bad%20id",
		bytes.NewReader([]byte(`{"display_name":"name"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantDeleteInvalidID(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodDelete, "/admin/tenants/bad%20id", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantStoreMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	server.tenantStore = nil
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantCreateStoreMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	server.tenantStore = nil
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants",
		bytes.NewReader([]byte(`{"tenant_id":"tenant-a"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantUpdateMissingTenant(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/tenants/tenant-a",
		bytes.NewReader([]byte(`{"display_name":"Name"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantGetInvalidID(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants/bad%20id", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantAuthMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      tenant.NewInMemoryStore(),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("expected WWW-Authenticate header")
	}
}

func TestAdminTenantGetStoreMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	server.tenantStore = nil
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants/tenant-a", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantDeleteStoreMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	server.tenantStore = nil
	request := httptest.NewRequest(http.MethodDelete, "/admin/tenants/tenant-a", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}
