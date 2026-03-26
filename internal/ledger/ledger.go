package ledger

import (
	"context"
	"time"
)

// Ledger defines the storage interface for usage records.
// SQLite and PostgreSQL implementations share this interface.
type Ledger interface {
	// RecordUsage inserts a single usage record.
	RecordUsage(ctx context.Context, record *UsageRecord) error

	// QueryCosts returns aggregated cost data matching the filter.
	QueryCosts(ctx context.Context, filter CostFilter) ([]CostEntry, error)

	// GetTotalSpend returns total USD spent for a given API key hash
	// within the specified time window.
	GetTotalSpend(ctx context.Context, apiKeyHash string, since, until time.Time) (float64, error)

	// GetTotalSpendByTenant returns total USD spent for a tenant within the time window.
	GetTotalSpendByTenant(ctx context.Context, tenantID string, since, until time.Time) (float64, error)

	// QueryCostTimeseries returns cost and request counts bucketed by time interval.
	// interval should be "minute", "hour", or "day". tenantID is optional (empty = all tenants).
	QueryCostTimeseries(ctx context.Context, interval string, since, until time.Time, tenantID string) ([]TimeseriesPoint, error)

	// QueryRecentExpensive returns the N most expensive individual requests in the time window.
	QueryRecentExpensive(ctx context.Context, since, until time.Time, tenantID string, limit int) ([]ExpensiveRequest, error)

	// QueryErrorStats returns error counts and average metrics for the time window.
	QueryErrorStats(ctx context.Context, since, until time.Time, tenantID string) (*ErrorStats, error)

	// QueryRecentSessions returns sessions within the time window, optionally filtered by status.
	QueryRecentSessions(ctx context.Context, since, until time.Time, status string, limit int) ([]SessionRecord, error)

	// QueryLatencyPercentiles returns P50/P90/P99 latency and a histogram distribution.
	QueryLatencyPercentiles(ctx context.Context, since, until time.Time, tenantID string) (*LatencyStats, error)

	// QueryTokenTimeseries returns token counts bucketed by time interval.
	QueryTokenTimeseries(ctx context.Context, interval string, since, until time.Time, tenantID string) ([]TokenTimeseriesPoint, error)

	// Close releases any held resources.
	Close() error
}
