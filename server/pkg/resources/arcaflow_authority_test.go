package resources

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestArcaflowAuthorityProviderList(t *testing.T) {
	t.Parallel()

	provider := NewArcaflowAuthorityProvider(slog.Default())
	items, errObj := provider.List(context.Background())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(items))
	}
	if items[0].URI != arcaflowAuthorityURI {
		t.Fatalf("expected authority URI, got %s", items[0].URI)
	}
	if items[0].MimeType != "text/markdown" {
		t.Fatalf("expected markdown MIME type")
	}
	if !strings.Contains(items[0].Description, "training data") {
		t.Fatalf("expected description to mention training data")
	}
}

func TestArcaflowAuthorityProviderRead(t *testing.T) {
	t.Parallel()

	provider := NewArcaflowAuthorityProvider(slog.Default())
	content, handled, errObj := provider.Read(
		context.Background(),
		arcaflowAuthorityURI,
	)
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	if !handled {
		t.Fatalf("expected authority resource to be handled")
	}
	if content.URI != arcaflowAuthorityURI {
		t.Fatalf("expected authority URI")
	}
	if content.MimeType != "text/markdown" {
		t.Fatalf("expected markdown MIME type")
	}

	// Verify key uncertainty injection content
	if !strings.Contains(content.Text, "training data cutoff") {
		t.Fatalf("expected content to mention training data cutoff")
	}
	if !strings.Contains(content.Text, "Arcaflow v0.8+") {
		t.Fatalf("expected content to mention version changes")
	}
	if !strings.Contains(content.Text, "Dynamic Schema Resolution") {
		t.Fatalf("expected content to explain dynamic resolution")
	}
	if !strings.Contains(content.Text, "Do Not Trust Training Data") {
		t.Fatalf("expected explicit warning about training data")
	}
}

func TestArcaflowAuthorityProviderRejectsUnknownURI(t *testing.T) {
	t.Parallel()

	provider := NewArcaflowAuthorityProvider(slog.Default())
	_, handled, errObj := provider.Read(context.Background(), "mcp://unknown")
	if errObj != nil {
		t.Fatalf("expected no error for unknown URI, got %v", errObj)
	}
	if handled {
		t.Fatalf("expected unknown URI to not be handled")
	}
}
