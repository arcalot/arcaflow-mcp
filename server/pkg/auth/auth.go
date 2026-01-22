// Package auth provides authentication helpers for server mode.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	adminTenantID = "admin"
)

type tenantKey struct{}

var (
	// ErrMissingToken indicates the request did not include a bearer token.
	ErrMissingToken = errors.New("missing bearer token")
	// ErrInvalidToken indicates the token is unknown.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken indicates the token has expired.
	ErrExpiredToken = errors.New("expired token")
	// ErrTenantRequired indicates a tenant ID was required but missing.
	ErrTenantRequired = errors.New("tenant_id is required")
)

// TokenInfo describes an issued access token.
type TokenInfo struct {
	Token     string     `json:"token"`
	TenantID  string     `json:"tenant_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Store provides token persistence for authentication.
type Store interface {
	Get(token string) (TokenInfo, bool)
	Create(tenantID string, expiresAt *time.Time) (TokenInfo, error)
	Revoke(token string) bool
	ListByTenant(tenantID string) []TokenInfo
}

// Manager handles authentication and admin token management.
type Manager struct {
	adminToken string
	store      Store
}

// NewManager builds a Manager with an admin token.
func NewManager(adminToken string, store Store) (*Manager, error) {
	if adminToken == "" {
		return nil, errors.New("admin token must not be empty")
	}
	if store == nil {
		store = NewInMemoryStore()
	}
	return &Manager{
		adminToken: adminToken,
		store:      store,
	}, nil
}

// Authenticate validates a bearer token and returns its token info.
func (m *Manager) Authenticate(token string) (TokenInfo, error) {
	if token == "" {
		return TokenInfo{}, ErrMissingToken
	}
	if m.IsAdmin(token) {
		return TokenInfo{
			Token:     token,
			TenantID:  adminTenantID,
			CreatedAt: time.Time{},
		}, nil
	}
	info, ok := m.store.Get(token)
	if !ok {
		return TokenInfo{}, ErrInvalidToken
	}
	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		return TokenInfo{}, ErrExpiredToken
	}
	return info, nil
}

// IsAdmin checks whether the token matches the configured admin token.
func (m *Manager) IsAdmin(token string) bool {
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare(
		[]byte(token),
		[]byte(m.adminToken),
	) == 1
}

// CreateToken issues a new tenant token.
func (m *Manager) CreateToken(
	tenantID string,
	expiresAt *time.Time,
) (TokenInfo, error) {
	if tenantID == "" {
		return TokenInfo{}, ErrTenantRequired
	}
	return m.store.Create(tenantID, expiresAt)
}

// RevokeToken revokes a token unless it is the admin token.
func (m *Manager) RevokeToken(token string) bool {
	if m.IsAdmin(token) {
		return false
	}
	return m.store.Revoke(token)
}

// ListTokens returns tokens scoped to a tenant.
func (m *Manager) ListTokens(tenantID string) []TokenInfo {
	if tenantID == "" {
		return nil
	}
	return m.store.ListByTenant(tenantID)
}

// InMemoryStore keeps tokens in memory for local deployments.
type InMemoryStore struct {
	mu     sync.Mutex
	tokens map[string]TokenInfo
}

// NewInMemoryStore constructs an in-memory token store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tokens: make(map[string]TokenInfo),
	}
}

// Get returns token information for a token.
func (s *InMemoryStore) Get(token string) (TokenInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.tokens[token]
	return info, ok
}

// Create generates a new token for a tenant.
func (s *InMemoryStore) Create(
	tenantID string,
	expiresAt *time.Time,
) (TokenInfo, error) {
	token, err := newToken()
	if err != nil {
		return TokenInfo{}, err
	}
	info := TokenInfo{
		Token:     token,
		TenantID:  tenantID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}
	s.mu.Lock()
	s.tokens[token] = info
	s.mu.Unlock()
	return info, nil
}

// Revoke removes a token from the store.
func (s *InMemoryStore) Revoke(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[token]; !ok {
		return false
	}
	delete(s.tokens, token)
	return true
}

// ListByTenant returns token info for a tenant.
func (s *InMemoryStore) ListByTenant(tenantID string) []TokenInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens := make([]TokenInfo, 0)
	for _, info := range s.tokens {
		if info.TenantID == tenantID {
			tokens = append(tokens, info)
		}
	}
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].CreatedAt.Equal(tokens[j].CreatedAt) {
			return tokens[i].Token < tokens[j].Token
		}
		return tokens[i].CreatedAt.Before(tokens[j].CreatedAt)
	})
	return tokens
}

func newToken() (string, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(seed), nil
}

// WithTenantID stores the tenant ID in the context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenantID)
}

// TenantIDFromContext retrieves a tenant ID from the context.
func TenantIDFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantKey{}).(string)
	return tenantID, ok
}
