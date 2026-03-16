package codegen

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAnalyticsCollector_RecordAppends(t *testing.T) {
	t.Parallel()

	collector := NewAnalyticsCollector()
	orgID := uuid.New()
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	records := []RunRecord{
		{Provider: "p1", OrgID: orgID, Success: true, Duration: 2 * time.Second, Timestamp: base.Add(1 * time.Minute)},
		{Provider: "p2", OrgID: orgID, Success: false, Duration: 3 * time.Second, Timestamp: base.Add(2 * time.Minute)},
		{Provider: "p3", OrgID: orgID, Success: true, Duration: 4 * time.Second, Timestamp: base.Add(3 * time.Minute)},
	}

	for _, record := range records {
		collector.Record(record)
	}

	recent := collector.RecentRuns(orgID, 0)
	if len(recent) != len(records) {
		t.Fatalf("expected %d recent runs, got %d", len(records), len(recent))
	}
}

func TestAnalyticsCollector_MetricsByProviderAggregates(t *testing.T) {
	t.Parallel()

	collector := NewAnalyticsCollector()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	base := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	collector.Record(RunRecord{Provider: "alpha", OrgID: orgID, Success: true, Duration: 10 * time.Second, TokensUsed: 100, CostUSD: 0.2, Timestamp: base.Add(1 * time.Minute)})
	collector.Record(RunRecord{Provider: "alpha", OrgID: orgID, Success: false, Duration: 20 * time.Second, TokensUsed: 50, CostUSD: 0.4, Timestamp: base.Add(2 * time.Minute)})
	collector.Record(RunRecord{Provider: "beta", OrgID: orgID, Success: true, Duration: 30 * time.Second, TokensUsed: 200, CostUSD: 0.6, Timestamp: base.Add(3 * time.Minute)})
	collector.Record(RunRecord{Provider: "alpha", OrgID: otherOrgID, Success: true, Duration: 999 * time.Second, TokensUsed: 999, CostUSD: 9.9, Timestamp: base.Add(4 * time.Minute)})

	metrics := collector.MetricsByProvider(orgID)

	alpha := metrics["alpha"]
	if alpha == nil {
		t.Fatal("expected alpha metrics")
	}
	if alpha.TotalRuns != 2 || alpha.SuccessfulRuns != 1 || alpha.FailedRuns != 1 {
		t.Fatalf("alpha run counts incorrect: %+v", *alpha)
	}
	if alpha.TotalDuration != 30*time.Second {
		t.Fatalf("alpha total duration: want 30s, got %s", alpha.TotalDuration)
	}
	if alpha.AvgDuration != 15*time.Second {
		t.Fatalf("alpha avg duration: want 15s, got %s", alpha.AvgDuration)
	}
	if alpha.TotalTokens != 150 {
		t.Fatalf("alpha total tokens: want 150, got %d", alpha.TotalTokens)
	}
	if math.Abs(alpha.TotalCostUSD-0.6) > 1e-9 {
		t.Fatalf("alpha total cost: want 0.6, got %f", alpha.TotalCostUSD)
	}
	if math.Abs(alpha.AvgCostUSD-0.3) > 1e-9 {
		t.Fatalf("alpha avg cost: want 0.3, got %f", alpha.AvgCostUSD)
	}
	if math.Abs(alpha.SuccessRate-0.5) > 1e-9 {
		t.Fatalf("alpha success rate: want 0.5, got %f", alpha.SuccessRate)
	}
	if !alpha.LastRunAt.Equal(base.Add(2 * time.Minute)) {
		t.Fatalf("alpha last run timestamp mismatch: got %s", alpha.LastRunAt)
	}

	beta := metrics["beta"]
	if beta == nil {
		t.Fatal("expected beta metrics")
	}
	if beta.TotalRuns != 1 || beta.SuccessfulRuns != 1 || beta.FailedRuns != 0 {
		t.Fatalf("beta run counts incorrect: %+v", *beta)
	}
	if math.Abs(beta.SuccessRate-1.0) > 1e-9 {
		t.Fatalf("beta success rate: want 1.0, got %f", beta.SuccessRate)
	}
	if beta.TotalTokens != 200 {
		t.Fatalf("beta total tokens: want 200, got %d", beta.TotalTokens)
	}
	if !beta.LastRunAt.Equal(base.Add(3 * time.Minute)) {
		t.Fatalf("beta last run timestamp mismatch: got %s", beta.LastRunAt)
	}
}

func TestAnalyticsCollector_TopProvidersSortedBySuccessRate(t *testing.T) {
	t.Parallel()

	collector := NewAnalyticsCollector()
	orgID := uuid.New()
	now := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		collector.Record(RunRecord{Provider: "alpha", OrgID: orgID, Success: true, Duration: time.Second, Timestamp: now.Add(time.Duration(i) * time.Minute)})
	}
	for i := 0; i < 2; i++ {
		collector.Record(RunRecord{Provider: "beta", OrgID: orgID, Success: true, Duration: time.Second, Timestamp: now.Add(time.Duration(i+10) * time.Minute)})
	}
	collector.Record(RunRecord{Provider: "beta", OrgID: orgID, Success: false, Duration: time.Second, Timestamp: now.Add(12 * time.Minute)})
	collector.Record(RunRecord{Provider: "gamma", OrgID: orgID, Success: false, Duration: time.Second, Timestamp: now.Add(20 * time.Minute)})

	top := collector.TopProviders(orgID, 0)
	if len(top) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(top))
	}

	if top[0].Provider != "alpha" {
		t.Fatalf("expected alpha first (highest success rate), got %s", top[0].Provider)
	}
	if top[1].Provider != "beta" {
		t.Fatalf("expected beta second, got %s", top[1].Provider)
	}
	if top[2].Provider != "gamma" {
		t.Fatalf("expected gamma third, got %s", top[2].Provider)
	}

	limited := collector.TopProviders(orgID, 2)
	if len(limited) != 2 {
		t.Fatalf("expected limit=2 to return 2 providers, got %d", len(limited))
	}
}

func TestAnalyticsCollector_RecentRunsLimitAndSort(t *testing.T) {
	t.Parallel()

	collector := NewAnalyticsCollector()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	base := time.Date(2025, 4, 1, 10, 0, 0, 0, time.UTC)

	collector.Record(RunRecord{Provider: "a", OrgID: orgID, Timestamp: base.Add(1 * time.Minute)})
	collector.Record(RunRecord{Provider: "b", OrgID: orgID, Timestamp: base.Add(3 * time.Minute)})
	collector.Record(RunRecord{Provider: "c", OrgID: orgID, Timestamp: base.Add(2 * time.Minute)})
	collector.Record(RunRecord{Provider: "d", OrgID: otherOrgID, Timestamp: base.Add(10 * time.Minute)})

	recent := collector.RecentRuns(orgID, 2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent runs, got %d", len(recent))
	}

	if !recent[0].Timestamp.After(recent[1].Timestamp) && !recent[0].Timestamp.Equal(recent[1].Timestamp) {
		t.Fatalf("expected descending timestamp order, got %s then %s", recent[0].Timestamp, recent[1].Timestamp)
	}
	if recent[0].Provider != "b" || recent[1].Provider != "c" {
		t.Fatalf("unexpected order/providers: got %s then %s", recent[0].Provider, recent[1].Provider)
	}
}
