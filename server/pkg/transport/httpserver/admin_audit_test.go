package httpserver

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/audit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
)

func TestParseAuditQueryErrors(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{
			name: "invalid status",
			url:  "/admin/audit?status=bad",
		},
		{
			name: "invalid from",
			url:  "/admin/audit?from=not-a-time",
		},
		{
			name: "invalid to",
			url:  "/admin/audit?to=not-a-time",
		},
		{
			name: "invalid limit",
			url:  "/admin/audit?limit=0",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			if _, err := parseAuditQuery(request); err == nil {
				t.Fatalf("expected error for %s", testCase.name)
			}
		})
	}
}

func TestParseAuditQueryLimitCap(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/audit?limit=2000",
		nil,
	)
	filter, err := parseAuditQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if filter.Limit != maxAuditQueryLimit {
		t.Fatalf("expected limit %d, got %d", maxAuditQueryLimit, filter.Limit)
	}
}

func TestAdminAuditMethodNotAllowed(t *testing.T) {
	server := NewServer(Config{}, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/admin/audit", nil)
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", recorder.Code)
	}
}

func TestAdminAuditMissingStore(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	server := NewServer(
		Config{
			Address:          "127.0.0.1:0",
			AuthManager:      authManager,
			WorkspaceManager: mustWorkspaceManager(t),
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/admin/audit", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestParseAuditQueryFromTo(t *testing.T) {
	now := time.Now().UTC()
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/audit?from="+now.Format(time.RFC3339)+"&to="+now.Format(time.RFC3339),
		nil,
	)
	filter, err := parseAuditQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if filter.From == nil || filter.To == nil {
		t.Fatalf("expected from/to timestamps to be set")
	}
}

func TestAdminAuditInvalidFilter(t *testing.T) {
	authManager, err := auth.NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
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
			WorkspaceManager: mustWorkspaceManager(t),
			AuditStore:       auditStore,
		},
		nil,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/audit?limit=0",
		nil,
	)
	request.Header.Set("Authorization", "Bearer admin-token")
	recorder := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
