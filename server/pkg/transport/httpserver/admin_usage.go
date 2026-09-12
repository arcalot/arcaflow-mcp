package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
)

type tenantUsageResponse struct {
	Usage tenant.UsageStats `json:"usage"`
}

type tenantUsageListResponse struct {
	Usages []tenant.UsageStats `json:"usages"`
}

func (s *Server) handleAdminTenantsUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/admin/usage/tenants" {
		http.NotFound(w, r)
		return
	}
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
		"admin_tenant_usage_list",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_tenant_usage_list",
	)
	if !ok {
		return
	}
	defer release()

	if s.usageStore == nil || s.tenantStore == nil {
		http.Error(w, "usage store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_usage_list",
			"error",
			http.StatusServiceUnavailable,
			errors.New("usage store not configured"),
		)
		return
	}
	if s.workspaceManager == nil {
		http.Error(w, "tenant workspace not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_usage_list",
			"error",
			http.StatusServiceUnavailable,
			errors.New("workspace manager not configured"),
		)
		return
	}

	records := s.tenantStore.List()
	usages := make([]tenant.UsageStats, 0, len(records))
	sessionCounts := s.sessions.activeSessionsAll()
	for _, record := range records {
		usage := s.usageStore.Get(record.ID)
		workspacePath, err := s.workspaceManager.Workspace(record.ID)
		if err != nil {
			http.Error(w, "tenant workspace unavailable", http.StatusInternalServerError)
			s.auditLog(
				request,
				"admin_tenant_usage_list",
				"error",
				http.StatusInternalServerError,
				err,
			)
			return
		}
		workspaceBytes, err := tenant.WorkspaceUsage(workspacePath)
		if err != nil {
			http.Error(w, "tenant workspace unavailable", http.StatusInternalServerError)
			s.auditLog(
				request,
				"admin_tenant_usage_list",
				"error",
				http.StatusInternalServerError,
				err,
			)
			return
		}
		s.usageStore.SetWorkspaceBytes(record.ID, workspaceBytes)
		usage.ActiveSessions = sessionCounts[record.ID]
		usage.WorkspaceBytes = workspaceBytes
		usages = append(usages, usage)
	}

	writeJSONResponse(w, http.StatusOK, tenantUsageListResponse{Usages: usages})
	s.auditLog(request, "admin_tenant_usage_list", "success", http.StatusOK, nil)
}

func (s *Server) handleAdminTenantUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := parseTenantUsagePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
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
		"admin_tenant_usage_get",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		contextTenantID,
		"admin_tenant_usage_get",
	)
	if !ok {
		return
	}
	defer release()

	if s.usageStore == nil || s.tenantStore == nil {
		http.Error(w, "usage store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_usage_get",
			"error",
			http.StatusServiceUnavailable,
			errors.New("usage store not configured"),
		)
		return
	}
	if s.workspaceManager == nil {
		http.Error(w, "tenant workspace not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_tenant_usage_get",
			"error",
			http.StatusServiceUnavailable,
			errors.New("workspace manager not configured"),
		)
		return
	}
	if err := tenant.ValidateID(tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_tenant_usage_get",
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
			"admin_tenant_usage_get",
			"error",
			http.StatusNotFound,
			tenant.ErrTenantNotFound,
		)
		return
	}

	usage := s.usageStore.Get(tenantID)
	workspacePath, err := s.workspaceManager.Workspace(tenantID)
	if err != nil {
		http.Error(w, "tenant workspace unavailable", http.StatusInternalServerError)
		s.auditLog(
			request,
			"admin_tenant_usage_get",
			"error",
			http.StatusInternalServerError,
			err,
		)
		return
	}
	workspaceBytes, err := tenant.WorkspaceUsage(workspacePath)
	if err != nil {
		http.Error(w, "tenant workspace unavailable", http.StatusInternalServerError)
		s.auditLog(
			request,
			"admin_tenant_usage_get",
			"error",
			http.StatusInternalServerError,
			err,
		)
		return
	}
	s.usageStore.SetWorkspaceBytes(tenantID, workspaceBytes)
	usage.ActiveSessions = s.sessions.activeSessions(tenantID)
	usage.WorkspaceBytes = workspaceBytes
	writeJSONResponse(w, http.StatusOK, tenantUsageResponse{Usage: usage})
	s.auditLog(request, "admin_tenant_usage_get", "success", http.StatusOK, nil)
}

func parseTenantUsagePath(path string) (string, bool) {
	trimmed := strings.TrimPrefix(path, "/admin/usage/tenants/")
	if trimmed == "" || trimmed == path {
		return "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}
