package tenant

import "testing"

func TestQuotaWorkspaceExceeded(t *testing.T) {
	quota := Quota{MaxWorkspaceBytes: 10}
	if err := quota.Check(11, 0); err != ErrWorkspaceQuotaExceeded {
		t.Fatalf("expected workspace quota exceeded, got %v", err)
	}
}

func TestQuotaRequestExceeded(t *testing.T) {
	quota := Quota{MaxRequests: 2}
	if err := quota.Check(0, 2); err != ErrRequestQuotaExceeded {
		t.Fatalf("expected request quota exceeded, got %v", err)
	}
}

func TestQuotaWithinLimits(t *testing.T) {
	quota := Quota{MaxWorkspaceBytes: 10, MaxRequests: 5}
	if err := quota.Check(5, 4); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
