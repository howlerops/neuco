package codegen

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProviderMetrics holds aggregated run analytics for a provider within an org.
type ProviderMetrics struct {
	Provider      string        `json:"provider"`
	OrgID         uuid.UUID     `json:"org_id"`
	TotalRuns     int           `json:"total_runs"`
	SuccessfulRuns int          `json:"successful_runs"`
	FailedRuns    int           `json:"failed_runs"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	TotalTokens   int64         `json:"total_tokens"`
	TotalCostUSD  float64       `json:"total_cost_usd"`
	AvgCostUSD    float64       `json:"avg_cost_usd"`
	SuccessRate   float64       `json:"success_rate"`
	LastRunAt     time.Time     `json:"last_run_at"`
}

// RunRecord is a single provider execution used for analytics aggregation.
type RunRecord struct {
	Provider     string        `json:"provider"`
	OrgID        uuid.UUID     `json:"org_id"`
	GenerationID uuid.UUID     `json:"generation_id"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	TokensUsed   int64         `json:"tokens_used"`
	CostUSD      float64       `json:"cost_usd"`
	ExitCode     int           `json:"exit_code"`
	FileChanges  int           `json:"file_changes"`
	Timestamp    time.Time     `json:"timestamp"`
}

// AnalyticsCollector buffers run records and computes in-memory analytics.
type AnalyticsCollector struct {
	mu   sync.Mutex
	runs []RunRecord
}

// NewAnalyticsCollector constructs a new analytics collector.
func NewAnalyticsCollector() *AnalyticsCollector {
	return &AnalyticsCollector{
		runs: make([]RunRecord, 0),
	}
}

// Record appends a run record into the in-memory analytics buffer.
func (c *AnalyticsCollector) Record(record RunRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.runs = append(c.runs, record)
}

// MetricsByProvider aggregates run metrics by provider for a given org.
func (c *AnalyticsCollector) MetricsByProvider(orgID uuid.UUID) map[string]*ProviderMetrics {
	runs := c.snapshotRuns()
	metrics := make(map[string]*ProviderMetrics)

	for _, run := range runs {
		if run.OrgID != orgID {
			continue
		}

		entry, ok := metrics[run.Provider]
		if !ok {
			entry = &ProviderMetrics{
				Provider: run.Provider,
				OrgID:    orgID,
			}
			metrics[run.Provider] = entry
		}

		entry.TotalRuns++
		if run.Success {
			entry.SuccessfulRuns++
		} else {
			entry.FailedRuns++
		}
		entry.TotalDuration += run.Duration
		entry.TotalTokens += run.TokensUsed
		entry.TotalCostUSD += run.CostUSD
		if run.Timestamp.After(entry.LastRunAt) {
			entry.LastRunAt = run.Timestamp
		}
	}

	for _, entry := range metrics {
		if entry.TotalRuns == 0 {
			continue
		}

		entry.AvgDuration = entry.TotalDuration / time.Duration(entry.TotalRuns)
		entry.AvgCostUSD = entry.TotalCostUSD / float64(entry.TotalRuns)
		entry.SuccessRate = float64(entry.SuccessfulRuns) / float64(entry.TotalRuns)
	}

	return metrics
}

// MetricsForProvider returns aggregated metrics for one provider in an org.
func (c *AnalyticsCollector) MetricsForProvider(orgID uuid.UUID, provider string) *ProviderMetrics {
	metricsByProvider := c.MetricsByProvider(orgID)
	metric, ok := metricsByProvider[provider]
	if !ok {
		return nil
	}

	return metric
}

// TopProviders returns providers sorted by success rate descending for an org.
func (c *AnalyticsCollector) TopProviders(orgID uuid.UUID, limit int) []ProviderMetrics {
	metricsByProvider := c.MetricsByProvider(orgID)
	providers := make([]ProviderMetrics, 0, len(metricsByProvider))
	for _, metric := range metricsByProvider {
		providers = append(providers, *metric)
	}

	sort.Slice(providers, func(i, j int) bool {
		if providers[i].SuccessRate != providers[j].SuccessRate {
			return providers[i].SuccessRate > providers[j].SuccessRate
		}
		if providers[i].SuccessfulRuns != providers[j].SuccessfulRuns {
			return providers[i].SuccessfulRuns > providers[j].SuccessfulRuns
		}
		if providers[i].TotalRuns != providers[j].TotalRuns {
			return providers[i].TotalRuns > providers[j].TotalRuns
		}
		return providers[i].Provider < providers[j].Provider
	})

	if limit <= 0 || limit >= len(providers) {
		return providers
	}

	return providers[:limit]
}

// RecentRuns returns the most recent runs for an org sorted by timestamp descending.
func (c *AnalyticsCollector) RecentRuns(orgID uuid.UUID, limit int) []RunRecord {
	runs := c.snapshotRuns()
	recent := make([]RunRecord, 0)

	for _, run := range runs {
		if run.OrgID == orgID {
			recent = append(recent, run)
		}
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Timestamp.After(recent[j].Timestamp)
	})

	if limit <= 0 || limit >= len(recent) {
		return recent
	}

	return recent[:limit]
}

func (c *AnalyticsCollector) snapshotRuns() []RunRecord {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.runs) == 0 {
		return nil
	}

	out := make([]RunRecord, len(c.runs))
	copy(out, c.runs)
	return out
}
