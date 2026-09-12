package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

const tenantPayloadLimit = 1 << 20

type tenantCreateRequest struct {
	TenantID    string            `json:"tenant_id"`
	DisplayName string            `json:"display_name,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type tenantUpdateRequest struct {
	DisplayName *string            `json:"display_name,omitempty"`
	Metadata    *map[string]string `json:"metadata,omitempty"`
}

type tenantResponse struct {
	Tenant tenant.Record `json:"tenant"`
}

type tenantsResponse struct {
	Tenants []tenant.Record `json:"tenants"`
}

type tenantDeleteResponse struct {
	TenantID string `json:"tenant_id"`
	Deleted  bool   `json:"deleted"`
}

func (s *Server) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminTenantList(w, r)
	case http.MethodPost:
		s.handleAdminTenantCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminTenant(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := parseTenantTokensPath(r.URL.Path); ok {
		s.handleAdminTenantTokens(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAdminTenantGet(w, r)
	case http.MethodPut:
		s.handleAdminTenantUpdate(w, r)
	case http.MethodDelete:
		s.handleAdminTenantDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminTenantList(w http.ResponseWriter, r *http.Request) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_list",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_list",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_list",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}

	writeJSONResponse(
		w,
		http.StatusOK,
		tenantsResponse{Tenants: s.tenantStore.List()},
	)
	s.auditLog(request, "admin_tenant_list", "success", http.StatusOK, nil)
}

func (s *Server) handleAdminTenantCreate(w http.ResponseWriter, r *http.Request) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_create",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_create",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_create",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}

	payload, err := parseTenantCreateRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	record, err := s.tenantStore.Create(tenant.Record{
		ID:          payload.TenantID,
		DisplayName: payload.DisplayName,
		Metadata:    payload.Metadata,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, tenant.ErrTenantExists) {
			status = http.StatusConflict
		}
		if errors.Is(err, tenant.ErrInvalidTenantID) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		s.auditLog(request, "admin_tenant_create", "error", status, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, tenantResponse{Tenant: record})
	s.auditLog(request, "admin_tenant_create", "success", http.StatusCreated, nil)
}

func (s *Server) handleAdminTenantGet(w http.ResponseWriter, r *http.Request) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_get",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_get",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_get",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}

	targetID := strings.TrimPrefix(request.URL.Path, "/admin/tenants/")
	if err := tenant.ValidateID(targetID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_get",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	record, ok := s.tenantStore.Get(targetID)
	if !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_get",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}

	writeJSONResponse(w, http.StatusOK, tenantResponse{Tenant: record})
	s.auditLog(request, "admin_tenant_get", "success", http.StatusOK, nil)
}

func (s *Server) handleAdminTenantUpdate(w http.ResponseWriter, r *http.Request) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_update",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_update",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_update",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}

	targetID := strings.TrimPrefix(request.URL.Path, "/admin/tenants/")
	if err := tenant.ValidateID(targetID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_update",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	payload, err := parseTenantUpdateRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_update",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	record, err := s.tenantStore.Update(targetID, tenant.Update{
		DisplayName: payload.DisplayName,
		Metadata:    payload.Metadata,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, tenant.ErrTenantNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, tenant.ErrTenantUpdateRequired) ||
			errors.Is(err, tenant.ErrInvalidTenantID) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		s.auditLog(request, "admin_tenant_update", "error", status, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, tenantResponse{Tenant: record})
	s.auditLog(request, "admin_tenant_update", "success", http.StatusOK, nil)
}

func (s *Server) handleAdminTenantDelete(w http.ResponseWriter, r *http.Request) {
	request, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(
		w,
		request,
		"admin_tenant_delete",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_delete",
	)
	if !ok {
		return
	}
	defer release()

	if s.tenantStore == nil {
		http.Error(w, "tenant store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_delete",
			"error",
			http.StatusServiceUnavailable,
			errors.New("tenant store not configured"),
		)
		return
	}

	targetID := strings.TrimPrefix(request.URL.Path, "/admin/tenants/")
	if err := tenant.ValidateID(targetID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_delete",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}
	if _, ok := s.tenantStore.Get(targetID); !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_delete",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}
	if !s.tenantStore.Delete(targetID) {
		http.Error(w, "tenant not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_tenant_delete",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}

	writeJSONResponse(
		w,
		http.StatusOK,
		tenantDeleteResponse{TenantID: targetID, Deleted: true},
	)
	s.auditLog(request, "admin_tenant_delete", "success", http.StatusOK, nil)
}

func parseTenantCreateRequest(r *http.Request) (tenantCreateRequest, error) {
	var payload tenantCreateRequest
	if err := readJSONBody(r, tenantPayloadLimit, &payload); err != nil {
		return tenantCreateRequest{}, err
	}
	if payload.TenantID == "" {
		return tenantCreateRequest{}, auth.ErrTenantRequired
	}
	if err := tenant.ValidateID(payload.TenantID); err != nil {
		return tenantCreateRequest{}, err
	}
	payload.DisplayName = strings.TrimSpace(payload.DisplayName)
	return payload, nil
}

func parseTenantUpdateRequest(r *http.Request) (tenantUpdateRequest, error) {
	var payload tenantUpdateRequest
	if err := readJSONBody(r, tenantPayloadLimit, &payload); err != nil {
		return tenantUpdateRequest{}, err
	}
	if payload.DisplayName == nil && payload.Metadata == nil {
		return tenantUpdateRequest{}, tenant.ErrTenantUpdateRequired
	}
	return payload, nil
}

func readJSONBody(r *http.Request, limit int64, dest interface{}) error {
	if r.Body == nil {
		return errors.New("missing request body")
	}
	defer func() {
		_ = r.Body.Close()
	}()
	data, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("empty request body")
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return err
	}
	return nil
}
