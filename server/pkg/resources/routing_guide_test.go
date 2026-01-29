package resources

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRoutingGuideProviderList(t *testing.T) {
	t.Parallel()

	provider := NewRoutingGuideProvider(slog.Default())
	items, errObj := provider.List(context.Background())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(items))
	}
	if items[0].URI != routingGuideURI {
		t.Fatalf("expected routing guide URI, got %s", items[0].URI)
	}
	if items[0].MimeType != "text/markdown" {
		t.Fatalf("expected markdown MIME type")
	}
}

func TestRoutingGuideProviderRead(t *testing.T) {
	t.Parallel()

	provider := NewRoutingGuideProvider(slog.Default())
	content, handled, errObj := provider.Read(context.Background(), routingGuideURI)
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	if !handled {
		t.Fatalf("expected routing guide to be handled")
	}
	if content.URI != routingGuideURI {
		t.Fatalf("expected routing guide URI")
	}
	if content.MimeType != "text/markdown" {
		t.Fatalf("expected markdown MIME type")
	}
	if !strings.Contains(content.Text, "workflow_input_recommend") {
		t.Fatalf("expected routing guide to mention workflow_input_recommend")
	}
	if !strings.Contains(content.Text, "DO NOT") {
		t.Fatalf("expected routing guide to include negative guidance")
	}
}

func TestRoutingGuideProviderRejectsUnknownURI(t *testing.T) {
	t.Parallel()

	provider := NewRoutingGuideProvider(slog.Default())
	_, handled, errObj := provider.Read(context.Background(), "mcp://unknown")
	if errObj != nil {
		t.Fatalf("expected no error for unknown URI, got %v", errObj)
	}
	if handled {
		t.Fatalf("expected unknown URI to not be handled")
	}
}
