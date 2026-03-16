package codegen

import (
	"context"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

type testAgentProvider struct {
	name string
}

func (p *testAgentProvider) Name() string { return p.name }

func (p *testAgentProvider) DisplayName() string { return p.name }

func (p *testAgentProvider) ValidateConfig(ctx context.Context, cfg AgentConfig) error { return nil }

func (p *testAgentProvider) InstallInstructions() string { return "" }

func (p *testAgentProvider) DetectInstalled(pathEnv string) bool { return true }

func (p *testAgentProvider) BuildCommand(req ExecutionRequest) (*exec.Cmd, error) { return nil, nil }

func (p *testAgentProvider) ParseOutput(r io.Reader) <-chan ProgressEvent {
	ch := make(chan ProgressEvent)
	close(ch)
	return ch
}

func TestAdaptiveSelector_SelectProviderReturnsDefaultWhenNoAnalyticsData(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry(
		&testAgentProvider{name: "default-provider"},
		&testAgentProvider{name: "other-provider"},
	)
	selector := NewAdaptiveSelector(registry, NewAnalyticsCollector(), "default-provider")

	selected, err := selector.SelectProvider(uuid.New())
	if err != nil {
		t.Fatalf("SelectProvider returned error: %v", err)
	}
	if selected.Name() != "default-provider" {
		t.Fatalf("expected default-provider, got %s", selected.Name())
	}
}

func TestAdaptiveSelector_ScoreProvidersComputesWeightedScores(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	registry := NewProviderRegistry(
		&testAgentProvider{name: "alpha"},
		&testAgentProvider{name: "beta"},
	)
	analytics := NewAnalyticsCollector()
	now := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 4; i++ {
		analytics.Record(RunRecord{
			Provider:  "alpha",
			OrgID:     orgID,
			Success:   true,
			Duration:  1 * time.Minute,
			CostUSD:   0.1,
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	for i := 0; i < 4; i++ {
		analytics.Record(RunRecord{
			Provider:  "beta",
			OrgID:     orgID,
			Success:   i < 2,
			Duration:  25 * time.Minute,
			CostUSD:   0.9,
			Timestamp: now.Add(time.Duration(i+10) * time.Minute),
		})
	}

	selector := NewAdaptiveSelector(registry, analytics, "alpha")
	scores := selector.ScoreProviders(orgID)
	if len(scores) != 2 {
		t.Fatalf("expected 2 provider scores, got %d", len(scores))
	}

	if scores[0].Provider != "alpha" {
		t.Fatalf("expected alpha to have higher weighted score, got %s", scores[0].Provider)
	}
	if scores[1].Provider != "beta" {
		t.Fatalf("expected beta second, got %s", scores[1].Provider)
	}

	if scores[0].SampleSize != 4 || scores[1].SampleSize != 4 {
		t.Fatalf("expected sample size 4 for both providers, got %d and %d", scores[0].SampleSize, scores[1].SampleSize)
	}

	if scores[0].Confidence <= 0 || scores[1].Confidence <= 0 {
		t.Fatalf("expected non-zero confidence values, got %f and %f", scores[0].Confidence, scores[1].Confidence)
	}

	if scores[0].Score <= scores[1].Score {
		t.Fatalf("expected alpha score > beta score, got %f <= %f", scores[0].Score, scores[1].Score)
	}
}

func TestAdaptiveSelector_FallbackBehaviorOnInsufficientSamples(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	registry := NewProviderRegistry(
		&testAgentProvider{name: "alpha"},
		&testAgentProvider{name: "fallback"},
	)
	analytics := NewAnalyticsCollector()
	now := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 2; i++ {
		analytics.Record(RunRecord{
			Provider:  "alpha",
			OrgID:     orgID,
			Success:   true,
			Duration:  1 * time.Minute,
			CostUSD:   0.1,
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	selector := NewAdaptiveSelector(registry, analytics, "fallback")
	selected, err := selector.SelectProvider(orgID)
	if err != nil {
		t.Fatalf("SelectProvider returned error: %v", err)
	}
	if selected.Name() != "fallback" {
		t.Fatalf("expected fallback provider due to insufficient samples, got %s", selected.Name())
	}
}

func TestAdaptiveSelector_FallbackBehaviorWhenDefaultMissing(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry(&testAgentProvider{name: "only-provider"})
	selector := NewAdaptiveSelector(registry, NewAnalyticsCollector(), "missing-default")

	_, err := selector.SelectProvider(uuid.New())
	if err == nil {
		t.Fatal("expected error when default provider does not exist")
	}
	if !errorsIs(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}

func errorsIs(err error, target error) bool {
	if err == nil {
		return target == nil
	}
	if target == nil {
		return false
	}
	return err.Error() == target.Error()
}

var _ AgentProvider = (*testAgentProvider)(nil)
var _ = domain.Spec{}
