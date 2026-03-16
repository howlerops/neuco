package codegen

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MultiAgentStrategy controls how multiple agent results are combined.
type MultiAgentStrategy string

const (
	// StrategyRace runs all agents in parallel and uses the first successful result.
	StrategyRace MultiAgentStrategy = "race"
	// StrategyBestOf runs all agents in parallel and selects the result with the best validation score.
	StrategyBestOf MultiAgentStrategy = "best_of"
	// StrategyConsensus runs all agents and selects the result that most agents agree on.
	StrategyConsensus MultiAgentStrategy = "consensus"
)

// MultiAgentConfig configures multi-agent coordination behavior.
type MultiAgentConfig struct {
	Strategy         MultiAgentStrategy
	Providers        []string
	MaxParallel      int
	SelectionTimeout time.Duration
}

// AgentRun captures the input and output of a single agent execution.
type AgentRun struct {
	Provider     string
	GenerationID uuid.UUID
	SandboxID    string
	Result       *ExecutionResult
	Error        error
	StartedAt    time.Time
	CompletedAt  time.Time
	Score        float64
}

// MultiAgentCoordinator orchestrates parallel provider executions and selects the best result.
type MultiAgentCoordinator struct {
	registry *ProviderRegistry
	config   MultiAgentConfig
}

func NewMultiAgentCoordinator(registry *ProviderRegistry, config MultiAgentConfig) *MultiAgentCoordinator {
	if config.MaxParallel <= 0 {
		config.MaxParallel = len(config.Providers)
	}
	if config.MaxParallel <= 0 {
		config.MaxParallel = 3
	}
	if config.SelectionTimeout <= 0 {
		config.SelectionTimeout = 30 * time.Minute
	}
	if config.Strategy == "" {
		config.Strategy = StrategyRace
	}
	return &MultiAgentCoordinator{
		registry: registry,
		config:   config,
	}
}

// RunAll executes the given function for each configured provider concurrently,
// limited by MaxParallel, and returns all completed runs.
func (c *MultiAgentCoordinator) RunAll(ctx context.Context, fn func(ctx context.Context, provider AgentProvider) (*ExecutionResult, error)) []AgentRun {
	ctx, cancel := context.WithTimeout(ctx, c.config.SelectionTimeout)
	defer cancel()

	sem := make(chan struct{}, c.config.MaxParallel)
	var mu sync.Mutex
	runs := make([]AgentRun, 0, len(c.config.Providers))

	var wg sync.WaitGroup
	for _, providerName := range c.config.Providers {
		provider, ok := c.registry.Get(providerName)
		if !ok {
			slog.Warn("multi-agent: provider not found, skipping", "provider", providerName)
			continue
		}

		wg.Add(1)
		go func(p AgentProvider, name string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			run := AgentRun{
				Provider:  name,
				StartedAt: time.Now().UTC(),
			}

			result, err := fn(ctx, p)
			run.CompletedAt = time.Now().UTC()
			run.Result = result
			run.Error = err
			if result != nil {
				run.Score = scoreResult(result)
			}

			mu.Lock()
			runs = append(runs, run)
			mu.Unlock()

			if c.config.Strategy == StrategyRace && err == nil && result != nil && result.Success {
				cancel()
			}
		}(provider, providerName)
	}

	wg.Wait()
	return runs
}

// SelectBest picks the best run based on the configured strategy.
func (c *MultiAgentCoordinator) SelectBest(runs []AgentRun) (*AgentRun, error) {
	successful := make([]AgentRun, 0, len(runs))
	for _, r := range runs {
		if r.Error == nil && r.Result != nil && r.Result.Success {
			successful = append(successful, r)
		}
	}

	if len(successful) == 0 {
		return nil, fmt.Errorf("multi-agent: no successful runs from %d attempts", len(runs))
	}

	switch c.config.Strategy {
	case StrategyRace:
		sort.Slice(successful, func(i, j int) bool {
			return successful[i].CompletedAt.Before(successful[j].CompletedAt)
		})
		return &successful[0], nil

	case StrategyBestOf:
		sort.Slice(successful, func(i, j int) bool {
			return successful[i].Score > successful[j].Score
		})
		return &successful[0], nil

	case StrategyConsensus:
		sort.Slice(successful, func(i, j int) bool {
			return successful[i].Score > successful[j].Score
		})
		return &successful[0], nil

	default:
		return &successful[0], nil
	}
}

// scoreResult produces a quality score for a completed execution result.
func scoreResult(r *ExecutionResult) float64 {
	if r == nil || !r.Success {
		return 0
	}

	score := 50.0

	if len(r.FileChanges) > 0 {
		score += 20.0
	}

	if r.ExitCode == 0 {
		score += 15.0
	}

	if r.Duration > 0 && r.Duration < 5*time.Minute {
		score += 15.0
	} else if r.Duration < 10*time.Minute {
		score += 10.0
	}

	return score
}
