package codegen

import (
	"errors"
	"testing"
)

func TestSandboxLimiter_AcquireReleaseAndActiveCount(t *testing.T) {
	t.Parallel()

	limiter := NewSandboxLimiter(2)
	orgID := "org-1"

	if got := limiter.ActiveCount(orgID); got != 0 {
		t.Fatalf("expected initial active count 0, got %d", got)
	}

	if err := limiter.Acquire(orgID); err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}
	if got := limiter.ActiveCount(orgID); got != 1 {
		t.Fatalf("expected active count 1 after first acquire, got %d", got)
	}

	if err := limiter.Acquire(orgID); err != nil {
		t.Fatalf("second acquire returned error: %v", err)
	}
	if got := limiter.ActiveCount(orgID); got != 2 {
		t.Fatalf("expected active count 2 after second acquire, got %d", got)
	}

	err := limiter.Acquire(orgID)
	if !errors.Is(err, ErrConcurrencyLimitReached) {
		t.Fatalf("expected ErrConcurrencyLimitReached, got %v", err)
	}

	limiter.Release(orgID)
	if got := limiter.ActiveCount(orgID); got != 1 {
		t.Fatalf("expected active count 1 after release, got %d", got)
	}

	limiter.Release(orgID)
	if got := limiter.ActiveCount(orgID); got != 0 {
		t.Fatalf("expected active count 0 after final release, got %d", got)
	}
}

func TestSandboxLimiter_EmptyOrgIDHandling(t *testing.T) {
	t.Parallel()

	limiter := NewSandboxLimiter(2)

	if err := limiter.Acquire(""); err != nil {
		t.Fatalf("acquire empty org id returned error: %v", err)
	}

	if err := limiter.Acquire("   "); err != nil {
		t.Fatalf("acquire whitespace org id returned error: %v", err)
	}

	if got := limiter.ActiveCount(""); got != 2 {
		t.Fatalf("expected empty/whitespace org IDs to map to same key with count 2, got %d", got)
	}

	err := limiter.Acquire("\t")
	if !errors.Is(err, ErrConcurrencyLimitReached) {
		t.Fatalf("expected limit reached for empty org ID bucket, got %v", err)
	}

	limiter.Release("   ")
	if got := limiter.ActiveCount(""); got != 1 {
		t.Fatalf("expected count 1 after release for whitespace key, got %d", got)
	}
}
