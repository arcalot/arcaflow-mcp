// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileStore persists tenant records to a JSON file.
type FileStore struct {
	path    string
	mu      sync.Mutex
	records map[string]Record
}

type fileStorePayload struct {
	Tenants []Record `json:"tenants"`
}

// NewFileStore initializes a file-backed tenant store.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("tenant store path must not be empty")
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil, errors.New("tenant store path must include a directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	store := &FileStore{
		path:    path,
		records: make(map[string]Record),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// List returns a deterministic list of tenant records.
func (s *FileStore) List() []Record {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].ID < records[j].ID
	})
	return records
}

// ListIDs returns tenant IDs in sorted order.
func (s *FileStore) ListIDs() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Get retrieves a tenant record by ID.
func (s *FileStore) Get(id string) (Record, bool) {
	if s == nil {
		return Record{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	return record, ok
}

// Create adds a tenant record if it does not exist.
func (s *FileStore) Create(record Record) (Record, error) {
	if s == nil {
		return Record{}, ErrTenantNotFound
	}
	if err := ValidateID(record.ID); err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now
	record.Metadata = cloneMetadata(record.Metadata)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.ID]; ok {
		return Record{}, ErrTenantExists
	}
	s.records[record.ID] = record
	if err := s.persistLocked(); err != nil {
		delete(s.records, record.ID)
		return Record{}, err
	}
	return record, nil
}

// Update applies updates to an existing tenant record.
func (s *FileStore) Update(id string, update Update) (Record, error) {
	if s == nil {
		return Record{}, ErrTenantNotFound
	}
	if err := ValidateID(id); err != nil {
		return Record{}, err
	}
	if update.DisplayName == nil && update.Metadata == nil {
		return Record{}, ErrTenantUpdateRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	if !ok {
		return Record{}, ErrTenantNotFound
	}
	if update.DisplayName != nil {
		record.DisplayName = strings.TrimSpace(*update.DisplayName)
	}
	if update.Metadata != nil {
		record.Metadata = cloneMetadata(*update.Metadata)
	}
	record.UpdatedAt = time.Now().UTC()
	s.records[id] = record
	if err := s.persistLocked(); err != nil {
		return Record{}, err
	}
	return record, nil
}

// Delete removes a tenant record by ID.
func (s *FileStore) Delete(id string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	if !ok {
		return false
	}
	delete(s.records, id)
	if err := s.persistLocked(); err != nil {
		s.records[id] = record
		return false
	}
	return true
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
	for _, record := range payload.Tenants {
		if record.ID == "" {
			continue
		}
		s.records[record.ID] = record
	}
	return nil
}

func (s *FileStore) persistLocked() error {
	payload := fileStorePayload{
		Tenants: make([]Record, 0, len(s.records)),
	}
	for _, record := range s.records {
		payload.Tenants = append(payload.Tenants, record)
	}
	sort.Slice(payload.Tenants, func(i, j int) bool {
		if payload.Tenants[i].CreatedAt.Equal(payload.Tenants[j].CreatedAt) {
			return payload.Tenants[i].ID < payload.Tenants[j].ID
		}
		return payload.Tenants[i].CreatedAt.Before(payload.Tenants[j].CreatedAt)
	})
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(s.path), "tenants-*.json")
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
