package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Listen         string           `mapstructure:"listen"`
	Providers      ProvidersConfig  `mapstructure:"providers"`
	Storage        StorageConfig    `mapstructure:"storage"`
	Log            LogConfig        `mapstructure:"log"`
	Recording      RecordingConfig  `mapstructure:"recording"`
	Budgets        BudgetsConfig    `mapstructure:"budgets"`
	CircuitBreaker CBConfig         `mapstructure:"circuit_breaker"`
	Agent          AgentConfig      `mapstructure:"agent"`
	Dashboard      DashboardConfig  `mapstructure:"dashboard"`
	Tenants        TenantsConfig    `mapstructure:"tenants"`
	Alerts         AlertsConfig     `mapstructure:"alerts"`
	RateLimits     RateLimitsConfig `mapstructure:"rate_limits"`
	Admin          AdminConfig      `mapstructure:"admin"`
	MCP            MCPConfig        `mapstructure:"mcp"`
}

// ProvidersConfig holds per-provider settings.
type ProvidersConfig struct {
	OpenAI    ProviderConfig            `mapstructure:"openai"`
	Anthropic ProviderConfig            `mapstructure:"anthropic"`
	Extra     map[string]ProviderConfig `mapstructure:"extra"`
}

// ProviderConfig holds settings for a single upstream provider.
type ProviderConfig struct {
	Upstream   string `mapstructure:"upstream"`
	Enabled    bool   `mapstructure:"enabled"`
	Type       string `mapstructure:"type"`        // provider type: "openai", "anthropic", "gemini", "cohere"
	PathPrefix string `mapstructure:"path_prefix"` // URL path prefix for routing (e.g., "/groq")
}

// StorageConfig holds database settings.
type StorageConfig struct {
	Driver       string `mapstructure:"driver"`
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// RecordingConfig holds async recording pipeline settings.
type RecordingConfig struct {
	BufferSize int `mapstructure:"buffer_size"`
	Workers    int `mapstructure:"workers"`
}

// BudgetsConfig holds budget enforcement settings.
type BudgetsConfig struct {
	Default BudgetRuleConfig   `mapstructure:"default"`
	Rules   []BudgetRuleConfig `mapstructure:"rules"`
}

// BudgetRuleConfig defines budget limits for a set of API keys.
type BudgetRuleConfig struct {
	APIKeyPattern   string  `mapstructure:"api_key_pattern"`
	DailyLimitUSD   float64 `mapstructure:"daily_limit_usd"`
	MonthlyLimitUSD float64 `mapstructure:"monthly_limit_usd"`
	SoftLimitPct    float64 `mapstructure:"soft_limit_pct"`
	Action          string  `mapstructure:"action"`
}

// CBConfig holds circuit breaker settings.
type CBConfig struct {
	MaxFailures int   `mapstructure:"max_failures"`
	TimeoutSecs int64 `mapstructure:"timeout_secs"`
}

// AgentConfig holds agent session tracking settings.
type AgentConfig struct {
	SessionTimeoutMins int     `mapstructure:"session_timeout_mins"`
	LoopThreshold      int     `mapstructure:"loop_threshold"`
	LoopWindowMins     int     `mapstructure:"loop_window_mins"`
	LoopAction         string  `mapstructure:"loop_action"`
	GhostMaxAgeMins    int     `mapstructure:"ghost_max_age_mins"`
	GhostMinCalls      int     `mapstructure:"ghost_min_calls"`
	GhostMinCostUSD    float64 `mapstructure:"ghost_min_cost_usd"`
}

// DashboardConfig holds dashboard UI settings.
type DashboardConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// TenantsConfig holds multi-tenancy settings.
type TenantsConfig struct {
	Enabled     bool               `mapstructure:"enabled"`
	KeyMappings []TenantKeyMapping `mapstructure:"key_mappings"`
}

// TenantKeyMapping maps an API key glob pattern to a tenant ID.
type TenantKeyMapping struct {
	APIKeyPattern string `mapstructure:"api_key_pattern"`
	TenantID      string `mapstructure:"tenant_id"`
}

// AlertsConfig holds alerting/notification settings.
type AlertsConfig struct {
	Slack       AlertSlackConfig     `mapstructure:"slack"`
	Webhooks    []AlertWebhookConfig `mapstructure:"webhooks"`
	CooldownMin int                  `mapstructure:"cooldown_mins"`
}

// AlertSlackConfig holds Slack webhook settings.
type AlertSlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
}

// AlertWebhookConfig holds generic webhook settings.
type AlertWebhookConfig struct {
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
}

// RateLimitsConfig holds request rate limiting settings.
type RateLimitsConfig struct {
	Default RateLimitRule   `mapstructure:"default"`
	Rules   []RateLimitRule `mapstructure:"rules"`
}

// RateLimitRule defines rate limits for a set of API keys.
type RateLimitRule struct {
	APIKeyPattern     string `mapstructure:"api_key_pattern"`
	RequestsPerMinute int    `mapstructure:"requests_per_minute"`
	RequestsPerHour   int    `mapstructure:"requests_per_hour"`
}

// AdminConfig holds admin API settings.
type AdminConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"` // Bearer token for admin API auth
}

// MCPConfig holds MCP (Model Context Protocol) integration settings.
type MCPConfig struct {
	Enabled  bool             `mapstructure:"enabled"`
	Upstream string           `mapstructure:"upstream"`
	Pricing  []MCPPricingRule `mapstructure:"pricing"`
}

// MCPPricingRule defines per-call cost for an MCP server/tool combination.
type MCPPricingRule struct {
	Server      string  `mapstructure:"server"`
	Tool        string  `mapstructure:"tool"`
	CostPerCall float64 `mapstructure:"cost_per_call"`
}

