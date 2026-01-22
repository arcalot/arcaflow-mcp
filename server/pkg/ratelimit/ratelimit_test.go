package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinWindow(t *testing.T) {
	base := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	current := base
	limiter, err := NewLimiter(Config{
		Limit:          2,
		Window:         time.Minute,
		BackoffEnabled: true,
		BackoffBase:    time.Second,
		BackoffMax:     8 * time.Second,
		Clock: func() time.Time {
			return current
		},
	})
	if err != nil {
		t.Fatalf("create limiter: %v", err)
	}

	first := limiter.Allow("tenant-a")
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("expected first request allowed, got %#v", first)
	}

	second := limiter.Allow("tenant-a")
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("expected second request allowed, got %#v", second)
	}

	third := limiter.Allow("tenant-a")
	if third.Allowed {
		t.Fatalf("expected third request rejected")
	}
	if third.ResetAt != base.Add(time.Second) {
		t.Fatalf("expected backoff reset: %v", third.ResetAt)
	}
}

func TestLimiterResetsWindow(t *testing.T) {
	base := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	current := base
	limiter, err := NewLimiter(Config{
		Limit:          1,
		Window:         time.Minute,
		BackoffEnabled: true,
		BackoffBase:    time.Second,
		BackoffMax:     8 * time.Second,
		Clock: func() time.Time {
			return current
		},
	})
	if err != nil {
		t.Fatalf("create limiter: %v", err)
	}

	if !limiter.Allow("tenant-a").Allowed {
		t.Fatalf("expected first request allowed")
	}

	current = base.Add(2 * time.Minute)
	if !limiter.Allow("tenant-a").Allowed {
		t.Fatalf("expected request after window reset allowed")
	}
}

func TestLimiterProgressiveBackoff(t *testing.T) {
	base := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	current := base
	limiter, err := NewLimiter(Config{
		Limit:          1,
		Window:         time.Minute,
		BackoffEnabled: true,
		BackoffBase:    time.Second,
		BackoffMax:     4 * time.Second,
		Clock: func() time.Time {
			return current
		},
	})
	if err != nil {
		t.Fatalf("create limiter: %v", err)
	}

	if !limiter.Allow("tenant-a").Allowed {
		t.Fatalf("expected first request allowed")
	}

	second := limiter.Allow("tenant-a")
	if second.Allowed {
		t.Fatalf("expected second request rejected")
	}
	if second.ResetAt != base.Add(time.Second) {
		t.Fatalf("expected 1s backoff, got %v", second.ResetAt.Sub(base))
	}

	third := limiter.Allow("tenant-a")
	if third.Allowed {
		t.Fatalf("expected third request rejected")
	}
	if third.ResetAt != base.Add(2*time.Second) {
		t.Fatalf("expected 2s backoff, got %v", third.ResetAt.Sub(base))
	}

	fourth := limiter.Allow("tenant-a")
	if fourth.Allowed {
		t.Fatalf("expected fourth request rejected")
	}
	if fourth.ResetAt != base.Add(4*time.Second) {
		t.Fatalf("expected 4s backoff, got %v", fourth.ResetAt.Sub(base))
	}

	fifth := limiter.Allow("tenant-a")
	if fifth.Allowed {
		t.Fatalf("expected fifth request rejected")
	}
	if fifth.ResetAt != base.Add(4*time.Second) {
		t.Fatalf("expected backoff capped at 4s, got %v", fifth.ResetAt.Sub(base))
	}
}

func TestLimiterAccessors(t *testing.T) {
	limiter, err := NewLimiter(Config{
		Limit:  5,
		Window: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("create limiter: %v", err)
	}
	if limiter.Limit() != 5 {
		t.Fatalf("expected limit 5")
	}
	if limiter.Window() != 2*time.Minute {
		t.Fatalf("expected window 2m")
	}
}
