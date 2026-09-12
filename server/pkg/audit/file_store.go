// Package audit provides audit logging persistence and querying.
package audit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const defaultQueryLimit = 200

// FileStore persists audit records to a JSON file.
type FileStore struct {
	path           string
	retentionDays  int
	mu             sync.Mutex
	records        []Record
	lastPruneCheck time.Time
}

type fileStorePayload struct {
	Records []Record `json:"records"`
}

// NewFileStore initializes a file-backed audit store.
func NewFileStore(path string, retentionDays int) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("audit store path must not be empty")
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil, errors.New("audit store path must include a directory")
	}
	if retentionDays < 0 {
		return nil, errors.New("audit retention days must be >= 0")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	store := &FileStore{
		path:          path,
		retentionDays: retentionDays,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// Add appends an audit record and persists it.
func (s *FileStore) Add(record Record) error {
	if s == nil {
		return errors.New("audit store not configured")
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	s.pruneLocked()
	return s.persistLocked()
}

// Query returns audit records matching the filter.
func (s *FileStore) Query(filter Filter) []Record {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	results := make([]Record, 0, limit)
	for i := len(s.records) - 1; i >= 0; i-- {
		record := s.records[i]
		if filter.TenantID != "" && record.TenantID != filter.TenantID {
			continue
		}
		if filter.Action != "" && record.Action != filter.Action {
			continue
		}
		if filter.Outcome != "" && record.Outcome != filter.Outcome {
			continue
		}
		if filter.Status != nil && record.Status != *filter.Status {
			continue
		}
		if filter.From != nil && record.Timestamp.Before(*filter.From) {
			continue
		}
		if filter.To != nil && record.Timestamp.After(*filter.To) {
			continue
		}
		results = append(results, record)
		if len(results) >= limit {
			break
		}
	}
	return results
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
	s.records = append([]Record(nil), payload.Records...)
	sort.Slice(s.records, func(i, j int) bool {
		return s.records[i].Timestamp.Before(s.records[j].Timestamp)
	})
	s.pruneLocked()
	return nil
}

func (s *FileStore) pruneLocked() {
	if s.retentionDays == 0 {
		return
	}
	now := time.Now().UTC()
	if s.lastPruneCheck.After(now.Add(-5 * time.Minute)) {
		return
	}
	s.lastPruneCheck = now
	cutoff := now.Add(-time.Duration(s.retentionDays) * 24 * time.Hour)
	if len(s.records) == 0 {
		return
	}
	idx := 0
	for idx < len(s.records) && s.records[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx > 0 {
		s.records = append([]Record(nil), s.records[idx:]...)
	}
}

func (s *FileStore) persistLocked() error {
	payload := fileStorePayload{
		Records: append([]Record(nil), s.records...),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(s.path), "audit-*.json")
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
