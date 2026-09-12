package auth

import (
	"context"
	"testing"
	"time"
)

func TestManagerAuthenticateAdminToken(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	info, err := manager.Authenticate("admin-token")
	if err != nil {
		t.Fatalf("expected admin token to authenticate, got %v", err)
	}
	if info.TenantID != adminTenantID {
		t.Fatalf("expected admin tenant, got %q", info.TenantID)
	}
}

func TestManagerAuthenticateUnknownToken(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	_, err = manager.Authenticate("missing")
	if err != ErrInvalidToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestManagerCreateAndRevokeToken(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	info, err := manager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if _, err := manager.Authenticate(info.Token); err != nil {
		t.Fatalf("expected token to authenticate, got %v", err)
	}

	if !manager.RevokeToken(info.Token) {
		t.Fatalf("expected token to be revoked")
	}

	if _, err := manager.Authenticate(info.Token); err != ErrInvalidToken {
		t.Fatalf("expected invalid token after revoke, got %v", err)
	}
}

func TestManagerExpiredToken(t *testing.T) {
	store := NewInMemoryStore()
	manager, err := NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	expiresAt := time.Now().Add(-1 * time.Minute)
	info, err := manager.CreateToken("tenant-a", &expiresAt)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if _, err := manager.Authenticate(info.Token); err != ErrExpiredToken {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestManagerCreateTokenMissingTenant(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if _, err := manager.CreateToken("", nil); err != ErrTenantRequired {
		t.Fatalf("expected tenant required error, got %v", err)
	}
}

func TestManagerRevokeAdminToken(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if manager.RevokeToken("admin-token") {
		t.Fatalf("expected admin token not to be revoked")
	}
}

func TestManagerIsAdminFalse(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if manager.IsAdmin("other-token") {
		t.Fatalf("expected token to not be admin")
	}
}

func TestManagerIsAdminEmptyToken(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if manager.IsAdmin("") {
		t.Fatalf("expected empty token to not be admin")
	}
}

func TestInMemoryStoreRevokeMissing(t *testing.T) {
	store := NewInMemoryStore()
	if store.Revoke("missing") {
		t.Fatalf("expected revoke to be false for missing token")
	}
}

func TestInMemoryStoreListByTenantSorted(t *testing.T) {
	store := NewInMemoryStore()
	first, err := store.Create("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	second, err := store.Create("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	tokens := store.ListByTenant("tenant-a")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Token != first.Token && tokens[1].Token != first.Token {
		t.Fatalf("expected first token in list")
	}
	if tokens[0].Token != second.Token && tokens[1].Token != second.Token {
		t.Fatalf("expected second token in list")
	}
}

func TestManagerListTokensEmptyTenant(t *testing.T) {
	manager, err := NewManager("admin-token", nil)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if tokens := manager.ListTokens(""); tokens != nil {
		t.Fatalf("expected nil tokens for empty tenant")
	}
}

func TestTenantIDContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := TenantIDFromContext(ctx); ok {
		t.Fatalf("expected no tenant id")
	}
	ctx = WithTenantID(ctx, "tenant-a")
	tenantID, ok := TenantIDFromContext(ctx)
	if !ok || tenantID != "tenant-a" {
		t.Fatalf("unexpected tenant id: %v %v", tenantID, ok)
	}
}

func TestManagerListTokens(t *testing.T) {
	store := NewInMemoryStore()
	manager, err := NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	tokenA, err := manager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, err := manager.CreateToken("tenant-b", nil); err != nil {
		t.Fatalf("create token: %v", err)
	}
	tokenA2, err := manager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	tokens := manager.ListTokens("tenant-a")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Token != tokenA.Token && tokens[1].Token != tokenA.Token {
		t.Fatalf("expected tenant-a token in list")
	}
	if tokens[0].Token != tokenA2.Token && tokens[1].Token != tokenA2.Token {
		t.Fatalf("expected second tenant-a token in list")
	}
}
