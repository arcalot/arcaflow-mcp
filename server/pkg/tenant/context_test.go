package tenant

import (
	"context"
	"testing"
)

func TestWorkspaceContextRoundTrip(t *testing.T) {
	ctx := WithWorkspace(context.Background(), "/tmp/workspace")
	workspace, ok := WorkspaceFromContext(ctx)
	if !ok || workspace != "/tmp/workspace" {
		t.Fatalf("expected workspace in context")
	}
}
