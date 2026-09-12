package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

const tokenPayloadLimit = 1 << 20

type tokenCreateRequest struct {
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type tokenResponse struct {
	Token     string     `json:"token"`
	TenantID  string     `json:"tenant_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type tokenListResponse struct {
	Tokens []tokenResponse `json:"tokens"`
}

func (s *Server) handleAdminTenantTokens(w http.ResponseWriter, r *http.Request) {
	tenantID, token, ok := parseTenantTokensPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if token == "" {
		if r.Method == http.MethodGet {
			s.handleAdminTenantTokenList(w, r, tenantID)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleAdminTenantTokenCreate(w, r, tenantID)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handleAdminTenantTokenRevoke(w, r, tenantID, token)
}

func (s *Server) handleAdminTenantTokenCreate(
	w http.ResponseWriter,
	r *http.Request,
	tenantID string,
) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, contextTenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_token_create",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		contextTenantID,
		"admin_tenant_token_create",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_token_create",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}
	if err := tenant.ValidateID(tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	if _, ok := s.tenantStore.Get(tenantID); !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_token_create",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}

	payload, err := parseTokenCreateRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	info, err := s.authManager.CreateToken(tenantID, payload.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	writeJSONResponse(w, http.StatusCreated, tokenResponse{
		Token:     info.Token,
		TenantID:  info.TenantID,
		CreatedAt: info.CreatedAt,
		ExpiresAt: info.ExpiresAt,
	})
	s.auditLog(
		request,
		"admin_tenant_token_create",
		"success",
		http.StatusCreated,
		nil,
	)
}

func (s *Server) handleAdminTenantTokenList(
	w http.ResponseWriter,
	r *http.Request,
	tenantID string,
) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, contextTenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_token_list",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		contextTenantID,
		"admin_tenant_token_list",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_token_list",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}
	if err := tenant.ValidateID(tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_list",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	if _, ok := s.tenantStore.Get(tenantID); !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_token_list",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}

	infos := s.authManager.ListTokens(tenantID)
	tokens := make([]tokenResponse, 0, len(infos))
	for _, info := range infos {
		tokens = append(tokens, tokenResponse{
			Token:     info.Token,
			TenantID:  info.TenantID,
			CreatedAt: info.CreatedAt,
			ExpiresAt: info.ExpiresAt,
		})
	}
	writeJSONResponse(w, http.StatusOK, tokenListResponse{Tokens: tokens})
	s.auditLog(
		request,
		"admin_tenant_token_list",
		"success",
		http.StatusOK,
		nil,
	)
}

func (s *Server) handleAdminTenantTokenRevoke(
	w http.ResponseWriter,
	r *http.Request,
	tenantID string,
	token string,
) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, contextTenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_token_revoke",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		contextTenantID,
		"admin_tenant_token_revoke",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}
	if err := tenant.ValidateID(tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusBadRequest,
			errors.New("missing token"),
		)
		return
	}
	if s.authManager.IsAdmin(token) {
		http.Error(w, "admin token cannot be revoked", http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusBadRequest,
			errors.New("admin token cannot be revoked"),
		)
		return
	}

	info, err := s.authManager.Authenticate(token)
	if err != nil {
		http.Error(w, "token not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusNotFound,
			err,
		)
		return
	}
	if info.TenantID != tenantID {
		http.Error(w, "token not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusNotFound,
			auth.ErrInvalidToken,
		)
		return
	}
	if !s.authManager.RevokeToken(token) {
		http.Error(w, "token not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_token_revoke",
			"error",
			http.StatusNotFound,
			auth.ErrInvalidToken,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	s.auditLog(
		request,
		"admin_tenant_token_revoke",
		"success",
		http.StatusNoContent,
		nil,
	)
}

func parseTokenCreateRequest(r *http.Request) (tokenCreateRequest, error) {
	var payload tokenCreateRequest
	if err := readJSONBody(r, tokenPayloadLimit, &payload); err != nil {
		return tokenCreateRequest{}, err
	}
	return payload, nil
}

func parseTenantTokensPath(path string) (string, string, bool) {
	trimmed := strings.TrimPrefix(path, "/admin/tenants/")
	if trimmed == "" || trimmed == path {
		return "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] != "tokens" {
		return "", "", false
	}
	if len(parts) == 2 {
		return parts[0], "", true
	}
	if len(parts) == 3 && parts[2] != "" {
		return parts[0], parts[2], true
	}
	return "", "", false
}
