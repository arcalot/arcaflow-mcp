package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/audit"
)

const maxAuditQueryLimit = 1000

type auditResponse struct {
	Records []audit.Record `json:"records"`
}

func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		"admin_audit_query",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_audit_query",
	)
	if !ok {
		return
	}
	defer release()

	if s.auditStore == nil {
		http.Error(w, "audit store not configured", http.StatusServiceUnavailable)
		s.auditLog(
			request,
			"admin_audit_query",
			"error",
			http.StatusServiceUnavailable,
			errors.New("audit store not configured"),
		)
		return
	}

	filter, err := parseAuditQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_audit_query",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	records := s.auditStore.Query(filter)
	writeJSONResponse(w, http.StatusOK, auditResponse{Records: records})
	s.auditLog(request, "admin_audit_query", "success", http.StatusOK, nil)
}

func parseAuditQuery(r *http.Request) (audit.Filter, error) {
	query := r.URL.Query()
	filter := audit.Filter{
		TenantID: query.Get("tenant_id"),
		Action:   query.Get("action"),
		Outcome:  query.Get("outcome"),
	}
	if value := query.Get("status"); value != "" {
		status, err := strconv.Atoi(value)
		if err != nil {
			return audit.Filter{}, errors.New("invalid status")
		}
		filter.Status = &status
	}
	if value := query.Get("from"); value != "" {
		timestamp, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return audit.Filter{}, errors.New("invalid from timestamp")
		}
		filter.From = &timestamp
	}
	if value := query.Get("to"); value != "" {
		timestamp, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return audit.Filter{}, errors.New("invalid to timestamp")
		}
		filter.To = &timestamp
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 {
			return audit.Filter{}, errors.New("invalid limit")
		}
		if limit > maxAuditQueryLimit {
			limit = maxAuditQueryLimit
		}
		filter.Limit = limit
	}
	return filter, nil
}
