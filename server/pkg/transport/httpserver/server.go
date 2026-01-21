// Package httpserver implements MCP HTTP/SSE transport scaffolding.
package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/ratelimit"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tenant"
	"github.com/arcalot/arcaflow-mcp/server/pkg/version"
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 60 * time.Second
)

const (
	sessionHeader = "Mcp-Session-Id"
)

// Server provides an HTTP/SSE transport wrapper for MCP.
type Server struct {
	logger           *slog.Logger
	server           *http.Server
	handler          *protocol.Server
	sessions         *sessionStore
	authManager      *auth.Manager
	rateLimiter      *ratelimit.Limiter
	auditLogger      *slog.Logger
	workspaceManager *tenant.WorkspaceManager
	requestLimiter   *tenant.Limiter
}

// Config holds HTTP server configuration for MCP transport.
type Config struct {
	Address string
	// AuthManager enforces bearer authentication in server mode.
	AuthManager *auth.Manager
	// RateLimiter enforces per-tenant rate limiting.
	RateLimiter *ratelimit.Limiter
	// WorkspaceManager provides tenant workspace paths.
	WorkspaceManager *tenant.WorkspaceManager
	// RequestLimiter enforces per-tenant concurrency limits.
	RequestLimiter *tenant.Limiter
	// SessionLimiter enforces per-tenant SSE session limits.
	SessionLimiter *tenant.Limiter
}

// NewServer constructs a new HTTP/SSE transport server.
func NewServer(
	config Config,
	handler *protocol.Server,
	logger *slog.Logger,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if handler == nil {
		handler = protocol.NewServer(logger.With("component", "protocol"), protocol.ServerInfo{
			Name:    "arcaflow-mcp",
			Version: version.Current(),
		})
	}

	mux := http.NewServeMux()
	server := &Server{
		logger:           logger,
		handler:          handler,
		authManager:      config.AuthManager,
		rateLimiter:      config.RateLimiter,
		auditLogger:      logger.With("component", "audit"),
		workspaceManager: config.WorkspaceManager,
		requestLimiter:   config.RequestLimiter,
		sessions: &sessionStore{
			items:   make(map[string]map[string]*sseSession),
			limiter: config.SessionLimiter,
		},
	}
	mux.HandleFunc("/mcp", server.handleMCPPost)
	mux.HandleFunc("/mcp/events", server.handleMCPSSE)
	mux.HandleFunc("/admin/tokens", server.handleAdminTokens)
	mux.HandleFunc("/admin/tokens/", server.handleAdminToken)
	mux.HandleFunc("/healthz", handleHealthz)

	httpServer := &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	server.server = httpServer
	return server
}

