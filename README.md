# AgentLedger

**Know what your agents cost.** Meter. Budget. Control.

AgentLedger is a reverse proxy that sits between your AI agents and LLM providers, tracking every token, calculating costs, and enforcing budgets — all without changing a single line of your application code.

```bash
export OPENAI_BASE_URL=http://localhost:8787/v1
# That's it. Your agents now have cost tracking and budget enforcement.
```

## Why AgentLedger?

AI agents make dozens of LLM calls per task. Costs compound fast, loops happen silently, and provider dashboards only show you the damage after the fact.

AgentLedger gives you:

- **Real-time cost tracking** — every request metered, every token counted
- **Budget enforcement** — daily and monthly limits with automatic blocking
- **Pre-flight estimation** — rejects requests that would exceed your budget before they hit the API
- **Agent-level attribution** — group multi-call agent runs into sessions (coming soon)
- **Circuit breaker** — automatic upstream failure protection
- **Zero code changes** — works with any OpenAI/Anthropic SDK via base URL override

## Quick Start

### Install from source

```bash
git clone https://github.com/WDZ-Dev/agent-ledger.git
cd agent-ledger
make build
```

### Run

```bash
# Start the proxy with defaults (listens on :8787)
./bin/agentledger serve

# Or with a config file
./bin/agentledger serve -c configs/agentledger.example.yaml
```

### Point your agents at it

```bash
# Python (OpenAI SDK)
export OPENAI_BASE_URL=http://localhost:8787/v1

# Node.js
const openai = new OpenAI({ baseURL: 'http://localhost:8787/v1' });

# Claude Code
export ANTHROPIC_BASE_URL=http://localhost:8787
```

### Check your costs

```bash
# Last 24 hours, grouped by model
./bin/agentledger costs

# Last 7 days, grouped by API key
./bin/agentledger costs --last 7d --by key
```

```
PROVIDER   MODEL            REQUESTS   INPUT TOKENS   OUTPUT TOKENS   COST (USD)
--------   -----            --------   ------------   -------------   ----------
openai     gpt-4o-mini      142        28400          14200           $0.0128
openai     gpt-4o           38         19000          9500            $0.1425
anthropic  claude-sonnet-4   12         6000           3000            $0.0630
--------   -----            --------   ------------   -------------   ----------
TOTAL                       192        53400          26700           $0.2183
```

## Architecture

```
┌─────────────┐       ┌──────────────────────┐       ┌──────────────┐
│   Agents    │──────▶│    AgentLedger :8787  │──────▶│  OpenAI API  │
│  (any SDK)  │       │                      │       │ Anthropic API│
└─────────────┘       │  ┌────────────────┐  │       └──────────────┘
                      │  │ Budget Check   │  │
                      │  │ Pre-flight Est │  │
                      │  │ Token Metering │  │
                      │  │ Cost Calc      │  │
                      │  │ Async Record   │  │
                      │  └────────────────┘  │
                      │          │           │
                      │  ┌───────▼────────┐  │
                      │  │ SQLite/Postgres │  │
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

**Supported models:**

| Provider | Models |
|----------|--------|
| OpenAI | gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-4, gpt-3.5-turbo, o1, o1-mini, o3, o3-mini, o4-mini |
| Anthropic | claude-opus-4, claude-sonnet-4, claude-3.5-sonnet, claude-3.5-haiku, claude-3-opus, claude-3-sonnet, claude-3-haiku |

Versioned model names (e.g., `gpt-4o-2024-11-20`) are matched via longest prefix.

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

### Circuit Breaker

Protects against upstream failures. After a configurable number of consecutive 5xx responses, the circuit opens and rejects requests immediately with a clear error. Auto-recovers after a timeout.

```yaml
circuit_breaker:
  max_failures: 5
  timeout_secs: 30
```

### Agent Identification

Tag requests with agent metadata using headers (stripped before forwarding to the provider):

```
X-Agent-Id: code-reviewer
X-Agent-Session: sess_abc123
X-Agent-User: user@example.com
X-Agent-Task: "Review PR #456"
```

These are stored in the usage ledger for per-agent cost attribution.

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

### Full Reference

```yaml
# Listen address
listen: ":8787"

