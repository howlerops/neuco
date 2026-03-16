package codegen

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const defaultSandboxCleanupInterval = 2 * time.Minute

// SandboxCleaner periodically destroys expired sandboxes.
type SandboxCleaner struct {
	manager  SandboxManager
	interval time.Duration

	mu      sync.Mutex
	tracked map[string]time.Time
	stopCh  chan struct{}
	stopped bool
}

// NewSandboxCleaner constructs a sandbox cleaner with a periodic cleanup interval.
func NewSandboxCleaner(manager SandboxManager, interval time.Duration) *SandboxCleaner {
	cleanupInterval := interval
	if cleanupInterval <= 0 {
		cleanupInterval = defaultSandboxCleanupInterval
	}

	return &SandboxCleaner{
		manager:  manager,
		interval: cleanupInterval,
		tracked:  make(map[string]time.Time),
		stopCh:   make(chan struct{}),
	}
}

// Track registers a sandbox expiration for background cleanup.
func (c *SandboxCleaner) Track(sandboxID string, expiresAt time.Time) {
	trimmedID := strings.TrimSpace(sandboxID)
	if trimmedID == "" {
		return
	}

	c.mu.Lock()
	c.tracked[trimmedID] = expiresAt
	c.mu.Unlock()
}

// Untrack removes a sandbox from background cleanup tracking.
func (c *SandboxCleaner) Untrack(sandboxID string) {
	trimmedID := strings.TrimSpace(sandboxID)
	if trimmedID == "" {
		return
	}

	c.mu.Lock()
	delete(c.tracked, trimmedID)
	c.mu.Unlock()
}

// Start runs a blocking cleanup loop until context cancellation or Stop is called.
func (c *SandboxCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("sandbox cleaner stopped: context cancelled", "error", ctx.Err())
			return
		case <-c.stopCh:
			slog.Debug("sandbox cleaner stopped")
			return
		case <-ticker.C:
			c.cleanupExpired(ctx)
		}
	}
}

// Stop terminates the cleaner loop.
func (c *SandboxCleaner) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}

	close(c.stopCh)
	c.stopped = true
}

func (c *SandboxCleaner) cleanupExpired(ctx context.Context) {
	if c.manager == nil {
		slog.Warn("sandbox cleaner skipped: manager is nil")
		return
	}

	now := time.Now().UTC()
	expired := make([]string, 0)

	c.mu.Lock()
	for sandboxID, expiresAt := range c.tracked {
		if !expiresAt.IsZero() && !expiresAt.After(now) {
			expired = append(expired, sandboxID)
		}
	}
	c.mu.Unlock()

	for _, sandboxID := range expired {
		if err := c.manager.Destroy(ctx, sandboxID); err != nil {
			slog.Warn("sandbox cleaner failed to destroy expired sandbox", "sandbox_id", sandboxID, "error", err)
			continue
		}

		c.Untrack(sandboxID)
		slog.Debug("sandbox cleaner destroyed expired sandbox", "sandbox_id", sandboxID)
	}
}