// Load reads configuration from the given file path, environment variables,
// and defaults.
func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("listen", ":8787")
	v.SetDefault("providers.openai.upstream", "https://api.openai.com")
	v.SetDefault("providers.openai.enabled", true)
	v.SetDefault("providers.anthropic.upstream", "https://api.anthropic.com")
	v.SetDefault("providers.anthropic.enabled", true)
	// Extra providers — all disabled by default.
	v.SetDefault("providers.extra.groq.type", "openai")
	v.SetDefault("providers.extra.groq.upstream", "https://api.groq.com/openai")
	v.SetDefault("providers.extra.groq.path_prefix", "/groq")
	v.SetDefault("providers.extra.groq.enabled", false)
	v.SetDefault("providers.extra.mistral.type", "openai")
	v.SetDefault("providers.extra.mistral.upstream", "https://api.mistral.ai")
	v.SetDefault("providers.extra.mistral.path_prefix", "/mistral")
	v.SetDefault("providers.extra.mistral.enabled", false)
	v.SetDefault("providers.extra.deepseek.type", "openai")
	v.SetDefault("providers.extra.deepseek.upstream", "https://api.deepseek.com")
	v.SetDefault("providers.extra.deepseek.path_prefix", "/deepseek")
	v.SetDefault("providers.extra.deepseek.enabled", false)
	v.SetDefault("providers.extra.gemini.type", "gemini")
	v.SetDefault("providers.extra.gemini.upstream", "https://generativelanguage.googleapis.com")
	v.SetDefault("providers.extra.gemini.path_prefix", "/gemini")
	v.SetDefault("providers.extra.gemini.enabled", false)
	v.SetDefault("providers.extra.cohere.type", "cohere")
	v.SetDefault("providers.extra.cohere.upstream", "https://api.cohere.com")
	v.SetDefault("providers.extra.cohere.path_prefix", "/cohere")
	v.SetDefault("providers.extra.cohere.enabled", false)
	v.SetDefault("providers.extra.azure.type", "azure")
	v.SetDefault("providers.extra.azure.upstream", "")
	v.SetDefault("providers.extra.azure.path_prefix", "/azure")
	v.SetDefault("providers.extra.azure.enabled", false)
	v.SetDefault("providers.extra.together.type", "openai")
	v.SetDefault("providers.extra.together.upstream", "https://api.together.xyz")
	v.SetDefault("providers.extra.together.path_prefix", "/together")
	v.SetDefault("providers.extra.together.enabled", false)
	v.SetDefault("providers.extra.fireworks.type", "openai")
	v.SetDefault("providers.extra.fireworks.upstream", "https://api.fireworks.ai/inference")
	v.SetDefault("providers.extra.fireworks.path_prefix", "/fireworks")
	v.SetDefault("providers.extra.fireworks.enabled", false)
	v.SetDefault("providers.extra.perplexity.type", "openai")
	v.SetDefault("providers.extra.perplexity.upstream", "https://api.perplexity.ai")
	v.SetDefault("providers.extra.perplexity.path_prefix", "/perplexity")
	v.SetDefault("providers.extra.perplexity.enabled", false)
	v.SetDefault("providers.extra.openrouter.type", "openai")
	v.SetDefault("providers.extra.openrouter.upstream", "https://openrouter.ai/api")
	v.SetDefault("providers.extra.openrouter.path_prefix", "/openrouter")
	v.SetDefault("providers.extra.openrouter.enabled", false)
	v.SetDefault("providers.extra.xai.type", "openai")
	v.SetDefault("providers.extra.xai.upstream", "https://api.x.ai")
	v.SetDefault("providers.extra.xai.path_prefix", "/xai")
	v.SetDefault("providers.extra.xai.enabled", false)
	v.SetDefault("providers.extra.cerebras.type", "openai")
	v.SetDefault("providers.extra.cerebras.upstream", "https://api.cerebras.ai")
	v.SetDefault("providers.extra.cerebras.path_prefix", "/cerebras")
	v.SetDefault("providers.extra.cerebras.enabled", false)
	v.SetDefault("providers.extra.sambanova.type", "openai")
	v.SetDefault("providers.extra.sambanova.upstream", "https://api.sambanova.ai")
	v.SetDefault("providers.extra.sambanova.path_prefix", "/sambanova")
	v.SetDefault("providers.extra.sambanova.enabled", false)

	v.SetDefault("storage.driver", "sqlite")
	v.SetDefault("storage.dsn", "data/agentledger.db")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("recording.buffer_size", 10000)
	v.SetDefault("recording.workers", 4)
	v.SetDefault("agent.session_timeout_mins", 30)
	v.SetDefault("agent.loop_threshold", 0)
	v.SetDefault("agent.loop_window_mins", 5)
	v.SetDefault("agent.loop_action", "warn")
	v.SetDefault("agent.ghost_max_age_mins", 0)
	v.SetDefault("agent.ghost_min_calls", 50)
	v.SetDefault("agent.ghost_min_cost_usd", 1.0)
	v.SetDefault("dashboard.enabled", true)
	v.SetDefault("mcp.enabled", false)
	v.SetDefault("mcp.upstream", "")

	// Environment variables: AGENTLEDGER_LISTEN, AGENTLEDGER_STORAGE_DSN, etc.
	v.SetEnvPrefix("AGENTLEDGER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Config file
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("agentledger")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.config/agentledger")
		v.AddConfigPath("/etc/agentledger")
	}

	if err := v.ReadInConfig(); err != nil {
		// Missing config file is fine when using defaults + env vars,
		// but an explicit path must exist.
		if path != "" {
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}
