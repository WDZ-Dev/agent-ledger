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

	// QueryCostTimeseries returns cost and request counts bucketed by time interval.
	// interval should be "hour" or "day".
	QueryCostTimeseries(ctx context.Context, interval string, since, until time.Time) ([]TimeseriesPoint, error)

	// Close releases any held resources.
	Close() error
}
