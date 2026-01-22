package state

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
)

func TestManagerSetGetIsolation(t *testing.T) {
	t.Parallel()
	manager := NewManager(10 * time.Minute)

	ctxA := auth.WithTenantID(context.Background(), "tenant-a")
	ctxB := auth.WithTenantID(context.Background(), "tenant-b")

	payload := json.RawMessage(`{"value":"test"}`)
	err := manager.Set(ctxA, SessionData{
		SessionID:  "session-1",
		WorkflowID: "workflow-a",
		DraftInput: payload,
	})
	if err != nil {
		t.Fatalf("set session: %v", err)
	}

	_, ok, err := manager.Get(ctxB, "session-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if ok {
		t.Fatalf("expected session to be isolated by tenant")
	}

	data, ok, err := manager.Get(ctxA, "session-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if !ok {
		t.Fatalf("expected session to exist")
	}
	if data.WorkflowID != "workflow-a" {
		t.Fatalf("unexpected workflow id: %s", data.WorkflowID)
	}
}

func TestManagerCleanupExpired(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)
	manager := NewManager(5 * time.Minute)
	manager.WithClock(func() time.Time { return now })

	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	err := manager.Set(ctx, SessionData{
		SessionID: "session-expired",
		ExpiresAt: now.Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("set session: %v", err)
	}
	removed := manager.CleanupExpired()
	if removed != 1 {
		t.Fatalf("expected 1 session removed, got %d", removed)
	}
}

func TestManagerDelete(t *testing.T) {
	t.Parallel()
	manager := NewManager(time.Minute)

	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	err := manager.Set(ctx, SessionData{
		SessionID: "session-a",
	})
	if err != nil {
		t.Fatalf("set session: %v", err)
	}
	if err := manager.Delete(ctx, "session-a"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, ok, err := manager.Get(ctx, "session-a"); err != nil || ok {
		t.Fatalf("expected session deleted")
	}
}

func TestManagerRequiresTenant(t *testing.T) {
	t.Parallel()
	manager := NewManager(10 * time.Minute)

	err := manager.Set(context.Background(), SessionData{SessionID: "s"})
	if err != ErrTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
	_, _, err = manager.Get(context.Background(), "s")
	if err != ErrTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
	err = manager.Delete(context.Background(), "s")
	if err != ErrTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
}
