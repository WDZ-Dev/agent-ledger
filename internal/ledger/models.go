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
	TenantID     string
}

// CostFilter specifies which records to include in a cost query.
type CostFilter struct {
	Since    time.Time
	Until    time.Time
	GroupBy  string // "model", "provider", "key", "agent", "session"
	TenantID string // optional tenant filter (empty = all tenants)
}

// TimeseriesPoint represents a single data point in a cost timeseries.
type TimeseriesPoint struct {
	Timestamp time.Time
	CostUSD   float64
	Requests  int
}

// CostEntry is an aggregated cost row returned by QueryCosts.
type CostEntry struct {
	Provider     string
	Model        string
	APIKeyHash   string
	AgentID      string
	SessionID    string
	Requests     int
	InputTokens  int64
	OutputTokens int64
	TotalCostUSD float64
}

// ExpensiveRequest is a single high-cost request for the "top expensive" view.
type ExpensiveRequest struct {
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	AgentID      string    `json:"agent_id"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	DurationMS   int64     `json:"duration_ms"`
}

// ErrorStats contains error rate information for a time window.
type ErrorStats struct {
	TotalRequests int     `json:"total_requests"`
	ErrorRequests int     `json:"error_requests"`
	ErrorRate     float64 `json:"error_rate"`
	Count429      int     `json:"count_429"`
	Count5xx      int     `json:"count_5xx"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	AvgCostPerReq float64 `json:"avg_cost_per_request"`
}
