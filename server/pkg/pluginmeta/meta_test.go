package pluginmeta

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// validYAML is a minimal config used across tests.
const validYAML = `
plugins:
  arcaflow-plugin-fio:
    keywords: [fio, storage, io, disk]
    category: storage
  arcaflow-plugin-pcp:
    keywords: [pcp, metrics, monitoring]
    category: monitoring
`

func TestLookupKnownPlugin(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte(validYAML))

	tests := []struct {
		name     string
		repo     string
		wantCat  string
		wantKeys []string
	}{
		{
			name:     "fio plugin",
			repo:     "arcaflow-plugin-fio",
			wantCat:  "storage",
			wantKeys: []string{
				"fio", "storage", "io", "disk",
			},
		},
		{
			name:     "pcp plugin",
			repo:     "arcaflow-plugin-pcp",
			wantCat:  "monitoring",
			wantKeys: []string{
				"pcp", "metrics", "monitoring",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := cat.Lookup(tc.repo)
			if e.Category != tc.wantCat {
				t.Errorf(
					"category = %q, want %q",
					e.Category, tc.wantCat,
				)
			}
			if !reflect.DeepEqual(
				e.Keywords, tc.wantKeys,
			) {
				t.Errorf(
					"keywords = %v, want %v",
					e.Keywords, tc.wantKeys,
				)
			}
		})
	}
}

func TestLookupUnknownPlugin(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte(validYAML))

	e := cat.Lookup("arcaflow-plugin-banana")
	if e.Category != "other" {
		t.Errorf("category = %q, want other", e.Category)
	}
	want := []string{"banana"}
	if !reflect.DeepEqual(e.Keywords, want) {
		t.Errorf("keywords = %v, want %v", e.Keywords, want)
	}
}

func TestLookupInference(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte(validYAML))

	e := cat.Lookup(
		"arcaflow-plugin-network-observability",
	)
	if e.Category != "other" {
		t.Errorf("category = %q, want other", e.Category)
	}
	want := []string{"network", "observability"}
	if !reflect.DeepEqual(e.Keywords, want) {
		t.Errorf("keywords = %v, want %v", e.Keywords, want)
	}
}

func TestNewCatalogFromBytes(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte(validYAML))

	if cat == nil {
		t.Fatal("catalog is nil")
	}
	if len(cat.plugins) != 2 {
		t.Errorf(
			"plugins count = %d, want 2",
			len(cat.plugins),
		)
	}
}

func TestNewCatalogFromBytesInvalid(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte("{{invalid"))
	if cat == nil {
		t.Fatal("catalog is nil")
	}
	if len(cat.plugins) != 0 {
		t.Errorf(
			"plugins count = %d, want 0",
			len(cat.plugins),
		)
	}
}

func TestNewCatalogMissingFile(t *testing.T) {
	t.Parallel()
	cat := NewCatalog("/nonexistent/path/config.yaml")
	if cat == nil {
		t.Fatal("catalog is nil")
	}
	if len(cat.plugins) != 0 {
		t.Errorf(
			"plugins count = %d, want 0",
			len(cat.plugins),
		)
	}

	// Should still infer from name.
	e := cat.Lookup("arcaflow-plugin-fio")
	if e.Category != "other" {
		t.Errorf("category = %q, want other", e.Category)
	}
	want := []string{"fio"}
	if !reflect.DeepEqual(e.Keywords, want) {
		t.Errorf("keywords = %v, want %v", e.Keywords, want)
	}
}

func TestLookupEmptyCatalog(t *testing.T) {
	t.Parallel()
	cat := NewCatalogFromBytes([]byte(""))
	if cat == nil {
		t.Fatal("catalog is nil")
	}

	e := cat.Lookup("arcaflow-plugin-stress-test")
	if e.Category != "other" {
		t.Errorf("category = %q, want other", e.Category)
	}
	want := []string{"stress", "test"}
	if !reflect.DeepEqual(e.Keywords, want) {
		t.Errorf("keywords = %v, want %v", e.Keywords, want)
	}
}

func TestNewCatalogFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "meta.yaml")
	if err := os.WriteFile(
		p, []byte(validYAML), 0o600,
	); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cat := NewCatalog(p)
	if cat == nil {
		t.Fatal("catalog is nil")
	}
	if len(cat.plugins) != 2 {
		t.Errorf(
			"plugins count = %d, want 2",
			len(cat.plugins),
		)
	}

	e := cat.Lookup("arcaflow-plugin-fio")
	if e.Category != "storage" {
		t.Errorf(
			"category = %q, want storage",
			e.Category,
		)
	}
}
