# AgentLedger

**Know what your agents cost.** Meter. Budget. Control.

AgentLedger is a reverse proxy that sits between your AI agents and LLM providers, tracking every token, calculating costs, and enforcing budgets — all without changing a single line of your application code.

```bash
export OPENAI_BASE_URL=http://localhost:8787/v1
# That's it. Your agents now have cost tracking and budget enforcement.
```

**[Documentation](https://wdz-dev.github.io/agent-ledger/)** | **[GitHub](https://github.com/WDZ-Dev/agent-ledger)**

## Why AgentLedger?

AI agents make dozens of LLM calls per task. Costs compound fast, loops happen silently, and provider dashboards only show you the damage after the fact.

AgentLedger gives you:

- **Real-time cost tracking** — every request metered, every token counted
- **Budget enforcement** — daily and monthly limits with automatic blocking
- **Pre-flight estimation** — rejects requests that would exceed your budget before they hit the API
- **Agent session tracking** — group multi-call agent runs into sessions, detect loops and ghost agents
- **MCP tool metering** — track costs of MCP tool calls alongside LLM usage
- **Dashboard** — embedded web UI for real-time cost visibility
- **Observability** — OpenTelemetry metrics with Prometheus endpoint
- **Circuit breaker** — automatic upstream failure protection
- **Multi-provider** — 15 providers: OpenAI, Anthropic, Azure OpenAI, Gemini, Groq, Mistral, DeepSeek, Cohere, xAI, Perplexity, Together AI, Fireworks AI, OpenRouter, Cerebras, SambaNova
- **Multi-tenancy** — isolate costs by team/org with tenant-scoped budgets
- **Alerting** — Slack and webhook notifications for budget warnings and anomalies
- **Rate limiting** — per-key request throttling with sliding window counters
- **Admin API** — runtime budget rule management without restarts
- **Zero code changes** — works with any OpenAI/Anthropic SDK via base URL override

## Quick Start

### Install

**Homebrew:**

```bash
brew install wdz-dev/tap/agentledger
```

**Binary download** — grab the latest release from [GitHub Releases](https://github.com/WDZ-Dev/agent-ledger/releases).

**From source:**

```bash
go install github.com/WDZ-Dev/agent-ledger/cmd/agentledger@latest
```

**Docker:**

```bash
docker run --rm -p 8787:8787 ghcr.io/wdz-dev/agent-ledger:latest
```

**Helm (Kubernetes):**

```bash
helm install agentledger deploy/helm/agentledger
```

### Run

```bash
# Start the proxy with defaults (listens on :8787)
agentledger serve

# Or with a config file
agentledger serve -c configs/agentledger.example.yaml
```

### Point your agents at it

```bash
# Python (OpenAI SDK)
export OPENAI_BASE_URL=http://localhost:8787/v1

# Node.js
const openai = new OpenAI({ baseURL: 'http://localhost:8787/v1' });

# Claude Code
export ANTHROPIC_BASE_URL=http://localhost:8787

# Other providers — route via path prefix
# curl http://localhost:8787/groq/v1/chat/completions
# curl http://localhost:8787/mistral/v1/chat/completions
# curl http://localhost:8787/deepseek/v1/chat/completions
# curl http://localhost:8787/gemini/v1beta/models/gemini-2.5-pro:generateContent
# curl http://localhost:8787/cohere/v2/chat
# curl http://localhost:8787/xai/v1/chat/completions
# curl http://localhost:8787/together/v1/chat/completions
# curl http://localhost:8787/fireworks/v1/chat/completions
# curl http://localhost:8787/perplexity/v1/chat/completions
# curl http://localhost:8787/openrouter/v1/chat/completions
# curl http://localhost:8787/cerebras/v1/chat/completions
# curl http://localhost:8787/sambanova/v1/chat/completions
```

### OpenClaw

Track what your OpenClaw agents cost. Add AgentLedger as a custom provider in `~/.openclaw/openclaw.json`:

```json
{
  "models": {
    "providers": {
      "agentledger-openai": {
        "baseUrl": "http://localhost:8787/v1",
        "apiKey": "${OPENAI_API_KEY}",
        "api": "openai-completions",
        "models": [
          { "id": "gpt-4o", "name": "GPT-4o via AgentLedger", "contextWindow": 128000, "maxTokens": 16384 }
        ]
      },
      "agentledger-anthropic": {
        "baseUrl": "http://localhost:8787",
        "apiKey": "${ANTHROPIC_API_KEY}",
        "api": "anthropic-messages",
        "models": [
          { "id": "claude-sonnet-4-20250514", "name": "Claude Sonnet via AgentLedger", "contextWindow": 200000, "maxTokens": 8192 }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": ["agentledger-openai/gpt-4o", "agentledger-anthropic/claude-sonnet-4-20250514"]
    }
  }
}
```

Apply the config and all OpenClaw LLM calls flow through AgentLedger:

```bash
openclaw gateway config.apply --file ~/.openclaw/openclaw.json
```

Open `http://localhost:8787` to see every call, set budget limits, and detect runaway agents.

### Check your costs

```bash
# Last 24 hours, grouped by model
agentledger costs

# Last 7 days, grouped by API key
agentledger costs --last 7d --by key
```

```
PROVIDER   MODEL            REQUESTS   INPUT TOKENS   OUTPUT TOKENS   COST (USD)
--------   -----            --------   ------------   -------------   ----------
openai     gpt-4.1-mini     142        28400          14200           $0.0341
openai     gpt-4.1          38         19000          9500            $0.1140
anthropic  claude-sonnet-4   12         6000           3000            $0.0630
--------   -----            --------   ------------   -------------   ----------
TOTAL                       192        53400          26700           $0.2111
```

### Docker Compose

```bash
cd deploy && docker compose up
```

## Architecture

```
┌─────────────┐       ┌──────────────────────┐       ┌──────────────┐
│   Agents    │──────▶│    AgentLedger :8787  │──────▶│  OpenAI      │
│  (any SDK)  │       │                      │       │  Anthropic   │
└─────────────┘       │  ┌────────────────┐  │       │  Azure OpenAI│
                      │  │ Rate Limiting  │  │       │  Gemini      │
┌─────────────┐       │  │ Budget Check   │  │       │  Groq        │
│ MCP Servers │◀─────▶│  │ Token Metering │  │       │  Mistral     │
│(stdio/HTTP) │       │  │ Agent Sessions │  │       │  DeepSeek    │
└─────────────┘       │  │ Cost Calc      │  │       │  + 8 more    │
                      │  │ Async Record   │  │       └──────────────┘
                      │  └────────────────┘  │       ┌──────────────┐
                      │          │           │──────▶│  Slack       │
                      │  ┌───────▼────────┐  │       │  Webhooks    │
                      │  │ SQLite/Postgres │  │       └──────────────┘
                      │  └────────────────┘  │
                      │          │           │
                      │  ┌───────▼────────┐  │
                      │  │ Dashboard :8787 │  │
                      │  │ Admin API      │  │
                      │  │ Prometheus      │  │
                      │  └────────────────┘  │
                      └──────────────────────┘
```

**Request flow:**

1. Agent sends request to AgentLedger
2. Budget check — reject immediately if over limit
3. Pre-flight estimation — reject if `max_tokens` cost exceeds remaining budget
4. Forward to upstream provider (API key passes through untouched)
5. Parse response for token usage
6. Calculate cost from model pricing table
7. Record asynchronously (never blocks the response)
8. Return response to agent with optional budget warning headers

## Features

### Cost Tracking

Every request is metered with provider-reported token counts. When streaming responses don't include usage data, AgentLedger falls back to tiktoken estimation (flagged as `estimated: true`).

**Supported providers and models:**

| Provider | Routing | Models |
|----------|---------|--------|
| OpenAI | `/v1/` (default) | gpt-5 family, gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, gpt-4o, gpt-4o-mini, o3, o3-pro, o3-mini, o4-mini, o1, o1-pro, o1-mini, gpt-4-turbo, gpt-4, gpt-3.5-turbo |
| Anthropic | `/v1/messages` | claude-opus-4.6, claude-sonnet-4.6, claude-opus-4.5, claude-sonnet-4.5, claude-haiku-4.5, claude-opus-4, claude-sonnet-4, claude-3.7-sonnet, claude-3.5-sonnet, claude-3.5-haiku, claude-3-opus, claude-3-haiku |
| Azure OpenAI | `/azure/` | Same as OpenAI (custom deployment names) |
| Gemini | `/gemini/` | gemini-2.5-pro, gemini-2.5-flash, gemini-2.0-flash, gemini-1.5-pro, gemini-1.5-flash |
| Groq | `/groq/v1/` | llama-3.3-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b-32768, gemma2-9b-it |
| Mistral | `/mistral/v1/` | mistral-large-latest, mistral-small-latest, codestral-latest, open-mistral-nemo |
| DeepSeek | `/deepseek/v1/` | deepseek-chat, deepseek-reasoner |
| Cohere | `/cohere/` | command-r-plus, command-r, command-light |
| xAI | `/xai/v1/` | grok-3, grok-3-mini, grok-2 |
| Perplexity | `/perplexity/v1/` | sonar-pro, sonar, sonar-reasoning |
| Together AI | `/together/v1/` | Llama 3.3 70B, Llama 3.1 405B/8B, Qwen 2.5 72B, DeepSeek V3 |
| Fireworks AI | `/fireworks/v1/` | Llama 3.3 70B, Llama 3.1 8B, Qwen 2.5 72B |
| OpenRouter | `/openrouter/v1/` | Any model via OpenRouter routing |
| Cerebras | `/cerebras/v1/` | llama-3.3-70b, llama-3.1-8b |
| SambaNova | `/sambanova/v1/` | Llama 3.3 70B, Llama 3.1 8B |

**83+ models** with built-in pricing. Groq, Mistral, DeepSeek, xAI, Perplexity, Together, Fireworks, OpenRouter, Cerebras, and SambaNova use the OpenAI-compatible API format. Gemini and Cohere have custom parsers. Versioned model names (e.g., `gpt-4o-2024-11-20`) are matched via longest prefix.

### Budget Enforcement

Set daily and monthly spend limits. When exceeded, requests are rejected with a `429` status and a clear JSON error:

```json
{
  "error": {
    "type": "budget_exceeded",
    "message": "spending limit exceeded",
    "daily_spent": 12.50,
    "daily_limit": 10.00,
    "monthly_spent": 45.00,
    "monthly_limit": 500.00
  }
}
```

**Soft limits** add an `X-AgentLedger-Budget-Warning` response header when you're approaching the threshold, without blocking.

**Pre-flight estimation** calculates worst-case cost from `max_tokens` before the request reaches the API. If it would exceed the remaining budget, it's rejected immediately — no wasted spend.

**Per-key rules** let you set different limits for different API keys using glob patterns:

```yaml
budgets:
  default:
    daily_limit_usd: 50.0
    monthly_limit_usd: 500.0
    soft_limit_pct: 0.8
    action: "block"
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      daily_limit_usd: 5.0
      action: "block"
```

### Agent Session Tracking

AgentLedger groups multi-call agent runs into sessions, enabling per-execution cost attribution. Tag requests with agent metadata using headers (stripped before forwarding to the provider):

```
X-Agent-Id: code-reviewer
X-Agent-Session: sess_abc123
X-Agent-User: user@example.com
X-Agent-Task: "Review PR #456"
```

**Loop detection** identifies runaway agents making repetitive calls:

```yaml
agent:
  loop_threshold: 20      # same path N times in window = loop
  loop_window_mins: 5
  loop_action: "warn"     # "warn" or "block"
```

**Ghost detection** finds forgotten agents still burning tokens:

```yaml
agent:
  ghost_max_age_mins: 60
  ghost_min_calls: 50
  ghost_min_cost_usd: 1.0
```

### MCP Tool Metering

Meter MCP (Model Context Protocol) tool calls alongside LLM costs. Two modes:

**HTTP proxy** — forward JSON-RPC to an upstream MCP server:

```yaml
mcp:
  enabled: true
  upstream: "http://localhost:3000"
  pricing:
    - server: "filesystem"
      tool: "read_file"
      cost_per_call: 0.01
```

**Stdio wrapper** — wrap any MCP server process:

```bash
agentledger mcp-wrap -- npx @modelcontextprotocol/server-filesystem /tmp
```

### Dashboard

Embedded web UI at the root URL (`http://localhost:8787/`) with real-time cost breakdowns, session views, and spending trends.

### Observability

OpenTelemetry metrics exported via Prometheus at `/metrics`:

- Request latency, token counts, cost totals
- Session lifecycle, loop/ghost alerts
- MCP tool call counts and costs

### Circuit Breaker

Protects against upstream failures. After a configurable number of consecutive 5xx responses, the circuit opens and rejects requests immediately. Auto-recovers after a timeout.

```yaml
circuit_breaker:
  max_failures: 5
  timeout_secs: 30
```

### Multi-Tenancy

Isolate costs, budgets, and dashboards by team or organization. Enable tenancy and map API keys to tenants:

```yaml
tenants:
  enabled: true
  key_mappings:
    - api_key_pattern: "sk-proj-team-alpha-*"
      tenant_id: "alpha"
    - api_key_pattern: "sk-proj-team-beta-*"
      tenant_id: "beta"
```

Or set the tenant per-request via header: `X-AgentLedger-Tenant: alpha`.

All dashboard and cost endpoints accept an optional `?tenant=` filter.

### Alerting

Get notified when budgets are approaching limits or agents are misbehaving:

```yaml
alerts:
  slack:
    webhook_url: "https://hooks.slack.com/services/..."
  webhooks:
    - url: "https://api.example.com/alerts"
      headers:
        Authorization: "Bearer token"
  cooldown_mins: 5   # deduplication window per alert
```

Alert types: `budget_warning`, `budget_exceeded`, `loop_detected`, `ghost_detected`.

### Rate Limiting

Throttle request volume per API key with sliding window counters:

```yaml
rate_limits:
  default:
    requests_per_minute: 60
    requests_per_hour: 1000
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      requests_per_minute: 10
```

Returns `429 Too Many Requests` with a `Retry-After` header when exceeded.

### Admin API

Manage budget rules at runtime without restarting. Protected by Bearer token auth:

```yaml
admin:
  enabled: true
  token: "your-secret-admin-token"
```

Endpoints:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/budgets/rules` | List budget rules |
| POST | `/api/admin/budgets/rules` | Create a budget rule |
| DELETE | `/api/admin/budgets/rules?pattern=...` | Delete a rule by pattern |
| GET | `/api/admin/api-keys` | List API key hashes with monthly spend |
| GET | `/api/admin/providers` | List provider status |

Runtime rules take effect immediately and persist across restarts.

### API Key Security

Raw API keys are never stored. AgentLedger creates a SHA-256 fingerprint from the first 8 and last 4 characters of the key. The full key passes through to the upstream provider untouched.

## Configuration

AgentLedger looks for config in these locations (in order):

1. Path passed via `--config` / `-c` flag
2. `./agentledger.yaml`
3. `./configs/agentledger.yaml`
4. `~/.config/agentledger/agentledger.yaml`
5. `/etc/agentledger/agentledger.yaml`

All settings can be overridden with environment variables prefixed `AGENTLEDGER_`:

```bash
AGENTLEDGER_LISTEN=":9090"
AGENTLEDGER_STORAGE_DSN="/tmp/ledger.db"
AGENTLEDGER_LOG_LEVEL="debug"
```

See [`configs/agentledger.example.yaml`](configs/agentledger.example.yaml) for the full reference.

## CLI

```
agentledger serve       Start the proxy
  -c, --config          Path to config file

agentledger costs       Show cost report
  -c, --config          Path to config file
  --last                Time window: 1h, 24h, 7d, 30d (default: 24h)
  --by                  Group by: model, provider, key (default: model)

agentledger export      Export cost data as CSV or JSON
  -c, --config          Path to config file
  --last                Time window (default: 30d)
  --by                  Group by: model, provider, key, agent, session
  -f, --format          Output format: csv or json (default: csv)
  --tenant              Filter by tenant ID

agentledger mcp-wrap    Wrap an MCP server process for tool call metering
  -c, --config          Path to config file
  -- command [args...]  MCP server command to wrap

agentledger version     Print version
```

## Performance

AgentLedger adds minimal overhead. Cost recording is fully async — it never blocks responses.

| Benchmark | Latency | Allocations |
|-----------|---------|-------------|
| Non-streaming proxy | ~115 us | moderate |
| Streaming proxy (SSE) | ~110 us | moderate |
| Health check | ~2 us | minimal |
| Cost calculation | ~192 ns | 0 allocs |
| Token estimation (tiktoken) | ~16 us | cached |

Target: <1ms proxy overhead per request. Actual: ~0.1ms.

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/) v2
- [lefthook](https://github.com/evilmartians/lefthook) (git hooks)

### Setup

```bash
make setup    # Install dev tools and git hooks
```

### Build & Test

```bash
make build         # Build binary to bin/agentledger
make test          # Run all tests with race detection
make lint          # Run golangci-lint
make dev           # Build and run with example config
make check         # Run all checks (fmt, vet, lint, test, vulncheck)
make docker        # Build Docker image
make docker-run    # Build and run in Docker
make helm-lint     # Lint Helm chart
make release-dry   # GoReleaser snapshot
make docs          # Build documentation site
make docs-serve    # Serve docs locally with live reload
```

## Project Structure

```
agent-ledger/
├── cmd/agentledger/           CLI entrypoint
│   ├── main.go                Root command + healthcheck (cobra)
│   ├── serve.go               Proxy server command
│   ├── costs.go               Cost report command
│   ├── export.go              CSV/JSON export command
│   └── mcpwrap.go             MCP stdio wrapper command
├── internal/
│   ├── proxy/                 Reverse proxy core
│   │   ├── proxy.go           httputil.ReverseProxy + budget integration
│   │   └── streaming.go       SSE stream interception
│   ├── provider/              LLM provider parsers
│   │   ├── provider.go        Provider interface + API key handling
│   │   ├── openai_compat.go   OpenAI-compatible base (shared by Groq/Mistral/DeepSeek)
│   │   ├── openai.go          OpenAI provider constructor
│   │   ├── anthropic.go       Anthropic messages API
│   │   ├── gemini.go          Google Gemini custom parser
│   │   ├── cohere.go          Cohere custom parser
│   │   ├── groq.go            Groq (OpenAI-compatible)
│   │   ├── mistral.go         Mistral (OpenAI-compatible)
│   │   ├── deepseek.go        DeepSeek (OpenAI-compatible)
│   │   ├── azure.go           Azure OpenAI
│   │   ├── xai.go             xAI/Grok (OpenAI-compatible)
│   │   ├── perplexity.go      Perplexity (OpenAI-compatible)
│   │   ├── together.go        Together AI (OpenAI-compatible)
│   │   ├── fireworks.go       Fireworks AI (OpenAI-compatible)
│   │   ├── openrouter.go      OpenRouter (OpenAI-compatible)
│   │   ├── cerebras.go        Cerebras (OpenAI-compatible)
│   │   ├── sambanova.go       SambaNova (OpenAI-compatible)
│   │   └── registry.go        Auto-detect provider from request + path prefix routing
│   ├── meter/                 Cost calculation
│   │   ├── meter.go           Token-to-USD conversion
│   │   ├── pricing.go         Model pricing table (83+ models)
│   │   └── estimator.go       Tiktoken fallback estimation
│   ├── ledger/                Storage layer
│   │   ├── ledger.go          Ledger interface
│   │   ├── models.go          UsageRecord, CostFilter, CostEntry
│   │   ├── sqlite.go          SQLite impl (CGO-free)
│   │   ├── postgres.go        PostgreSQL impl
│   │   ├── recorder.go        Async buffered recording
│   │   └── migrations/        Embedded SQL migrations (goose)
│   │       ├── sqlite/        SQLite-specific migrations
│   │       └── postgres/      PostgreSQL-specific migrations
│   ├── budget/                Budget enforcement
│   │   ├── budget.go          Per-key spend limits + caching
│   │   └── circuit_breaker.go Transport wrapper for upstream failures
│   ├── agent/                 Agent session tracking
│   │   ├── session.go         Session lifecycle management
│   │   └── detector.go        Loop/ghost detection
│   ├── mcp/                   MCP tool metering
│   │   ├── httpproxy.go       HTTP proxy for MCP servers
│   │   ├── interceptor.go     JSON-RPC interception
│   │   ├── stdio.go           Stdio wrapper for MCP processes
│   │   └── pricing.go         Per-call cost rules
│   ├── otel/                  Observability
│   │   ├── metrics.go         OTel metrics recording
│   │   └── provider.go        Prometheus exporter setup
│   ├── dashboard/             Web UI
│   │   ├── handlers.go        REST API handlers
│   │   ├── server.go          HTTP server + embedded assets
│   │   └── static/            Embedded JS/CSS/HTML assets
│   ├── tenant/                Multi-tenancy
│   │   └── tenant.go          Tenant resolver (header/config/chain)
│   ├── alert/                 Alerting
│   │   ├── alert.go           Alert types + multi-notifier
│   │   ├── slack.go           Slack webhook notifier
│   │   ├── webhook.go         Generic webhook notifier
│   │   └── ratelimit.go       Deduplication wrapper
│   ├── ratelimit/             Request rate limiting
│   │   └── limiter.go         Sliding window counter
│   ├── admin/                 Admin API
│   │   ├── handlers.go        REST API (budget CRUD, key listing)
│   │   └── store.go           Runtime config persistence
│   └── config/                YAML/env config (viper)
├── deploy/
│   ├── docker-compose.yml     One-command local dev
│   ├── Dockerfile.goreleaser  Slim image for releases
│   └── helm/agentledger/      Kubernetes Helm chart
├── configs/
│   └── agentledger.example.yaml
├── docs/                        MkDocs Material documentation site
│   ├── getting-started/         Installation, quickstart, CLI reference
│   ├── configuration/           Config overview and full reference
│   ├── features/                Per-feature documentation
│   ├── deployment/              Docker and Kubernetes guides
│   └── stylesheets/             Custom CSS overrides
├── .github/workflows/
│   ├── ci.yml                 Lint, test, build, vulncheck
│   ├── release.yml            GoReleaser on tag push
│   └── docs.yml               Build and deploy docs to GitHub Pages
├── Dockerfile                 Multi-stage Docker build
├── .goreleaser.yml            Cross-platform release config
├── mkdocs.yml                   Documentation site config
├── Makefile
├── go.mod
└── lefthook.yml               Pre-commit and pre-push hooks
```

## Roadmap

- [x] **Phase 1: Core Proxy** — Reverse proxy, token metering, cost calculation, SQLite storage, CLI
- [x] **Phase 2: Budget Enforcement** — Per-key budgets, pre-flight estimation, circuit breaker
- [x] **Phase 3: Agent Attribution** — Session tracking, loop detection, ghost agent detection
- [x] **Phase 4: Observability** — OpenTelemetry metrics, Prometheus endpoint, web dashboard
- [x] **Phase 5: MCP Integration** — Meter MCP tool calls alongside LLM costs
- [x] **Phase 6: Polish & Launch** — Docker, GoReleaser, Helm chart, docs
- [x] **Phase 7: Multi-Provider** — 15 providers with path-prefix routing (Groq, Mistral, DeepSeek, Gemini, Cohere, xAI, Perplexity, Together, Fireworks, OpenRouter, Cerebras, SambaNova, Azure)
- [x] **Phase 8: Postgres** — Production-grade PostgreSQL storage backend
- [x] **Phase 9: Multi-Tenancy** — Tenant isolation with header and config-based resolution
- [x] **Phase 10: Alerting** — Slack and webhook notifications with deduplication
- [x] **Phase 11: Rate Limiting** — Per-key request throttling + Homebrew tap
- [x] **Phase 12: Admin API** — Runtime budget rule management

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Run checks (`make check`)
4. Commit your changes
5. Open a pull request

## License

Apache 2.0
