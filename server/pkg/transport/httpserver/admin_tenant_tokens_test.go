package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

func TestAdminTenantTokenListMissingTenant(t *testing.T) {
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
		http.MethodGet,
		"/admin/tenants/tenant-a/tokens",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenCreateInvalidTenantID(t *testing.T) {
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
		http.MethodPost,
		"/admin/tenants/bad%20id/tokens",
		bytes.NewReader([]byte(`{}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenDeleteAdminToken(t *testing.T) {
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
		http.MethodDelete,
		"/admin/tenants/tenant-a/tokens/"+testAdminToken,
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenDeleteWrongTenant(t *testing.T) {
	store := tenant.NewInMemoryStore()
	if _, err := store.Create(tenant.Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err := store.Create(tenant.Record{ID: "tenant-b"}); err != nil {
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

	body := []byte(`{}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants/tenant-a/tokens",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	request = httptest.NewRequest(
		http.MethodDelete,
		"/admin/tenants/tenant-b/tokens/"+extractToken(t, recorder.Body.Bytes()),
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenStoreMissing(t *testing.T) {
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
	server.tenantStore = nil
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants/tenant-a/tokens",
		bytes.NewReader([]byte(`{}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantTokensMethodNotAllowed(t *testing.T) {
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
		"/admin/tenants/tenant-a/tokens",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenDeleteMethodNotAllowed(t *testing.T) {
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
		http.MethodGet,
		"/admin/tenants/tenant-a/tokens/token-1",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenCreateInvalidJSON(t *testing.T) {
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
		http.MethodPost,
		"/admin/tenants/tenant-a/tokens",
		bytes.NewReader([]byte("{invalid")),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantTokensNotFoundPath(t *testing.T) {
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
		http.MethodGet,
		"/admin/tenants/tenant-a/tokens/",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.handleAdminTenantTokens(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenCreateMissingTenant(t *testing.T) {
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
		http.MethodPost,
		"/admin/tenants/tenant-a/tokens",
		bytes.NewReader([]byte(`{}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenListInvalidTenantID(t *testing.T) {
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
		http.MethodGet,
		"/admin/tenants/bad%20id/tokens",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenListStoreMissing(t *testing.T) {
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
		http.MethodGet,
		"/admin/tenants/tenant-a/tokens",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenRevokeStoreMissing(t *testing.T) {
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
		http.MethodDelete,
		"/admin/tenants/tenant-a/tokens/token-1",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenRevokeInvalidToken(t *testing.T) {
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
		http.MethodDelete,
		"/admin/tenants/tenant-a/tokens/unknown",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenRevokeInvalidTenantID(t *testing.T) {
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
		http.MethodDelete,
		"/admin/tenants/bad%20id/tokens/token-1",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdminTenantTokenRevokeMissingToken(t *testing.T) {
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
	request := httptest.NewRequest(http.MethodDelete, "/admin/tenants/tenant-a/tokens", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.handleAdminTenantTokenRevoke(recorder, request, "tenant-a", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func extractToken(t *testing.T, payload []byte) string {
	t.Helper()
	var response tokenResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("parse token response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("expected token to be set")
	}
	return response.Token
}
