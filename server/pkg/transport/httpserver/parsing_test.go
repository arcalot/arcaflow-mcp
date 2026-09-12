package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

func TestParseTenantTokensPath(t *testing.T) {
	tenantID, token, ok := parseTenantTokensPath("/admin/tenants/tenant-a/tokens")
	if !ok || tenantID != "tenant-a" || token != "" {
		t.Fatalf("unexpected result: %v %v %v", tenantID, token, ok)
	}
	tenantID, token, ok = parseTenantTokensPath("/admin/tenants/tenant-a/tokens/token-1")
	if !ok || tenantID != "tenant-a" || token != "token-1" {
		t.Fatalf("unexpected result: %v %v %v", tenantID, token, ok)
	}
	_, _, ok = parseTenantTokensPath("/admin/tenants/tenant-a/token")
	if ok {
		t.Fatalf("expected parse to fail")
	}
}

func TestParseTenantUsagePath(t *testing.T) {
	tenantID, ok := parseTenantUsagePath("/admin/usage/tenants/tenant-a")
	if !ok || tenantID != "tenant-a" {
		t.Fatalf("unexpected result: %v %v", tenantID, ok)
	}
	_, ok = parseTenantUsagePath("/admin/usage/tenants")
	if ok {
		t.Fatalf("expected parse to fail")
	}
}

func TestParseTenantCreateRequestMissingID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants",
		bytes.NewReader([]byte(`{"display_name":"Tenant"}`)),
	)
	if _, err := parseTenantCreateRequest(request); err != auth.ErrTenantRequired {
		t.Fatalf("expected tenant required error, got %v", err)
	}
}

func TestParseTenantUpdateRequestMissingField(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPut,
		"/admin/tenants/tenant-a",
		bytes.NewReader([]byte(`{}`)),
	)
	if _, err := parseTenantUpdateRequest(request); err != tenant.ErrTenantUpdateRequired {
		t.Fatalf("expected update required error, got %v", err)
	}
}

func TestReadJSONBodyErrors(t *testing.T) {
	request := &http.Request{Body: nil}
	var payload map[string]interface{}
	if err := readJSONBody(request, 128, &payload); err == nil {
		t.Fatalf("expected error for missing body")
	}

	request = httptest.NewRequest(http.MethodPost, "/admin/tenants", nil)
	if err := readJSONBody(request, 128, &payload); err == nil {
		t.Fatalf("expected error for empty body")
	}

	request = httptest.NewRequest(
		http.MethodPost,
		"/admin/tenants",
		bytes.NewReader([]byte(`{invalid`)),
	)
	if err := readJSONBody(request, 128, &payload); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}