// Serve starts the HTTP server and blocks until shutdown.
func (s *Server) Serve(ctx context.Context) error {
	if s.server == nil {
		return errors.New("http server not configured")
	}
	s.logger.Info("starting HTTP/SSE server", "address", s.server.Addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Handler exposes the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	if s.server == nil {
		return http.NewServeMux()
	}
	return s.server.Handler
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) handleMCPPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(w, request, "mcp_request")
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(w, request, tenantID, "mcp_request")
	if !ok {
		return
	}
	defer release()

	sessionID, err := s.validateSessionHeader(request, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"mcp_request",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	payload, err := s.readPayload(w, request)
	if err != nil {
		s.logger.Warn("failed to read MCP request", "error", err)
		s.auditLog(
			request,
			"mcp_request",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	responses, err := s.handler.Handle(request.Context(), payload)
	if err != nil {
		s.logger.Error("protocol handler error", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		s.auditLog(
			request,
			"mcp_request",
			"error",
			http.StatusInternalServerError,
			err,
		)
		return
	}
	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		s.auditLog(
			request,
			"mcp_notification",
			"success",
			http.StatusNoContent,
			nil,
		)
		return
	}

	if sessionID != "" {
		s.sessions.send(tenantID, sessionID, responses[0])
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(responses[0]); err != nil {
		return
	}
	s.auditLog(request, "mcp_request", "success", http.StatusOK, nil)
}

func (s *Server) readPayload(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Body == nil {
		http.Error(w, "missing request body", http.StatusBadRequest)
		return nil, errors.New("missing request body")
	}

	limited := http.MaxBytesReader(w, r.Body, 10<<20)
	defer func() {
		_ = limited.Close()
	}()

	payload, err := io.ReadAll(limited)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return nil, err
	}
	if len(payload) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return nil, errors.New("empty request body")
	}
	return payload, nil
}

func (s *Server) handleMCPSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !s.requireRateLimit(w, request) {
		return
	}
	request, tenantID, ok := s.requireTenantContext(w, request, "mcp_sse")
	if !ok {
		return
	}
	if request.Context().Err() != nil {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		s.auditLog(
			request,
			"mcp_sse",
			"error",
			http.StatusInternalServerError,
			errors.New("streaming unsupported"),
		)
		return
	}

	session, err := s.sessions.newSession(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		s.auditLog(
			request,
			"mcp_sse",
			"denied",
			http.StatusTooManyRequests,
			err,
		)
		return
	}
	defer s.sessions.remove(tenantID, session.id)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set(sessionHeader, session.id)

	if err := writeSSE(w, "session", session.id); err != nil {
		s.auditLog(
			request,
			"mcp_sse",
			"error",
			http.StatusInternalServerError,
			err,
		)
		return
	}
	flusher.Flush()
	s.auditLog(request, "mcp_sse", "success", http.StatusOK, nil)

	for {
		select {
		case <-request.Context().Done():
			return
		case payload, ok := <-session.send:
			if !ok {
				return
			}
			if err := writeSSE(w, "message", string(payload)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// handleAdminTokens creates tenant tokens via POST /admin/tokens.
func (s *Server) handleAdminTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		"admin_token_create",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_token_create",
	)
	if !ok {
		return
	}
	defer release()

	payload, err := parseTokenRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_token_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	info, err := s.authManager.CreateToken(payload.TenantID, payload.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_token_create",
			"error",
			http.StatusBadRequest,
			err,
		)
		return
	}

	writeJSON(w, http.StatusCreated, tokenResponse{
		Token:     info.Token,
		TenantID:  info.TenantID,
		CreatedAt: info.CreatedAt,
		ExpiresAt: info.ExpiresAt,
	})
	s.auditLog(
		request,
		"admin_token_create",
		"success",
		http.StatusCreated,
		nil,
	)
}

// handleAdminToken revokes a specific token via DELETE /admin/tokens/{token}.
func (s *Server) handleAdminToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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
		"admin_token_revoke",
	)
	if !ok {
		return
	}
	release, ok := s.acquireRequestSlot(
		w,
		request,
		tenantID,
		"admin_token_revoke",
	)
	if !ok {
		return
	}
	defer release()

	token := strings.TrimPrefix(request.URL.Path, "/admin/tokens/")
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		s.auditLog(
			request,
			"admin_token_revoke",
			"error",
			http.StatusBadRequest,
			errors.New("missing token"),
		)
		return
	}
	if !s.authManager.RevokeToken(token) {
		http.Error(w, "token not found", http.StatusNotFound)
		s.auditLog(
			request,
			"admin_token_revoke",
			"error",
			http.StatusNotFound,
			errors.New("token not found"),
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	s.auditLog(
		request,
		"admin_token_revoke",
		"success",
		http.StatusNoContent,
		nil,
	)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		return
	}
}

type sseSession struct {
	id      string
	send    chan []byte
	release func()
}

// sessionStore tracks SSE sessions per tenant for isolation.
type sessionStore struct {
	mu      sync.Mutex
	items   map[string]map[string]*sseSession
	limiter *tenant.Limiter
}

func (s *sessionStore) newSession(tenantID string) (*sseSession, error) {
	release := func() {}
	if s.limiter != nil {
		var ok bool
		release, ok = s.limiter.Acquire(tenantID)
		if !ok {
			return nil, errors.New("tenant session limit exceeded")
		}
	}
	session := &sseSession{
		id:      newSessionID(),
		send:    make(chan []byte, 16),
		release: release,
	}
	s.mu.Lock()
	if s.items[tenantID] == nil {
		s.items[tenantID] = make(map[string]*sseSession)
	}
	s.items[tenantID][session.id] = session
	s.mu.Unlock()
	return session, nil
}

func (s *sessionStore) remove(tenantID, id string) {
	s.mu.Lock()
	tenantSessions, ok := s.items[tenantID]
	if !ok {
		s.mu.Unlock()
		return
	}
	session, ok := tenantSessions[id]
	if ok {
		delete(tenantSessions, id)
		close(session.send)
		if session.release != nil {
			session.release()
		}
	}
	if len(tenantSessions) == 0 {
		delete(s.items, tenantID)
	}
	s.mu.Unlock()
}

func (s *sessionStore) hasSessions(tenantID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items[tenantID]) > 0
}

