package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/audit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/ratelimit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

const testAdminToken = "test-admin-token"

func TestRoutes(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)
	if server.server == nil {
		t.Fatalf("expected server to be initialized")
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
		cancel bool
		auth   bool
		code   int
	}{
		{
			name:   "healthz",
			method: http.MethodGet,
			path:   "/healthz",
			code:   http.StatusOK,
		},
		{
			name:   "mcp-post-missing-body",
			method: http.MethodPost,
			path:   "/mcp",
			auth:   true,
			code:   http.StatusBadRequest,
		},
		{
			name:   "mcp-sse",
			method: http.MethodGet,
			path:   "/mcp/events",
			cancel: true,
			auth:   true,
			code:   http.StatusOK,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				testCase.method,
				testCase.path,
				bytes.NewReader(testCase.body),
			)
			if testCase.cancel {
				ctx, cancel := context.WithCancel(request.Context())
				cancel()
				request = request.WithContext(ctx)
			}
			if testCase.auth {
				request.Header.Set(
					"Authorization",
					"Bearer "+testAdminToken,
				)
			}
			server.server.Handler.ServeHTTP(recorder, request)
			if recorder.Code != testCase.code {
				t.Fatalf("expected %d, got %d", testCase.code, recorder.Code)
			}
		})
	}
}

func TestMCPPostFlow(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)

	initReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(protocol.InitializeParams{ProtocolVersion: protocol.ProtocolVersion}),
	}
	initResp := sendRequest(t, server, initReq)
	if initResp.Error != nil {
		t.Fatalf("unexpected initialize error: %#v", initResp.Error)
	}

	initialized := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		Method:  "initialized",
	}
	recorder := sendNotification(t, server, initialized)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}

	pingReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(2),
		Method:  "ping",
	}
	pingResp := sendRequest(t, server, pingReq)
	if pingResp.Error != nil {
		t.Fatalf("unexpected ping error: %#v", pingResp.Error)
	}
}

