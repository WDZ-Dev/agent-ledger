package budget

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

// Decision represents the outcome of a budget check.
type Decision int

const (
	Allow Decision = iota
	Warn
	Block
)

// Result holds the details of a budget check.
type Result struct {
	Decision     Decision
	DailySpent   float64
	DailyLimit   float64
	MonthlySpent float64
	MonthlyLimit float64
}

// Rule defines budget limits for a set of API keys.
type Rule struct {
	APIKeyPattern   string  `mapstructure:"api_key_pattern" json:"api_key_pattern"`
	TenantID        string  `mapstructure:"tenant_id" json:"tenant_id,omitempty"`
	DailyLimitUSD   float64 `mapstructure:"daily_limit_usd" json:"daily_limit_usd"`
	MonthlyLimitUSD float64 `mapstructure:"monthly_limit_usd" json:"monthly_limit_usd"`
	SoftLimitPct    float64 `mapstructure:"soft_limit_pct" json:"soft_limit_pct"`
	Action          string  `mapstructure:"action" json:"action"`
}

// Config holds the default budget and per-key override rules.
type Config struct {
	Default Rule   `mapstructure:"default"`
	Rules   []Rule `mapstructure:"rules"`
}

// AlertNotifier is an optional interface for sending budget alerts.
type AlertNotifier interface {
	Notify(ctx context.Context, alert interface{ GetType() string }) error
}

// Manager enforces budget limits by checking spend against configured rules.
type Manager struct {
	ledger   ledger.Ledger
	config   Config
	cache    sync.Map // apiKeyHash -> *spendEntry
	cacheTTL time.Duration
	logger   *slog.Logger
	onWarn   func(ctx context.Context, apiKeyHash string, result Result)
	onBlock  func(ctx context.Context, apiKeyHash string, result Result)

	mu     sync.RWMutex // protects config.Rules
	done   chan struct{}
	closed sync.Once
}

type spendEntry struct {
	daily   float64
	monthly float64
	fetched time.Time
}

const defaultCacheTTL = 30 * time.Second

// NewManager creates a budget enforcement manager.
func NewManager(l ledger.Ledger, cfg Config, logger *slog.Logger) *Manager {
	m := &Manager{
		ledger:   l,
		config:   cfg,
		cacheTTL: defaultCacheTTL,
		logger:   logger,
		done:     make(chan struct{}),
	}
	go m.cacheCleanupLoop()
	return m
}

// Close stops the background cache cleanup goroutine.
func (m *Manager) Close() {
	m.closed.Do(func() {
		close(m.done)
	})
}

// SetCallbacks configures alert callbacks for budget events.
func (m *Manager) SetCallbacks(
	onWarn func(ctx context.Context, apiKeyHash string, result Result),
	onBlock func(ctx context.Context, apiKeyHash string, result Result),
) {
	m.onWarn = onWarn
	m.onBlock = onBlock
}

// UpdateRules replaces the per-key rules at runtime (hot-reload from admin API).
// The default rule is not changed.
func (m *Manager) UpdateRules(rules []Rule) {
	m.mu.Lock()
	m.config.Rules = rules
	m.mu.Unlock()
	// Invalidate cache so new rules take effect immediately.
	m.cache.Range(func(key, _ any) bool {
		m.cache.Delete(key)
		return true
	})
}

// Enabled returns true if any budget limits are configured.
func (m *Manager) Enabled() bool {
	if m.config.Default.DailyLimitUSD > 0 || m.config.Default.MonthlyLimitUSD > 0 {
		return true
	}
	m.mu.RLock()
	n := len(m.config.Rules)
	m.mu.RUnlock()
	return n > 0
}

// Check evaluates budget for a request. rawKey is used for rule pattern
// matching; apiKeyHash is used for spend lookups. tenantID is optional;
// when non-empty a tenant-scoped budget rule is also checked and the
// stricter result wins.
func (m *Manager) Check(ctx context.Context, rawKey, apiKeyHash, tenantID string) Result {
	rule := m.matchRule(rawKey)
	if rule.DailyLimitUSD <= 0 && rule.MonthlyLimitUSD <= 0 {
		// Even with no key-level limits, tenant limits may apply.
		if tenantID == "" {
			return Result{Decision: Allow}
		}
		tenantRule := m.matchTenantRule(tenantID)
		if tenantRule == nil {
			return Result{Decision: Allow}
		}
		tenantDaily, tenantMonthly := m.getTenantSpend(ctx, tenantID)
		result := m.evaluateRule(*tenantRule, tenantDaily, tenantMonthly)
		if result.Decision == Block && m.onBlock != nil {
			m.onBlock(ctx, apiKeyHash, result)
		} else if result.Decision == Warn && m.onWarn != nil {
			m.onWarn(ctx, apiKeyHash, result)
		}
		return result
	}

	daily, monthly := m.getSpend(ctx, apiKeyHash)
	result := m.evaluateRule(rule, daily, monthly)

	if result.Decision == Block && m.onBlock != nil {
		m.onBlock(ctx, apiKeyHash, result)
	} else if result.Decision == Warn && m.onWarn != nil {
		m.onWarn(ctx, apiKeyHash, result)
	}

	// Tenant-level budget check (if applicable).
	if tenantID != "" {
		tenantRule := m.matchTenantRule(tenantID)
		if tenantRule != nil {
			tenantDaily, tenantMonthly := m.getTenantSpend(ctx, tenantID)
			tenantResult := m.evaluateRule(*tenantRule, tenantDaily, tenantMonthly)
			// Take the stricter decision.
			if tenantResult.Decision > result.Decision {
				result = tenantResult
				if result.Decision == Block && m.onBlock != nil {
					m.onBlock(ctx, apiKeyHash, result)
				} else if result.Decision == Warn && m.onWarn != nil {
					m.onWarn(ctx, apiKeyHash, result)
				}
			}
		}
	}

	return result
}