func (s *sessionStore) send(tenantID, id string, payload []byte) {
	s.mu.Lock()
	tenantSessions, ok := s.items[tenantID]
	if !ok {
		s.mu.Unlock()
		return
	}
	session, ok := tenantSessions[id]
	s.mu.Unlock()
	if !ok {
		return
	}
	select {
	case session.send <- payload:
	default:
	}
}

func newSessionID() string {
	seed := make([]byte, 16)
	if _, err := rand.Read(seed); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(seed)
}

func (s *Server) validateSessionHeader(
	r *http.Request,
	tenantID string,
) (string, error) {
	sessionID := r.Header.Get(sessionHeader)
	if sessionID == "" {
		if s.sessions.hasSessions(tenantID) {
			return "", errors.New("missing Mcp-Session-Id header")
		}
		return "", nil
	}
	if !s.sessions.hasSession(tenantID, sessionID) {
		return "", errors.New("unknown Mcp-Session-Id header")
	}
	return sessionID, nil
}

func (s *sessionStore) hasSession(tenantID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tenantSessions, ok := s.items[tenantID]
	if !ok {
		return false
	}
	_, ok = tenantSessions[id]
	return ok
}

func (s *Server) requireAuth(
	w http.ResponseWriter,
	r *http.Request,
) (*http.Request, bool) {
	if s.authManager == nil {
		http.Error(w, "authentication not configured", http.StatusServiceUnavailable)
		s.auditLog(
			r,
			"auth",
			"error",
			http.StatusServiceUnavailable,
			errors.New("authentication not configured"),
		)
		return nil, false
	}
	token, err := parseBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		respondUnauthorized(w, err.Error())
		s.auditLog(r, "auth", "denied", http.StatusUnauthorized, err)
		return nil, false
	}
	info, err := s.authManager.Authenticate(token)
	if err != nil {
		respondUnauthorized(w, err.Error())
		s.auditLog(r, "auth", "denied", http.StatusUnauthorized, err)
		return nil, false
	}
	ctx := auth.WithTenantID(r.Context(), info.TenantID)
	return r.WithContext(ctx), true
}

func (s *Server) requireRateLimit(w http.ResponseWriter, r *http.Request) bool {
	if s.rateLimiter == nil {
		return true
	}
	tenantID, ok := auth.TenantIDFromContext(r.Context())
	if !ok || tenantID == "" {
		http.Error(w, "missing tenant context", http.StatusInternalServerError)
		s.auditLog(
			r,
			"rate_limit",
			"error",
			http.StatusInternalServerError,
			errors.New("missing tenant context"),
		)
		return false
	}
	decision := s.rateLimiter.Allow(tenantID)
	applyRateLimitHeaders(w, s.rateLimiter, decision)
	if !decision.Allowed {
		retryAfter := int(time.Until(decision.ResetAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		s.auditLog(
			r,
			"rate_limit",
			"denied",
			http.StatusTooManyRequests,
			errors.New("rate limit exceeded"),
		)
		return false
	}
	return true
}

func (s *Server) requireTenantContext(
	w http.ResponseWriter,
	r *http.Request,
	action string,
) (*http.Request, string, bool) {
	tenantID, ok := auth.TenantIDFromContext(r.Context())
	if !ok || tenantID == "" {
		http.Error(w, "missing tenant context", http.StatusInternalServerError)
		s.auditLog(
			r,
			action,
			"error",
			http.StatusInternalServerError,
			errors.New("missing tenant context"),
		)
		return nil, "", false
	}
	if s.workspaceManager == nil {
		http.Error(w, "tenant workspace not configured", http.StatusServiceUnavailable)
		s.auditLog(
			r,
			action,
			"error",
			http.StatusServiceUnavailable,
			errors.New("workspace manager not configured"),
		)
		return nil, "", false
	}
	workspace, err := s.workspaceManager.Workspace(tenantID)
	if err != nil {
		http.Error(w, "tenant workspace unavailable", http.StatusInternalServerError)
		s.auditLog(
			r,
			action,
			"error",
			http.StatusInternalServerError,
			err,
		)
		return nil, "", false
	}
	ctx := tenant.WithWorkspace(r.Context(), workspace)
	return r.WithContext(ctx), tenantID, true
}

func (s *Server) acquireRequestSlot(
	w http.ResponseWriter,
	r *http.Request,
	tenantID string,
	action string,
) (func(), bool) {
	if s.requestLimiter == nil {
		return func() {}, true
	}
	release, ok := s.requestLimiter.Acquire(tenantID)
	if !ok {
		http.Error(w, "tenant concurrency limit exceeded", http.StatusTooManyRequests)
		s.auditLog(
			r,
			action,
			"denied",
			http.StatusTooManyRequests,
			errors.New("tenant concurrency limit exceeded"),
		)
		return nil, false
	}
	return release, true
}

func (s *Server) requireAdmin(
	w http.ResponseWriter,
	r *http.Request,
) (*http.Request, bool) {
	if s.authManager == nil {
		http.Error(w, "authentication not configured", http.StatusServiceUnavailable)
		s.auditLog(
			r,
			"admin_auth",
			"error",
			http.StatusServiceUnavailable,
			errors.New("authentication not configured"),
		)
		return nil, false
	}
	token, err := parseBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		respondUnauthorized(w, err.Error())
		s.auditLog(r, "admin_auth", "denied", http.StatusUnauthorized, err)
		return nil, false
	}
	info, err := s.authManager.Authenticate(token)
	if err != nil {
		respondUnauthorized(w, err.Error())
		s.auditLog(r, "admin_auth", "denied", http.StatusUnauthorized, err)
		return nil, false
	}
	request := r.WithContext(auth.WithTenantID(r.Context(), info.TenantID))
	if !s.authManager.IsAdmin(token) {
		respondUnauthorized(w, "admin token required")
		s.auditLog(
			request,
			"admin_auth",
			"denied",
			http.StatusUnauthorized,
			errors.New("admin token required"),
		)
		return nil, false
	}
	return request, true
}

func parseBearerToken(header string) (string, error) {
	if header == "" {
		return "", auth.ErrMissingToken
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}
	if parts[1] == "" {
		return "", auth.ErrMissingToken
	}
	return parts[1], nil
}

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, message, http.StatusUnauthorized)
}

