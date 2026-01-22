package tenant

import (
	"strings"
	"testing"
)

func TestValidateID(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid simple", value: "tenant-a", wantErr: false},
		{name: "valid underscore", value: "tenant_a", wantErr: false},
		{name: "valid dot", value: "tenant.a", wantErr: false},
		{name: "valid leading underscore", value: "_tenant", wantErr: false},
		{name: "invalid empty", value: "", wantErr: true},
		{name: "invalid space", value: "tenant a", wantErr: true},
		{name: "invalid slash", value: "tenant/a", wantErr: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateID(testCase.value)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInMemoryStoreLifecycle(t *testing.T) {
	store := NewInMemoryStore()
	created, err := store.Create(Record{
		ID:          "tenant-a",
		DisplayName: "Tenant A",
		Metadata: map[string]string{
			"region": "us-east",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", created.ID)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be set")
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != ErrTenantExists {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	list := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(list))
	}

	fetched, ok := store.Get("tenant-a")
	if !ok {
		t.Fatalf("expected tenant to exist")
	}
	if fetched.DisplayName != "Tenant A" {
		t.Fatalf("expected display name, got %q", fetched.DisplayName)
	}
	if fetched.Metadata["region"] != "us-east" {
		t.Fatalf("expected metadata region, got %q", fetched.Metadata["region"])
	}

	newName := "Tenant Alpha"
	updated, err := store.Update("tenant-a", Update{DisplayName: &newName})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.DisplayName != "Tenant Alpha" {
		t.Fatalf("expected updated name, got %q", updated.DisplayName)
	}
	if updated.Metadata["region"] != "us-east" {
		t.Fatalf("expected metadata preserved, got %q", updated.Metadata["region"])
	}

	updatedMetadata := map[string]string{"tier": "gold"}
	updated, err = store.Update("tenant-a", Update{Metadata: &updatedMetadata})
	if err != nil {
		t.Fatalf("update metadata: %v", err)
	}
	if updated.Metadata["tier"] != "gold" {
		t.Fatalf("expected metadata tier, got %q", updated.Metadata["tier"])
	}

	if !store.Delete("tenant-a") {
		t.Fatalf("expected delete to succeed")
	}
	if _, ok := store.Get("tenant-a"); ok {
		t.Fatalf("expected tenant to be removed")
	}
}

func TestInMemoryStoreUpdateRequiresFields(t *testing.T) {
	store := NewInMemoryStore()
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.Update("tenant-a", Update{}); err != ErrTenantUpdateRequired {
		t.Fatalf("expected update required error, got %v", err)
	}
}

func TestInMemoryStoreListIDs(t *testing.T) {
	store := NewInMemoryStore()
	if _, err := store.Create(Record{ID: "tenant-b"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	ids := store.ListIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	if ids[0] != "tenant-a" || ids[1] != "tenant-b" {
		t.Fatalf("unexpected id order: %v", ids)
	}
}

func TestValidateIDFailures(t *testing.T) {
	if err := ValidateID(""); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
	if err := ValidateID(" "); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
	if err := ValidateID("bad id"); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
	longID := strings.Repeat("a", 129)
	if err := ValidateID(longID); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
}

func TestInMemoryStoreDeleteMissing(t *testing.T) {
	store := NewInMemoryStore()
	if store.Delete("missing") {
		t.Fatalf("expected delete to be false for missing tenant")
	}
}

func TestInMemoryStoreNilListGet(t *testing.T) {
	var store *InMemoryStore
	if records := store.List(); records != nil {
		t.Fatalf("expected nil records for nil store")
	}
	if _, ok := store.Get("tenant-a"); ok {
		t.Fatalf("expected get to fail for nil store")
	}
}

func TestInMemoryStoreCreateInvalidID(t *testing.T) {
	store := NewInMemoryStore()
	if _, err := store.Create(Record{ID: "bad id"}); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
}

func TestInMemoryStoreUpdateInvalidID(t *testing.T) {
	store := NewInMemoryStore()
	if _, err := store.Update("bad id", Update{DisplayName: ptr("name")}); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
}

func ptr(value string) *string {
	return &value
}
