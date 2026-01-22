// Package auth provides authentication helpers for server mode.
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileStore persists tokens to a JSON file.
type FileStore struct {
	path   string
	mu     sync.Mutex
	tokens map[string]TokenInfo
}

type fileStorePayload struct {
	Tokens []TokenInfo `json:"tokens"`
}

// NewFileStore initializes a file-backed token store.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("token store path must not be empty")
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil, errors.New("token store path must include a directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	store := &FileStore{
		path:   path,
		tokens: make(map[string]TokenInfo),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// Get returns token information for a token.
func (s *FileStore) Get(token string) (TokenInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.tokens[token]
	return info, ok
}

// Create generates a new token for a tenant.
func (s *FileStore) Create(
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
	defer s.mu.Unlock()
	s.tokens[token] = info
	if err := s.persistLocked(); err != nil {
		delete(s.tokens, token)
		return TokenInfo{}, err
	}
	return info, nil
}

// Revoke removes a token from the store.
func (s *FileStore) Revoke(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.tokens[token]
	if !ok {
		return false
	}
	delete(s.tokens, token)
	if err := s.persistLocked(); err != nil {
		s.tokens[token] = info
		return false
	}
	return true
}

// ListByTenant returns token info for a tenant.
func (s *FileStore) ListByTenant(tenantID string) []TokenInfo {
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

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.persistLocked()
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var payload fileStorePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for _, token := range payload.Tokens {
		if token.Token == "" {
			continue
		}
		s.tokens[token.Token] = token
	}
	return nil
}

func (s *FileStore) persistLocked() error {
	payload := fileStorePayload{
		Tokens: make([]TokenInfo, 0, len(s.tokens)),
	}
	for _, info := range s.tokens {
		payload.Tokens = append(payload.Tokens, info)
	}
	sort.Slice(payload.Tokens, func(i, j int) bool {
		if payload.Tokens[i].CreatedAt.Equal(payload.Tokens[j].CreatedAt) {
			return payload.Tokens[i].Token < payload.Tokens[j].Token
		}
		return payload.Tokens[i].CreatedAt.Before(payload.Tokens[j].CreatedAt)
	})
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(s.path), "tokens-*.json")
	if err != nil {
		return err
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	if err := tempFile.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempFile.Name(), s.path)
}