// evaluateRule checks daily/monthly spend against a rule and returns the
// appropriate Result. It does not fire callbacks.
func (m *Manager) evaluateRule(rule Rule, daily, monthly float64) Result {
	result := Result{
		Decision:     Allow,
		DailySpent:   daily,
		DailyLimit:   rule.DailyLimitUSD,
		MonthlySpent: monthly,
		MonthlyLimit: rule.MonthlyLimitUSD,
	}

	// Check hard limits.
	exceeded := (rule.DailyLimitUSD > 0 && daily >= rule.DailyLimitUSD) ||
		(rule.MonthlyLimitUSD > 0 && monthly >= rule.MonthlyLimitUSD)

	if exceeded {
		if rule.Action == "block" {
			result.Decision = Block
		} else {
			result.Decision = Warn
		}
		return result
	}

	// Check soft limits.
	if rule.SoftLimitPct > 0 {
		if rule.DailyLimitUSD > 0 && daily >= rule.DailyLimitUSD*rule.SoftLimitPct {
			result.Decision = Warn
		}
		if rule.MonthlyLimitUSD > 0 && monthly >= rule.MonthlyLimitUSD*rule.SoftLimitPct {
			result.Decision = Warn
		}
	}

	return result
}

// matchRule returns the most specific rule matching the raw API key,
// falling back to the default rule.
func (m *Manager) matchRule(rawKey string) Rule {
	m.mu.RLock()
	rules := make([]Rule, len(m.config.Rules))
	copy(rules, m.config.Rules)
	m.mu.RUnlock()

	for _, r := range rules {
		if matched, _ := filepath.Match(r.APIKeyPattern, rawKey); matched {
			return m.mergeWithDefault(r)
		}
	}
	return m.config.Default
}

// mergeWithDefault fills zero fields from the default rule.
func (m *Manager) mergeWithDefault(r Rule) Rule {
	if r.DailyLimitUSD <= 0 {
		r.DailyLimitUSD = m.config.Default.DailyLimitUSD
	}
	if r.MonthlyLimitUSD <= 0 {
		r.MonthlyLimitUSD = m.config.Default.MonthlyLimitUSD
	}
	if r.SoftLimitPct <= 0 {
		r.SoftLimitPct = m.config.Default.SoftLimitPct
	}
	if r.Action == "" {
		r.Action = m.config.Default.Action
	}
	return r
}

// matchTenantRule finds a rule that targets a specific tenant (no API key pattern).
func (m *Manager) matchTenantRule(tenantID string) *Rule {
	m.mu.RLock()
	rules := make([]Rule, len(m.config.Rules))
	copy(rules, m.config.Rules)
	m.mu.RUnlock()

	for _, r := range rules {
		if r.TenantID == tenantID && r.APIKeyPattern == "" {
			merged := m.mergeWithDefault(r)
			return &merged
		}
	}
	return nil
}

// getTenantSpend returns daily and monthly spend for a tenant, using a short-lived cache.
func (m *Manager) getTenantSpend(ctx context.Context, tenantID string) (daily, monthly float64) {
	cacheKey := "tenant:" + tenantID
	if entry, ok := m.cache.Load(cacheKey); ok {
		e := entry.(*spendEntry)
		if time.Since(e.fetched) < m.cacheTTL {
			return e.daily, e.monthly
		}
	}

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var err error
	daily, err = m.ledger.GetTotalSpendByTenant(ctx, tenantID, dayStart, now)
	if err != nil {
		m.logger.Error("budget: querying tenant daily spend", "error", err, "tenant_id", tenantID)
	}

	monthly, err = m.ledger.GetTotalSpendByTenant(ctx, tenantID, monthStart, now)
	if err != nil {
		m.logger.Error("budget: querying tenant monthly spend", "error", err, "tenant_id", tenantID)
	}

	m.cache.Store(cacheKey, &spendEntry{
		daily:   daily,
		monthly: monthly,
		fetched: time.Now(),
	})

	return daily, monthly
}

// getSpend returns daily and monthly spend, using a short-lived cache.
func (m *Manager) getSpend(ctx context.Context, apiKeyHash string) (daily, monthly float64) {
	if entry, ok := m.cache.Load(apiKeyHash); ok {
		e := entry.(*spendEntry)
		if time.Since(e.fetched) < m.cacheTTL {
			return e.daily, e.monthly
		}
	}

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var err error
	daily, err = m.ledger.GetTotalSpend(ctx, apiKeyHash, dayStart, now)
	if err != nil {
		m.logger.Error("budget: querying daily spend", "error", err)
	}

	monthly, err = m.ledger.GetTotalSpend(ctx, apiKeyHash, monthStart, now)
	if err != nil {
		m.logger.Error("budget: querying monthly spend", "error", err)
	}

	m.cache.Store(apiKeyHash, &spendEntry{
		daily:   daily,
		monthly: monthly,
		fetched: time.Now(),
	})

	return daily, monthly
}

func (m *Manager) cacheCleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.evictStaleCache()
		case <-m.done:
			return
		}
	}
}

func (m *Manager) evictStaleCache() {
	now := time.Now()
	m.cache.Range(func(key, value any) bool {
		e := value.(*spendEntry)
		if now.Sub(e.fetched) > m.cacheTTL {
			m.cache.Delete(key)
		}
		return true
	})
}
