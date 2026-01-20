// Package httpserver implements MCP HTTP/SSE transport scaffolding.
package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
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
	logger   *slog.Logger
	server   *http.Server
	handler  *protocol.Server
	sessions *sessionStore
}

// Config holds HTTP server configuration for MCP transport.
type Config struct {
	Address string
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
		logger:  logger,
		handler: handler,
		sessions: &sessionStore{
			items: make(map[string]*sseSession),
		},
	}
	mux.HandleFunc("/mcp", server.handleMCPPost)
	mux.HandleFunc("/mcp/events", server.handleMCPSSE)
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

	sessionID, err := s.validateSessionHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	payload, err := s.readPayload(w, r)
	if err != nil {
		s.logger.Warn("failed to read MCP request", "error", err)
		return
	}

	responses, err := s.handler.Handle(r.Context(), payload)
	if err != nil {
		s.logger.Error("protocol handler error", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if sessionID != "" {
		s.sessions.send(sessionID, responses[0])
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(responses[0]); err != nil {
		return
	}
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
	if r.Context().Err() != nil {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	session := s.sessions.newSession()
	defer s.sessions.remove(session.id)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set(sessionHeader, session.id)

	if err := writeSSE(w, "session", session.id); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
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
	id   string
	send chan []byte
}

type sessionStore struct {
	mu    sync.Mutex
	items map[string]*sseSession
}

func (s *sessionStore) newSession() *sseSession {
	session := &sseSession{
		id:   newSessionID(),
		send: make(chan []byte, 16),
	}
	s.mu.Lock()
	s.items[session.id] = session
	s.mu.Unlock()
	return session
}

func (s *sessionStore) remove(id string) {
	s.mu.Lock()
	session, ok := s.items[id]
	if ok {
		delete(s.items, id)
		close(session.send)
	}
	s.mu.Unlock()
}

func (s *sessionStore) hasSessions() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) > 0
}

func (s *sessionStore) send(id string, payload []byte) {
	s.mu.Lock()
	session, ok := s.items[id]
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

func (s *Server) validateSessionHeader(r *http.Request) (string, error) {
	sessionID := r.Header.Get(sessionHeader)
	if sessionID == "" {
		if s.sessions.hasSessions() {
			return "", errors.New("missing Mcp-Session-Id header")
		}
		return "", nil
	}
	if !s.sessions.hasSession(sessionID) {
		return "", errors.New("unknown Mcp-Session-Id header")
	}
	return sessionID, nil
}

func (s *sessionStore) hasSession(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[id]
	return ok
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
