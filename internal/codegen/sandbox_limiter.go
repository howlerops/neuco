package codegen

import (
	"errors"
	"strings"
	"sync"
)

const defaultSandboxMaxPerOrg = 5

// ErrConcurrencyLimitReached indicates an org has reached the sandbox concurrency cap.
var ErrConcurrencyLimitReached = errors.New("sandbox concurrency limit reached")

// SandboxLimiter enforces a maximum number of active sandboxes per org.
type SandboxLimiter struct {
	maxPerOrg int
	mu        sync.Mutex
	active    map[string]int
}

// NewSandboxLimiter constructs a new per-org sandbox limiter.
func NewSandboxLimiter(maxPerOrg int) *SandboxLimiter {
	limit := maxPerOrg
	if limit <= 0 {
		limit = defaultSandboxMaxPerOrg
	}

	return &SandboxLimiter{
		maxPerOrg: limit,
		active:    make(map[string]int),
	}
}

// Acquire reserves one active sandbox slot for the given org.
func (l *SandboxLimiter) Acquire(orgID string) error {
	trimmedOrgID := strings.TrimSpace(orgID)

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.active[trimmedOrgID]
	if current >= l.maxPerOrg {
		return ErrConcurrencyLimitReached
	}

	l.active[trimmedOrgID] = current + 1
	return nil
}

// Release frees one active sandbox slot for the given org.
func (l *SandboxLimiter) Release(orgID string) {
	trimmedOrgID := strings.TrimSpace(orgID)

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.active[trimmedOrgID]
	if current <= 1 {
		delete(l.active, trimmedOrgID)
		return
	}

	l.active[trimmedOrgID] = current - 1
}

// ActiveCount returns the currently tracked active sandbox count for the org.
func (l *SandboxLimiter) ActiveCount(orgID string) int {
	trimmedOrgID := strings.TrimSpace(orgID)

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.active[trimmedOrgID]
}
