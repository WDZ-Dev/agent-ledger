package ledger

import "time"

// UsageRecord represents a single metered LLM API call.
type UsageRecord struct {
	ID           string
	Timestamp    time.Time
	Provider     string
	Model        string
	APIKeyHash   string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
	Estimated    bool
	DurationMS   int64
	StatusCode   int
	Path         string
	AgentID      string
	SessionID    string
	UserID       string
}

// CostFilter specifies which records to include in a cost query.
type CostFilter struct {
	Since   time.Time
	Until   time.Time
	GroupBy string // "model", "provider", "key"
}

// CostEntry is an aggregated cost row returned by QueryCosts.
type CostEntry struct {
	Provider     string
	Model        string
	APIKeyHash   string
	Requests     int
	InputTokens  int64
	OutputTokens int64
	TotalCostUSD float64
}
