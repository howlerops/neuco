package codegen

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ParallelSandboxConfig configures parallel sandbox execution.
type ParallelSandboxConfig struct {
	MaxSandboxes int
	Timeout      time.Duration
}

// SandboxResult captures a single sandbox execution outcome.
type SandboxResult struct {
	SandboxID   string
	Provider    string
	Success     bool
	FileChanges []FileChange
	Duration    time.Duration
	Error       error
}

// ParallelSandboxRunner runs multiple sandbox sessions concurrently.
type ParallelSandboxRunner struct {
	manager SandboxManager
	config  ParallelSandboxConfig
}

func NewParallelSandboxRunner(manager SandboxManager, config ParallelSandboxConfig) *ParallelSandboxRunner {
	if config.MaxSandboxes <= 0 {
		config.MaxSandboxes = 3
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Minute
	}
	return &ParallelSandboxRunner{
		manager: manager,
		config:  config,
	}
}

// Run executes the given function in parallel sandboxes, each provisioned from the base config.
// Returns all results including failures.
func (r *ParallelSandboxRunner) Run(ctx context.Context, configs []SandboxConfig, fn func(ctx context.Context, sb *Sandbox) (*SandboxResult, error)) []SandboxResult {
	ctx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	limit := r.config.MaxSandboxes
	if len(configs) < limit {
		limit = len(configs)
	}

	sem := make(chan struct{}, limit)
	var mu sync.Mutex
	results := make([]SandboxResult, 0, len(configs))
	var wg sync.WaitGroup

	for _, cfg := range configs {
		wg.Add(1)
		go func(sandboxCfg SandboxConfig) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			sb, err := r.manager.Provision(ctx, sandboxCfg)
			if err != nil {
				mu.Lock()
				results = append(results, SandboxResult{
					Provider: sandboxCfg.GenerationID,
					Error:    fmt.Errorf("provision sandbox: %w", err),
				})
				mu.Unlock()
				return
			}

			defer func() {
				if destroyErr := r.manager.Destroy(ctx, sb.ID); destroyErr != nil {
					slog.Warn("parallel sandbox: failed to destroy", "sandbox_id", sb.ID, "error", destroyErr)
				}
			}()

			result, err := fn(ctx, sb)
			if err != nil {
				mu.Lock()
				results = append(results, SandboxResult{
					SandboxID: sb.ID,
					Error:     err,
				})
				mu.Unlock()
				return
			}

			if result != nil {
				result.SandboxID = sb.ID
			}

			mu.Lock()
			if result != nil {
				results = append(results, *result)
			}
			mu.Unlock()
		}(cfg)
	}

	wg.Wait()
	return results
}

// BestResult returns the most successful result based on file change count and success status.
func (r *ParallelSandboxRunner) BestResult(results []SandboxResult) (*SandboxResult, error) {
	var best *SandboxResult
	for i := range results {
		res := &results[i]
		if !res.Success || res.Error != nil {
			continue
		}
		if best == nil || len(res.FileChanges) > len(best.FileChanges) {
			best = res
		}
	}

	if best == nil {
		return nil, fmt.Errorf("parallel sandbox: no successful results from %d runs", len(results))
	}
	return best, nil
}