func (s *Server) auditLog(
	r *http.Request,
	action string,
	outcome string,
	status int,
	err error,
) {
	if s.auditLogger == nil {
		return
	}
	fields := []any{
		"audit", true,
		"action", action,
		"outcome", outcome,
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
	}
	if tenantID, ok := auth.TenantIDFromContext(r.Context()); ok {
		fields = append(fields, "tenant_id", tenantID)
	}
	if err != nil {
		fields = append(fields, "error", err.Error())
	}
	s.auditLogger.Info("audit", fields...)
}

func applyRateLimitHeaders(
	w http.ResponseWriter,
	limiter *ratelimit.Limiter,
	decision ratelimit.Decision,
) {
	if limiter == nil {
		return
	}
	w.Header().Set("RateLimit-Limit", strconv.Itoa(limiter.Limit()))
	w.Header().Set("RateLimit-Remaining", strconv.Itoa(decision.Remaining))
	w.Header().Set("RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))
}

type tokenRequest struct {
	TenantID  string     `json:"tenant_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type tokenResponse struct {
	Token     string     `json:"token"`
	TenantID  string     `json:"tenant_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func parseTokenRequest(r *http.Request) (tokenRequest, error) {
	payload, err := readJSONPayload(r, 1<<20)
	if err != nil {
		return tokenRequest{}, err
	}
	if payload.TenantID == "" {
		return tokenRequest{}, auth.ErrTenantRequired
	}
	return payload, nil
}

func readJSONPayload(r *http.Request, limit int64) (tokenRequest, error) {
	if r.Body == nil {
		return tokenRequest{}, errors.New("missing request body")
	}
	defer func() {
		_ = r.Body.Close()
	}()
	data, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		return tokenRequest{}, err
	}
	if len(data) == 0 {
		return tokenRequest{}, errors.New("empty request body")
	}
	var payload tokenRequest
	if err := json.Unmarshal(data, &payload); err != nil {
		return tokenRequest{}, err
	}
	return payload, nil
}

func writeJSON(w http.ResponseWriter, status int, payload tokenResponse) {
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		return
	}
}

func writeSSE(writer io.Writer, event, data string) error {
	if _, err := fmt.Fprintf(writer, "event: %s\n", event); err != nil {
		return err
	}
	lines := splitSSELines(data)
	for _, line := range lines {
		if _, err := fmt.Fprintf(writer, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(writer, "\n"); err != nil {
		return err
	}
	return nil
}

func splitSSELines(data string) []string {
	if data == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	for start < len(data) {
		if data[start] == '\n' {
			lines = append(lines, "")
			start++
			continue
		}
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}
		lines = append(lines, data[start:end])
		start = end
	}
	return lines
}
