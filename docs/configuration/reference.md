# Configuration Reference

Full annotated configuration. All sections are optional — only configure what you need.

```yaml
# Proxy listen address
listen: ":8787"

# ─── Providers ───────────────────────────────────────────────────────

providers:
  openai:
    upstream: "https://api.openai.com"
    enabled: true
  anthropic:
    upstream: "https://api.anthropic.com"
    enabled: true

  # Additional providers — route via path prefix
  # e.g., /groq/v1/chat/completions → api.groq.com
  extra:
    groq:
      type: "openai"                    # OpenAI-compatible API format
      upstream: "https://api.groq.com/openai"
      path_prefix: "/groq"
      enabled: true
    mistral:
      type: "openai"
      upstream: "https://api.mistral.ai"
      path_prefix: "/mistral"
      enabled: true
    deepseek:
      type: "openai"
      upstream: "https://api.deepseek.com"
      path_prefix: "/deepseek"
      enabled: true
    gemini:
      type: "gemini"                    # Custom Gemini parser
      upstream: "https://generativelanguage.googleapis.com"
      path_prefix: "/gemini"
      enabled: true
    cohere:
      type: "cohere"                    # Custom Cohere parser
      upstream: "https://api.cohere.com"
      path_prefix: "/cohere"
      enabled: true
    azure:
      type: "azure"                     # Azure OpenAI
      upstream: "https://my-resource.openai.azure.com"
      path_prefix: "/azure"
      enabled: true
    together:
      type: "openai"
      upstream: "https://api.together.xyz"
      path_prefix: "/together"
      enabled: true
    fireworks:
      type: "openai"
      upstream: "https://api.fireworks.ai/inference"
      path_prefix: "/fireworks"
      enabled: true
    perplexity:
      type: "openai"
      upstream: "https://api.perplexity.ai"
      path_prefix: "/perplexity"
      enabled: true
    openrouter:
      type: "openai"
      upstream: "https://openrouter.ai/api"
      path_prefix: "/openrouter"
      enabled: true
    xai:
      type: "openai"                    # xAI (Grok)
      upstream: "https://api.x.ai"
      path_prefix: "/xai"
      enabled: true
    cerebras:
      type: "openai"
      upstream: "https://api.cerebras.ai"
      path_prefix: "/cerebras"
      enabled: true
    sambanova:
      type: "openai"
      upstream: "https://api.sambanova.ai"
      path_prefix: "/sambanova"
      enabled: true

# ─── Storage ─────────────────────────────────────────────────────────

storage:
  driver: "sqlite"                      # "sqlite" or "postgres"
  dsn: "data/agentledger.db"            # SQLite path or Postgres DSN
  # max_open_conns: 25                  # Postgres only
  # max_idle_conns: 5                   # Postgres only
  # Example Postgres DSN:
  # dsn: "postgres://user:pass@localhost:5432/agentledger?sslmode=disable"

# ─── Logging ─────────────────────────────────────────────────────────

log:
  level: "info"                         # debug, info, warn, error
  format: "text"                        # text or json

# ─── Async Recording ────────────────────────────────────────────────

recording:
  buffer_size: 10000                    # channel buffer for async writes
  workers: 4                            # recording goroutines

# ─── Budget Enforcement ─────────────────────────────────────────────

budgets:
  default:
    daily_limit_usd: 50.0
    monthly_limit_usd: 500.0
    soft_limit_pct: 0.8                 # warn at 80% of limit
    action: "block"                     # "block" returns 429, "warn" adds header only
  rules:
    - api_key_pattern: "sk-proj-dev-*"  # glob pattern
      daily_limit_usd: 5.0
      monthly_limit_usd: 50.0
      action: "block"
    - tenant_id: "alpha"                # tenant-scoped rule
      daily_limit_usd: 100.0
      monthly_limit_usd: 1000.0
      action: "block"

# ─── Circuit Breaker ────────────────────────────────────────────────

circuit_breaker:
  max_failures: 5                       # consecutive 5xx before opening
  timeout_secs: 30                      # seconds before half-open retry

# ─── Agent Session Tracking ─────────────────────────────────────────

agent:
  session_timeout_mins: 30              # auto-expire idle sessions
  loop_threshold: 20                    # same path N times = loop (0 = disabled)
  loop_window_mins: 5                   # sliding window
  loop_action: "warn"                   # "warn" or "block"
  ghost_max_age_mins: 60                # sessions older than this = ghost (0 = disabled)
  ghost_min_calls: 50
  ghost_min_cost_usd: 1.0

# ─── Dashboard ───────────────────────────────────────────────────────

dashboard:
  enabled: true

# ─── Multi-Tenancy ──────────────────────────────────────────────────

tenants:
  enabled: true
  key_mappings:
    - api_key_pattern: "sk-proj-team-alpha-*"
      tenant_id: "alpha"
    - api_key_pattern: "sk-proj-team-beta-*"
      tenant_id: "beta"

# ─── Alerting ────────────────────────────────────────────────────────

alerts:
  slack:
    webhook_url: "https://hooks.slack.com/services/T00/B00/xxx"
  webhooks:
    - url: "https://api.example.com/alerts"
      headers:
        Authorization: "Bearer token"
  cooldown_mins: 5                      # deduplication window per alert

# ─── Rate Limiting ──────────────────────────────────────────────────

rate_limits:
  default:
    requests_per_minute: 60
    requests_per_hour: 1000
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      requests_per_minute: 10

# ─── Admin API ───────────────────────────────────────────────────────

admin:
  enabled: true
  token: "your-secret-admin-token"      # Bearer token for auth

# ─── MCP Tool Metering ──────────────────────────────────────────────

mcp:
  enabled: true
  upstream: "http://localhost:3000"
  pricing:
    - server: "filesystem"
      tool: "read_file"
      cost_per_call: 0.01
    - server: "filesystem"
      tool: ""                          # wildcard: all tools on server
      cost_per_call: 0.005
    - server: "github"
      tool: ""
      cost_per_call: 0.02
```
