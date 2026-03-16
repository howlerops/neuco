package codegen

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	adaptiveMinSamples = 3
	adaptiveMaxDuration = 30 * time.Minute
	adaptiveMaxCostUSD = 1.0
)

// ProviderScore captures adaptive scoring output for a provider.
type ProviderScore struct {
	Provider   string  `json:"provider"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
	SampleSize int     `json:"sample_size"`
}

// AdaptiveSelector chooses the best provider using historical run analytics.
type AdaptiveSelector struct {
	analytics       *AnalyticsCollector
	registry        *ProviderRegistry
	defaultProvider string
}

// NewAdaptiveSelector constructs a new adaptive provider selector.
func NewAdaptiveSelector(registry *ProviderRegistry, analytics *AnalyticsCollector, defaultProvider string) *AdaptiveSelector {
	return &AdaptiveSelector{
		analytics:       analytics,
		registry:        registry,
		defaultProvider: strings.TrimSpace(defaultProvider),
	}
}

// SelectProvider returns the best provider for the org.
// Falls back to the default provider when metrics are missing or sample sizes are insufficient.
func (s *AdaptiveSelector) SelectProvider(orgID uuid.UUID) (AgentProvider, error) {
	fallback, err := s.defaultAgentProvider()
	if err != nil {
		return nil, err
	}

	scores := s.ScoreProviders(orgID)
	if len(scores) == 0 {
		return fallback, nil
	}

	for _, score := range scores {
		if score.SampleSize < adaptiveMinSamples {
			return fallback, nil
		}
	}

	selected, ok := s.registry.Get(scores[0].Provider)
	if !ok {
		return fallback, nil
	}

	return selected, nil
}

// ScoreProviders computes weighted scores for all registered providers.
func (s *AdaptiveSelector) ScoreProviders(orgID uuid.UUID) []ProviderScore {
	if s == nil || s.registry == nil {
		return nil
	}

	providers := s.registry.List()
	if len(providers) == 0 {
		return nil
	}

	metricsByProvider := map[string]*ProviderMetrics{}
	if s.analytics != nil {
		metricsByProvider = s.analytics.MetricsByProvider(orgID)
	}

	scores := make([]ProviderScore, 0, len(providers))
	for _, provider := range providers {
		metrics := metricsByProvider[provider]
		providerScore := ProviderScore{Provider: provider}
		if metrics != nil {
			providerScore.SampleSize = metrics.TotalRuns
			providerScore.Confidence = sampleConfidence(metrics.TotalRuns)
			providerScore.Score = weightedProviderScore(metrics)
		}
		scores = append(scores, providerScore)
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			if scores[i].Confidence == scores[j].Confidence {
				return scores[i].Provider < scores[j].Provider
			}
			return scores[i].Confidence > scores[j].Confidence
		}
		return scores[i].Score > scores[j].Score
	})

	return scores
}

func (s *AdaptiveSelector) defaultAgentProvider() (AgentProvider, error) {
	if s == nil || s.registry == nil {
		return nil, ErrProviderNotFound
	}

	name := strings.TrimSpace(s.defaultProvider)
	if name == "" {
		providers := s.registry.List()
		if len(providers) == 0 {
			return nil, ErrProviderNotFound
		}
		name = providers[0]
	}

	provider, ok := s.registry.Get(name)
	if !ok {
		return nil, ErrProviderNotFound
	}

	return provider, nil
}

func weightedProviderScore(metrics *ProviderMetrics) float64 {
	if metrics == nil || metrics.TotalRuns == 0 {
		return 0
	}

	successRate := clampAdaptive(metrics.SuccessRate)
	speedScore := clampAdaptive(1.0 - clampAdaptive(float64(metrics.AvgDuration)/float64(adaptiveMaxDuration)))
	costScore := clampAdaptive(1.0 - clampAdaptive(metrics.AvgCostUSD/adaptiveMaxCostUSD))

	return (successRate * 0.6) + (speedScore * 0.2) + (costScore * 0.2)
}

func sampleConfidence(sampleSize int) float64 {
	if sampleSize <= 0 {
		return 0
	}
	return clampAdaptive(float64(sampleSize) / 10.0)
}

func clampAdaptive(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