# Upstream providers
providers:
  openai:
    upstream: "https://api.openai.com"
    enabled: true
  anthropic:
    upstream: "https://api.anthropic.com"
    enabled: true

# Storage backend
storage:
  driver: "sqlite"             # sqlite (postgres coming soon)
  dsn: "data/agentledger.db"

# Logging
log:
  level: "info"                # debug, info, warn, error
  format: "text"               # text, json

# Async recording pipeline
recording:
  buffer_size: 10000           # channel buffer size
  workers: 4                   # recording goroutines

# Budget enforcement (optional)
budgets:
  default:
    daily_limit_usd: 50.0
    monthly_limit_usd: 500.0
    soft_limit_pct: 0.8        # warn at 80% of limit
    action: "block"            # "block" = 429, "warn" = header only
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      daily_limit_usd: 5.0
      monthly_limit_usd: 50.0
      action: "block"

# Circuit breaker (optional)
circuit_breaker:
  max_failures: 5              # consecutive 5xx to trip
  timeout_secs: 30             # recovery timeout
```

## CLI

```
agentledger serve   Start the proxy
  -c, --config      Path to config file

agentledger costs   Show cost report
  -c, --config      Path to config file
  --last            Time window: 1h, 24h, 7d, 30d (default: 24h)
  --by              Group by: model, provider, key (default: model)

agentledger version Print version
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

- Go 1.24+
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
```

### Run locally

```bash
make dev
# or
go run ./cmd/agentledger serve -c configs/agentledger.example.yaml
```

## Project Structure

```
agent-ledger/
├── cmd/agentledger/           CLI entrypoint
│   ├── main.go                Root command (cobra)
│   ├── serve.go               Proxy server command
│   └── costs.go               Cost report command
├── internal/
│   ├── proxy/                 Reverse proxy core
│   │   ├── proxy.go           httputil.ReverseProxy + budget integration
│   │   └── streaming.go       SSE stream interception
│   ├── provider/              LLM provider parsers
│   │   ├── provider.go        Provider interface + API key handling
│   │   ├── openai.go          OpenAI chat/completions/embeddings
│   │   ├── anthropic.go       Anthropic messages API
│   │   └── registry.go        Auto-detect provider from request
│   ├── meter/                 Cost calculation
│   │   ├── meter.go           Token-to-USD conversion
│   │   ├── pricing.go         Model pricing table (17 models)
│   │   └── estimator.go       Tiktoken fallback estimation
│   ├── ledger/                Storage layer
│   │   ├── ledger.go          Ledger interface
│   │   ├── models.go          UsageRecord, CostFilter, CostEntry
│   │   ├── sqlite.go          SQLite impl (CGO-free)
│   │   ├── recorder.go        Async buffered recording
│   │   └── migrations/        Embedded SQL migrations (goose)
│   ├── budget/                Budget enforcement
│   │   ├── budget.go          Per-key spend limits + caching
│   │   └── circuit_breaker.go Transport wrapper for upstream failures
│   └── config/                YAML/env config (viper)
├── configs/
│   └── agentledger.example.yaml
├── .github/workflows/ci.yml   Lint, test, build, vulncheck
├── Makefile
├── go.mod
└── lefthook.yml               Pre-commit and pre-push hooks
```

## Roadmap

- [x] **Phase 1: Core Proxy** — Reverse proxy, token metering, cost calculation, SQLite storage, CLI
- [x] **Phase 2: Budget Enforcement** — Per-key budgets, pre-flight estimation, circuit breaker
- [ ] **Phase 3: Agent Attribution** — Session tracking, loop detection, ghost agent detection
- [ ] **Phase 4: Observability** — OpenTelemetry metrics, Prometheus endpoint, web dashboard
- [ ] **Phase 5: MCP Integration** — Meter MCP tool calls alongside LLM costs
- [ ] **Phase 6: Polish & Launch** — Docker, GoReleaser, Helm chart, docs

## License

Apache 2.0
