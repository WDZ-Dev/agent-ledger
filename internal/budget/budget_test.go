package budget

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

// stubLedger returns configurable spend values.
type stubLedger struct {
	dailySpend   float64
	monthlySpend float64
}

func (s *stubLedger) RecordUsage(_ context.Context, _ *ledger.UsageRecord) error { return nil }
func (s *stubLedger) QueryCosts(_ context.Context, _ ledger.CostFilter) ([]ledger.CostEntry, error) {
	return nil, nil
}
func (s *stubLedger) GetTotalSpend(_ context.Context, _ string, since, _ time.Time) (float64, error) {
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if since.Equal(dayStart) || since.After(dayStart) {
		return s.dailySpend, nil
	}
	return s.monthlySpend, nil
}
func (s *stubLedger) GetTotalSpendByTenant(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (s *stubLedger) QueryCostTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]ledger.TimeseriesPoint, error) {
	return nil, nil
}
func (s *stubLedger) QueryRecentExpensive(_ context.Context, _, _ time.Time, _ string, _ int) ([]ledger.ExpensiveRequest, error) {
	return nil, nil
}
func (s *stubLedger) QueryErrorStats(_ context.Context, _, _ time.Time, _ string) (*ledger.ErrorStats, error) {
	return &ledger.ErrorStats{}, nil
}
func (s *stubLedger) Close() error { return nil }

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBudgetAllowWhenNoLimits(t *testing.T) {
	mgr := NewManager(&stubLedger{}, Config{}, newTestLogger())

	if mgr.Enabled() {
		t.Error("should not be enabled with no limits")
	}

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Allow {
		t.Errorf("expected Allow, got %d", result.Decision)
	}
}

func TestBudgetAllowUnderLimit(t *testing.T) {
	store := &stubLedger{dailySpend: 5.0, monthlySpend: 20.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD:   50.0,
			MonthlyLimitUSD: 500.0,
			SoftLimitPct:    0.8,
			Action:          "block",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	if !mgr.Enabled() {
		t.Fatal("should be enabled")
	}

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Allow {
		t.Errorf("expected Allow, got %d", result.Decision)
	}
}

func TestBudgetWarnAtSoftLimit(t *testing.T) {
	store := &stubLedger{dailySpend: 42.0, monthlySpend: 50.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD:   50.0,
			MonthlyLimitUSD: 500.0,
			SoftLimitPct:    0.8,
			Action:          "block",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Warn {
		t.Errorf("expected Warn at 84%% daily, got %d", result.Decision)
	}
}

func TestBudgetBlockAtHardLimit(t *testing.T) {
	store := &stubLedger{dailySpend: 55.0, monthlySpend: 100.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD:   50.0,
			MonthlyLimitUSD: 500.0,
			SoftLimitPct:    0.8,
			Action:          "block",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Block {
		t.Errorf("expected Block, got %d", result.Decision)
	}
	if result.DailySpent != 55.0 {
		t.Errorf("daily_spent = %f, want 55.0", result.DailySpent)
	}
	if result.DailyLimit != 50.0 {
		t.Errorf("daily_limit = %f, want 50.0", result.DailyLimit)
	}
}

func TestBudgetWarnActionAtHardLimit(t *testing.T) {
	store := &stubLedger{dailySpend: 55.0, monthlySpend: 100.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD: 50.0,
			Action:        "warn",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Warn {
		t.Errorf("expected Warn (action=warn), got %d", result.Decision)
	}
}

func TestBudgetMonthlyBlock(t *testing.T) {
	store := &stubLedger{dailySpend: 5.0, monthlySpend: 600.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD:   50.0,
			MonthlyLimitUSD: 500.0,
			Action:          "block",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	result := mgr.Check(context.Background(), "sk-test-key", "hash123", "")
	if result.Decision != Block {
		t.Errorf("expected Block on monthly limit, got %d", result.Decision)
	}
}

func TestBudgetRulePatternMatch(t *testing.T) {
	store := &stubLedger{dailySpend: 8.0, monthlySpend: 10.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD: 50.0,
			Action:        "block",
		},
		Rules: []Rule{
			{
				APIKeyPattern: "sk-proj-dev-*",
				DailyLimitUSD: 5.0,
				Action:        "block",
			},
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	// Dev key should be blocked at 8.0 > 5.0.
	result := mgr.Check(context.Background(), "sk-proj-dev-abc123", "hash-dev", "")
	if result.Decision != Block {
		t.Errorf("expected Block for dev key, got %d", result.Decision)
	}

	// Non-dev key should be allowed at 8.0 < 50.0.
	result = mgr.Check(context.Background(), "sk-proj-prod-xyz789", "hash-prod", "")
	if result.Decision != Allow {
		t.Errorf("expected Allow for prod key, got %d", result.Decision)
	}
}

func TestBudgetRuleMergesDefaults(t *testing.T) {
	store := &stubLedger{dailySpend: 3.0, monthlySpend: 600.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD:   50.0,
			MonthlyLimitUSD: 500.0,
			SoftLimitPct:    0.8,
			Action:          "block",
		},
		Rules: []Rule{
			{
				APIKeyPattern: "sk-proj-dev-*",
				DailyLimitUSD: 5.0,
				// MonthlyLimitUSD and Action inherit from default.
			},
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	// Should block because monthly 600 > default 500.
	result := mgr.Check(context.Background(), "sk-proj-dev-abc", "hash-dev", "")
	if result.Decision != Block {
		t.Errorf("expected Block from inherited monthly limit, got %d", result.Decision)
	}
}

func TestBudgetSpendCaching(t *testing.T) {
	store := &stubLedger{dailySpend: 1.0, monthlySpend: 5.0}
	cfg := Config{
		Default: Rule{
			DailyLimitUSD: 50.0,
			Action:        "block",
		},
	}
	mgr := NewManager(store, cfg, newTestLogger())

	// First call populates cache.
	mgr.Check(context.Background(), "sk-key", "hash1", "")

	// Change underlying spend — cached value should be returned.
	store.dailySpend = 100.0
	result := mgr.Check(context.Background(), "sk-key", "hash1", "")
	if result.Decision != Allow {
		t.Error("expected cached Allow, but got blocked from stale data")
	}
}

func TestBudgetEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"no limits", Config{}, false},
		{"daily limit", Config{Default: Rule{DailyLimitUSD: 10}}, true},
		{"monthly limit", Config{Default: Rule{MonthlyLimitUSD: 100}}, true},
		{"rule only", Config{Rules: []Rule{{APIKeyPattern: "*", DailyLimitUSD: 5}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(&stubLedger{}, tt.cfg, newTestLogger())
			if got := mgr.Enabled(); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
