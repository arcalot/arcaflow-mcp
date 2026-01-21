package auth

import (
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
