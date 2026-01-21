package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	body := []byte(`{"tenant_id":"tenant-a"}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tokens",
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
		"/admin/tokens/"+response.Token,
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+testAdminToken)
	recorder = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
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