func TestSessionHeaderRequiredWhenActive(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)

	session, err := server.sessions.newSession("admin")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("admin", session.id)

	req := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "ping",
	}

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	request.Header.Set(sessionHeader, "missing")
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	request.Header.Set(sessionHeader, session.id)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestSessionIsolationPerTenant(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)

	tenantAToken := mustTenantToken(t, authManager, "tenant-a")
	tenantBToken := mustTenantToken(t, authManager, "tenant-b")

	session, err := server.sessions.newSession("tenant-a")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("tenant-a", session.id)

	req := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "ping",
	}
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+tenantBToken)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+tenantAToken)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)

	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("{}")))
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/mcp/events", nil)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAdminTokenLifecycle(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	store := tenant.NewInMemoryStore()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      store,
		},
		handler,
		nil,
	)

	if _, err := store.Create(tenant.Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

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

	var response tokenResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("expected token to be returned")
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/admin/tenants/tenant-a/tokens",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var listResponse tokenListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	if len(listResponse.Tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(listResponse.Tokens))
	}
	if listResponse.Tokens[0].Token != response.Token {
		t.Fatalf("expected token to match")
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+response.Token)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	request = httptest.NewRequest(
		http.MethodDelete,
		"/admin/tenants/tenant-a/tokens/"+response.Token,
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestAdminTenantLifecycle(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	store := tenant.NewInMemoryStore()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      store,
		},
		handler,
		nil,
	)

	body := []byte(`{"tenant_id":"tenant-a","display_name":"Tenant A"}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants",
		bytes.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	var createResponse tenantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if createResponse.Tenant.ID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", createResponse.Tenant.ID)
	}

	request = httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var listResponse tenantsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	if len(listResponse.Tenants) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(listResponse.Tenants))
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/admin/tenants/tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var getResponse tenantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("parse get response: %v", err)
	}
	if getResponse.Tenant.DisplayName != "Tenant A" {
		t.Fatalf("expected display name, got %q", getResponse.Tenant.DisplayName)
	}

	updateBody := []byte(`{"display_name":"Tenant Alpha"}`)
	request = httptest.NewRequest(
		http.MethodPut,
		"/admin/tenants/tenant-a",
		bytes.NewReader(updateBody),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var updateResponse tenantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &updateResponse); err != nil {
		t.Fatalf("parse update response: %v", err)
	}
	if updateResponse.Tenant.DisplayName != "Tenant Alpha" {
		t.Fatalf("expected updated name, got %q", updateResponse.Tenant.DisplayName)
	}

	request = httptest.NewRequest(
		http.MethodDelete,
		"/admin/tenants/tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var deleteResponse tenantDeleteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &deleteResponse); err != nil {
		t.Fatalf("parse delete response: %v", err)
	}
	if !deleteResponse.Deleted {
		t.Fatalf("expected delete true")
	}

	request = httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	if len(listResponse.Tenants) != 0 {
		t.Fatalf("expected no tenants, got %d", len(listResponse.Tenants))
	}
}

func TestAdminTenantUsageEndpoints(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	store := tenant.NewInMemoryStore()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			TenantStore:      store,
		},
		handler,
		nil,
	)

	if _, err := store.Create(tenant.Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	tokenInfo, err := authManager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/admin/usage/tenants/tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var usageResponse tenantUsageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &usageResponse); err != nil {
		t.Fatalf("parse usage response: %v", err)
	}
	if usageResponse.Usage.RequestCount != 1 {
		t.Fatalf("expected request_count 1, got %d", usageResponse.Usage.RequestCount)
	}
	if usageResponse.Usage.AuditEvents != 1 {
		t.Fatalf("expected audit_events 1, got %d", usageResponse.Usage.AuditEvents)
	}
	if usageResponse.Usage.ActiveSessions != 0 {
		t.Fatalf(
			"expected active_sessions 0, got %d",
			usageResponse.Usage.ActiveSessions,
		)
	}
	if usageResponse.Usage.WorkspaceBytes != 0 {
		t.Fatalf(
			"expected workspace_bytes 0, got %d",
			usageResponse.Usage.WorkspaceBytes,
		)
	}

	request = httptest.NewRequest(http.MethodGet, "/admin/usage/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var listResponse tenantUsageListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("parse usage list: %v", err)
	}
	if len(listResponse.Usages) != 1 {
		t.Fatalf("expected 1 usage entry, got %d", len(listResponse.Usages))
	}
	if listResponse.Usages[0].TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a usage, got %q", listResponse.Usages[0].TenantID)
	}
}

func TestAdminAuditQuery(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	auditStore, err := audit.NewFileStore(
		filepath.Join(t.TempDir(), "audit.json"),
		0,
	)
	if err != nil {
		t.Fatalf("create audit store: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			AuditStore:       auditStore,
		},
		handler,
		nil,
	)

	tokenInfo, err := authManager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/admin/audit?tenant_id=tenant-a",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var response auditResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(response.Records) == 0 {
		t.Fatalf("expected audit records")
	}
	for _, record := range response.Records {
		if record.TenantID != "tenant-a" {
			t.Fatalf("expected tenant-a, got %q", record.TenantID)
		}
	}
}

func TestParseBearerTokenErrors(t *testing.T) {
	if _, err := parseBearerToken(""); !errors.Is(err, auth.ErrMissingToken) {
		t.Fatalf("expected missing token error, got %v", err)
	}
	if _, err := parseBearerToken("Basic abc"); err == nil {
		t.Fatalf("expected invalid authorization header")
	}
	if _, err := parseBearerToken("Bearer "); err == nil {
		t.Fatalf("expected invalid authorization header")
	}
	if token, err := parseBearerToken("Bearer token"); err != nil || token != "token" {
		t.Fatalf("expected token, got %q (%v)", token, err)
	}
}

func TestReadPayloadEmptyBody(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte{}))
	if _, err := server.readPayload(recorder, request); err == nil {
		t.Fatalf("expected error for empty body")
	}
}

func TestRequireRateLimitMissingTenant(t *testing.T) {
	server := NewServer(
		Config{
			Address:     "127.0.0.1:0",
			RateLimiter: mustRateLimiter(t, time.Now, 10),
		},
		nil,
		nil,
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	if ok := server.requireRateLimit(recorder, request); ok {
		t.Fatalf("expected rate limit to fail without tenant")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestRequireRateLimitNoLimiter(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	recorder := httptest.NewRecorder()
	if ok := server.requireRateLimit(recorder, request); !ok {
		t.Fatalf("expected rate limit to be allowed without limiter")
	}
}

func TestRequireRateLimitDenied(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		return base
	}
	usageStore := tenant.NewInMemoryUsageStore()
	server := NewServer(
		Config{
			Address:     "127.0.0.1:0",
			RateLimiter: mustRateLimiter(t, clock, 1),
			UsageStore:  usageStore,
		},
		nil,
		nil,
	)
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	if ok := server.requireRateLimit(recorder, request); !ok {
		t.Fatalf("expected first request allowed")
	}

	recorder = httptest.NewRecorder()
	if ok := server.requireRateLimit(recorder, request); ok {
		t.Fatalf("expected second request denied")
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header")
	}
	if usageStore.Get("tenant-a").RateLimitViolations != 1 {
		t.Fatalf("expected rate limit violation to be recorded")
	}
}

func TestRequireTenantContextMissingWorkspaceManager(t *testing.T) {
	server := NewServer(
		Config{
			Address: "127.0.0.1:0",
		},
		nil,
		nil,
	)
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	if _, _, ok := server.requireTenantContext(recorder, request, "test"); ok {
		t.Fatalf("expected tenant context to fail without workspace manager")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestRequireTenantContextQuotaMissingUsageStore(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			WorkspaceManager: mustWorkspaceManager(t),
			Quota: tenant.Quota{
				MaxRequests: 1,
			},
		},
		nil,
		nil,
	)
	server.usageStore = nil
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	if _, _, ok := server.requireTenantContext(recorder, request, "test"); ok {
		t.Fatalf("expected tenant context to fail without usage store")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestRequireTenantContextWorkspaceError(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	ctx := auth.WithTenantID(context.Background(), " ")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	if _, _, ok := server.requireTenantContext(recorder, request, "test"); ok {
		t.Fatalf("expected tenant context to fail for invalid workspace")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestRequireTenantContextRecordsUsage(t *testing.T) {
	usageStore := tenant.NewInMemoryUsageStore()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			WorkspaceManager: mustWorkspaceManager(t),
			UsageStore:       usageStore,
		},
		nil,
		nil,
	)
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	if _, _, ok := server.requireTenantContext(recorder, request, "test"); !ok {
		t.Fatalf("expected tenant context to succeed")
	}
	if usageStore.Get("tenant-a").RequestCount != 1 {
		t.Fatalf("expected request count to be recorded")
	}
}

func TestSessionStoreActiveCounts(t *testing.T) {
	store := &sessionStore{
		items: make(map[string]map[string]*sseSession),
	}
	store.items["tenant-a"] = map[string]*sseSession{
		"one": {id: "one"},
		"two": {id: "two"},
	}
	store.items["tenant-b"] = map[string]*sseSession{
		"three": {id: "three"},
	}
	if count := store.activeSessions("tenant-a"); count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
	counts := store.activeSessionsAll()
	if counts["tenant-a"] != 2 || counts["tenant-b"] != 1 {
		t.Fatalf("unexpected counts: %v", counts)
	}
}

func TestSessionStoreLimiter(t *testing.T) {
	store := &sessionStore{
		items:   make(map[string]map[string]*sseSession),
		limiter: tenant.NewLimiter(1),
	}
	session, err := store.newSession("tenant-a")
	if err != nil {
		t.Fatalf("expected session, got %v", err)
	}
	defer store.remove("tenant-a", session.id)
	if _, err := store.newSession("tenant-a"); err == nil {
		t.Fatalf("expected session limit error")
	}
}

func TestSessionStoreSendMissing(t *testing.T) {
	store := &sessionStore{
		items: make(map[string]map[string]*sseSession),
	}
	store.send("tenant-a", "missing", []byte("payload"))
}

func TestSessionStoreRemoveMissing(t *testing.T) {
	store := &sessionStore{
		items: make(map[string]map[string]*sseSession),
	}
	store.remove("tenant-a", "missing")
}

func TestHealthz(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handleHealthz(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "ok") {
		t.Fatalf("expected ok response, got %q", recorder.Body.String())
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	handleHealthz(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestHandleMCPPostMethodNotAllowed(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestHandleMCPSSEMethodNotAllowed(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/mcp/events", nil)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

type noFlushWriter struct {
	header http.Header
	code   int
}

func (w *noFlushWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *noFlushWriter) Write(data []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return len(data), nil
}

func (w *noFlushWriter) WriteHeader(statusCode int) {
	w.code = statusCode
}

func TestHandleMCPSSEStreamingUnsupported(t *testing.T) {
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/mcp/events", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	writer := &noFlushWriter{}
	server.handleMCPSSE(writer, request)
	if writer.code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", writer.code)
	}
}

func TestSplitSSELines(t *testing.T) {
	lines := splitSSELines("alpha\nbeta\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[0] != "alpha" || lines[1] != "" || lines[2] != "beta" || lines[3] != "" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestSplitSSELinesEmpty(t *testing.T) {
	lines := splitSSELines("")
	if len(lines) != 1 || lines[0] != "" {
		t.Fatalf("unexpected empty lines: %v", lines)
	}
}

func TestWriteSSE(t *testing.T) {
	var buf bytes.Buffer
	if err := writeSSE(&buf, "message", "one\ntwo"); err != nil {
		t.Fatalf("writeSSE: %v", err)
	}
	expected := "event: message\n" +
		"data: one\n" +
		"data: \n" +
		"data: two\n" +
		"\n"
	if buf.String() != expected {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestWriteJSONResponseError(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSONResponse(recorder, http.StatusOK, func() {})
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestApplyRateLimitHeadersNil(t *testing.T) {
	recorder := httptest.NewRecorder()
	applyRateLimitHeaders(recorder, nil, ratelimit.Decision{})
	if len(recorder.Header()) != 0 {
		t.Fatalf("expected no headers for nil limiter")
	}
}

func TestWriteSSEWriterError(t *testing.T) {
	if err := writeSSE(errWriter{}, "message", "payload"); err == nil {
		t.Fatalf("expected writeSSE error")
	}
}

func TestHandlerNilServer(t *testing.T) {
	server := &Server{}
	handler := server.Handler()
	if handler == nil {
		t.Fatalf("expected handler")
	}
}

func TestHandlerConfiguredServer(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	if handler := server.Handler(); handler == nil {
		t.Fatalf("expected handler for configured server")
	}
}

func TestShutdownNilServer(t *testing.T) {
	server := &Server{}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected nil shutdown error, got %v", err)
	}
}

func TestShutdownConfiguredServer(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	err := server.Shutdown(context.Background())
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestServeNilServer(t *testing.T) {
	server := &Server{}
	if err := server.Serve(context.Background()); err == nil {
		t.Fatalf("expected error for nil server")
	}
}

func TestServeContextCanceled(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("expected serve to exit cleanly, got %v", err)
	}
}

func TestHandleMCPSSEWritesSession(t *testing.T) {
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/mcp/events", nil)
	request = request.WithContext(ctx)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)

	recorder := &sseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		onWrite: func(p []byte) {
			if bytes.Contains(p, []byte("event: session")) {
				cancel()
			}
		},
	}

	server.handleMCPSSE(recorder, request)
	if recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected event-stream content type")
	}
	if recorder.Header().Get(sessionHeader) == "" {
		t.Fatalf("expected session header to be set")
	}
	if !strings.Contains(recorder.Body.String(), "event: session") {
		t.Fatalf("expected session event in body")
	}
}

type sseRecorder struct {
	*httptest.ResponseRecorder
	onWrite func([]byte)
}

func (s *sseRecorder) Write(p []byte) (int, error) {
	if s.onWrite != nil {
		s.onWrite(p)
	}
	return s.ResponseRecorder.Write(p)
}

func TestApplyRateLimitHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	limiter := mustRateLimiter(t, time.Now, 5)
	decision := limiter.Allow("tenant-a")
	applyRateLimitHeaders(recorder, limiter, decision)
	if recorder.Header().Get("RateLimit-Limit") == "" {
		t.Fatalf("expected RateLimit-Limit header")
	}
	if recorder.Header().Get("RateLimit-Remaining") == "" {
		t.Fatalf("expected RateLimit-Remaining header")
	}
	if recorder.Header().Get("RateLimit-Reset") == "" {
		t.Fatalf("expected RateLimit-Reset header")
	}
}

func TestRequireAuthMissingManager(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	if _, ok := server.requireAuth(recorder, request); ok {
		t.Fatalf("expected auth to fail without manager")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestRequireAuthInvalidToken(t *testing.T) {
	manager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(Config{AuthManager: manager}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	if _, ok := server.requireAuth(recorder, request); ok {
		t.Fatalf("expected auth to fail for invalid token")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAdminNonAdminToken(t *testing.T) {
	store := auth.NewInMemoryStore()
	manager, err := auth.NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	tokenInfo, err := manager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	server := NewServer(Config{AuthManager: manager}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	if _, ok := server.requireAdmin(recorder, request); ok {
		t.Fatalf("expected admin auth to fail for tenant token")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("expected WWW-Authenticate header")
	}
}

func TestRequireAdminMissingManager(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	if _, ok := server.requireAdmin(recorder, request); ok {
		t.Fatalf("expected admin auth to fail without manager")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestReadPayloadNilBody(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	recorder := httptest.NewRecorder()
	request := &http.Request{Body: nil}
	if _, err := server.readPayload(recorder, request); err == nil {
		t.Fatalf("expected error for missing body")
	}
}

func TestReadPayloadInvalidBody(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", io.NopCloser(errReader{}))
	if _, err := server.readPayload(recorder, request); err == nil {
		t.Fatalf("expected error for invalid body")
	}
}

func TestAcquireRequestSlotDenied(t *testing.T) {
	requestLimiter := tenant.NewLimiter(1)
	server := NewServer(
		Config{
			Address:        "127.0.0.1:0",
			RequestLimiter: requestLimiter,
		},
		nil,
		nil,
	)
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	release, ok := server.acquireRequestSlot(recorder, request, "tenant-a", "test")
	if !ok {
		t.Fatalf("expected first request slot to succeed")
	}
	defer release()

	recorder = httptest.NewRecorder()
	if _, ok := server.acquireRequestSlot(recorder, request, "tenant-a", "test"); ok {
		t.Fatalf("expected second request slot to be denied")
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func TestAcquireRequestSlotNoLimiter(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	recorder := httptest.NewRecorder()
	release, ok := server.acquireRequestSlot(recorder, request, "tenant-a", "test")
	if !ok {
		t.Fatalf("expected request slot to succeed without limiter")
	}
	release()
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestValidateSessionHeaderNoSessions(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request = request.WithContext(auth.WithTenantID(request.Context(), "tenant-a"))
	sessionID, err := server.validateSessionHeader(request, "tenant-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionID != "" {
		t.Fatalf("expected empty session id")
	}
}

func TestValidateSessionHeaderMissing(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	session, err := server.sessions.newSession("tenant-a")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("tenant-a", session.id)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request = request.WithContext(auth.WithTenantID(request.Context(), "tenant-a"))
	if _, err := server.validateSessionHeader(request, "tenant-a"); err == nil {
		t.Fatalf("expected missing header error")
	}
}

func TestValidateSessionHeaderUnknown(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	session, err := server.sessions.newSession("tenant-a")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("tenant-a", session.id)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request = request.WithContext(auth.WithTenantID(request.Context(), "tenant-a"))
	request.Header.Set(sessionHeader, "missing")
	if _, err := server.validateSessionHeader(request, "tenant-a"); err == nil {
		t.Fatalf("expected unknown session error")
	}
}

func TestValidateSessionHeaderValid(t *testing.T) {
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      mustAuthManager(t),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	session, err := server.sessions.newSession("tenant-a")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("tenant-a", session.id)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request = request.WithContext(auth.WithTenantID(request.Context(), "tenant-a"))
	request.Header.Set(sessionHeader, session.id)
	sessionID, err := server.validateSessionHeader(request, "tenant-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionID != session.id {
		t.Fatalf("expected session id %q, got %q", session.id, sessionID)
	}
}

func TestWorkspaceQuotaEnforced(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	workspaceManager := mustWorkspaceManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: workspaceManager,
			Quota: tenant.Quota{
				MaxWorkspaceBytes: 1,
			},
		},
		handler,
		nil,
	)

	tokenInfo, err := authManager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	workspacePath, err := workspaceManager.Workspace("tenant-a")
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	payloadPath := filepath.Join(workspacePath, "payload.txt")
	if err := os.WriteFile(payloadPath, []byte("over"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestRequestQuotaEnforced(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			Quota: tenant.Quota{
				MaxRequests: 1,
			},
		},
		handler,
		nil,
	)

	tokenInfo, err := authManager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func sendRequest(
	t *testing.T,
	server *Server,
	req protocol.Request,
) protocol.Response {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var resp protocol.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func sendNotification(
	t *testing.T,
	server *Server,
	req protocol.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	server.server.Handler.ServeHTTP(recorder, request)
	return recorder
}

func mustMarshalRaw(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func rawID(id int) *json.RawMessage {
	data, err := json.Marshal(id)
	if err != nil {
		panic(err)
	}
	raw := json.RawMessage(data)
	return &raw
}

func mustAuthManager(t *testing.T) *auth.Manager {
	t.Helper()
	manager, err := auth.NewManager(testAdminToken, nil)
	if err != nil {
		t.Fatalf("create auth manager: %v", err)
	}
	return manager
}

func mustWorkspaceManager(t *testing.T) *tenant.WorkspaceManager {
	t.Helper()
	manager, err := tenant.NewWorkspaceManager(t.TempDir())
	if err != nil {
		t.Fatalf("create workspace manager: %v", err)
	}
	return manager
}

func mustTenantToken(
	t *testing.T,
	manager *auth.Manager,
	tenantID string,
) string {
	t.Helper()
	info, err := manager.CreateToken(tenantID, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return info.Token
}

func TestRateLimiting(t *testing.T) {
	base := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	current := base
	clock := func() time.Time {
		return current
	}
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	limiter := mustRateLimiter(t, clock, 2)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      limiter,
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("RateLimit-Limit") != "2" {
		t.Fatalf("expected rate limit header to be set")
	}
	if recorder.Header().Get("RateLimit-Remaining") != "1" {
		t.Fatalf("expected remaining header to be 1")
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("RateLimit-Remaining") != "0" {
		t.Fatalf("expected remaining header to be 0")
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":3,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
	if recorder.Header().Get("RateLimit-Remaining") != "0" {
		t.Fatalf("expected remaining header to be 0")
	}

	current = base.Add(2 * time.Minute)
	request = httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":4,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 after reset, got %d", recorder.Code)
	}
}

func TestRequestConcurrencyLimit(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	requestLimiter := tenant.NewLimiter(1)
	release, ok := requestLimiter.Acquire("admin")
	if !ok {
		t.Fatalf("expected initial acquire to succeed")
	}
	defer release()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			RequestLimiter:   requestLimiter,
		},
		handler,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func TestSessionLimit(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	sessionLimiter := tenant.NewLimiter(1)
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
			SessionLimiter:   sessionLimiter,
		},
		handler,
		nil,
	)

	session, err := server.sessions.newSession("admin")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer server.sessions.remove("admin", session.id)

	request := httptest.NewRequest(http.MethodGet, "/mcp/events", nil)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func mustRateLimiter(
	t *testing.T,
	clock ratelimit.Clock,
	limit int,
) *ratelimit.Limiter {
	t.Helper()
	limiter, err := ratelimit.NewLimiter(ratelimit.Config{
		Limit:          limit,
		Window:         time.Minute,
		Clock:          clock,
		BackoffEnabled: true,
		BackoffBase:    time.Second,
		BackoffMax:     time.Minute,
	})
	if err != nil {
		t.Fatalf("create rate limiter: %v", err)
	}
	return limiter
}

func TestAuditLogAuthFailure(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	logger, buffer := newTestLogger()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		logger,
	)

	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("{}")))
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}

	entries := parseAuditEntries(t, buffer)
	if len(entries) == 0 {
		t.Fatalf("expected audit log entries")
	}
	if !hasAuditEntry(entries, "auth", "denied", http.StatusUnauthorized) {
		t.Fatalf("expected auth denied audit entry")
	}
}

func TestAuditLogSuccess(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	authManager := mustAuthManager(t)
	logger, buffer := newTestLogger()
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			RateLimiter:      mustRateLimiter(t, time.Now, 10),
			WorkspaceManager: mustWorkspaceManager(t),
		},
		handler,
		logger,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	entries := parseAuditEntries(t, buffer)
	if len(entries) == 0 {
		t.Fatalf("expected audit log entries")
	}
	if !hasAuditEntry(entries, "mcp_request", "success", http.StatusOK) {
		t.Fatalf("expected mcp_request audit entry")
	}
	if !hasAuditTenant(entries, "admin") {
		t.Fatalf("expected audit entry with tenant_id admin")
	}
}

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return logger, buffer
}

func parseAuditEntries(t *testing.T, buffer *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	var entries []map[string]interface{}
	for _, line := range bytes.Split(bytes.TrimSpace(buffer.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("parse audit log: %v", err)
		}
		if audit, ok := entry["audit"].(bool); ok && audit {
			entries = append(entries, entry)
		}
	}
	return entries
}

func hasAuditEntry(
	entries []map[string]interface{},
	action string,
	outcome string,
	status int,
) bool {
	for _, entry := range entries {
		if entry["action"] == action &&
			entry["outcome"] == outcome &&
			entry["status"] == float64(status) {
			return true
		}
	}
	return false
}

func hasAuditTenant(entries []map[string]interface{}, tenantID string) bool {
	for _, entry := range entries {
		if entry["tenant_id"] == tenantID {
			return true
		}
	}
	return false
}
