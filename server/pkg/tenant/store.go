// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxTenantIDLength = 128
)

var (
	// ErrInvalidTenantID indicates the tenant ID failed validation.
	ErrInvalidTenantID = errors.New(
		"tenant id must be 1-128 chars of [A-Za-z0-9_.-]",
	)
	// ErrTenantExists indicates a tenant ID already exists.
	ErrTenantExists = errors.New("tenant already exists")
	// ErrTenantNotFound indicates a tenant record is missing.
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrTenantUpdateRequired indicates no fields were provided for update.
	ErrTenantUpdateRequired = errors.New("tenant update requires at least one field")
)

var tenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

// Record describes a tenant record managed by admin APIs.
type Record struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Update describes mutable fields for a tenant record.
type Update struct {
	DisplayName *string `json:"display_name,omitempty"`
}

// Store provides tenant record management.
type Store interface {
	List() []Record
	Get(id string) (Record, bool)
	Create(record Record) (Record, error)
	Update(id string, update Update) (Record, error)
	Delete(id string) bool
	ListIDs() []string
}

// InMemoryStore keeps tenant records in memory.
type InMemoryStore struct {
	mu      sync.Mutex
	records map[string]Record
}

// NewInMemoryStore constructs an in-memory tenant store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		records: make(map[string]Record),
	}
}

// ValidateID ensures tenant IDs meet expected constraints.
func ValidateID(tenantID string) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || len(tenantID) > maxTenantIDLength {
		return ErrInvalidTenantID
	}
	if !tenantIDPattern.MatchString(tenantID) {
		return ErrInvalidTenantID
	}
	return nil
}

// List returns a deterministic list of tenant records.
func (s *InMemoryStore) List() []Record {
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

// Get retrieves a tenant record by ID.
func (s *InMemoryStore) Get(id string) (Record, bool) {
	if s == nil {
		return Record{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[id]
	return record, ok
}

// Create adds a tenant record if it does not exist.
func (s *InMemoryStore) Create(record Record) (Record, error) {
	if s == nil {
		return Record{}, ErrTenantNotFound
	}
	if err := ValidateID(record.ID); err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.ID]; ok {
		return Record{}, ErrTenantExists
	}
	s.records[record.ID] = record
	return record, nil
}

// Update applies updates to an existing tenant record.
func (s *InMemoryStore) Update(id string, update Update) (Record, error) {
	if s == nil {
		return Record{}, ErrTenantNotFound
	}
	if err := ValidateID(id); err != nil {
		return Record{}, err
	}
	if update.DisplayName == nil {
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
	record.UpdatedAt = time.Now().UTC()
	s.records[id] = record
	return record, nil
}

// Delete removes a tenant record by ID.
func (s *InMemoryStore) Delete(id string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[id]; !ok {
		return false
	}
	delete(s.records, id)
	return true
}

// ListIDs returns tenant IDs in sorted order.
func (s *InMemoryStore) ListIDs() []string {
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
