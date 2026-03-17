# AgentLedger

> **"Know what your agents cost."** — Meter. Budget. Control.

A Go-based open-source reverse proxy that provides real-time cost attribution, budget enforcement, and financial observability for AI agents.

## Quick Context

- **What:** Transparent reverse proxy between AI agents and LLM APIs (OpenAI, Anthropic, Azure OpenAI, Groq, Mistral, DeepSeek, Gemini, Cohere, Together AI, Fireworks AI, Perplexity, OpenRouter, xAI, Cerebras, SambaNova)
- **How:** `export OPENAI_BASE_URL=http://localhost:8787/v1` — zero code changes
- **Why:** No tool tracks per-agent-execution costs, detects loops, or meters MCP calls
- **Language:** Go — single binary, zero runtime dependencies
- **License:** Apache 2.0
- **Repo:** git@github.com:WDZ-Dev/agent-ledger.git

## Key Differentiators vs LiteLLM (Primary Competitor)

1. **Agent-native cost model** — per-agent-execution, not per-key/user/team
2. **Loop detection by default** — zero-config runaway agent detection at the proxy layer
3. **Go single binary** — no Python, no pip, no database server required
4. **All safety features free** — LiteLLM paywalls budget tracking, audit logs, SSO
5. **Sub-10ms proxy overhead** — vs LiteLLM's documented memory leaks and perf regressions

## Architecture

```
Agents → AgentLedger (Go proxy :8787) → LLM APIs (OpenAI, Anthropic, Groq, Mistral, DeepSeek, Gemini, Cohere)
              ↓                              ↓
         SQLite/Postgres ledger         Slack/Webhook alerts
              ↓
         Dashboard + Admin API + Prometheus + CLI
```

### Core Packages

| Package | Purpose |
|---------|---------|
| `cmd/agentledger/` | CLI entrypoint (cobra): `serve`, `costs`, `version` |
| `internal/proxy/` | Core reverse proxy (`httputil.ReverseProxy`), SSE streaming, middleware chain |
| `internal/provider/` | Provider interface + OpenAI/Anthropic/Azure/Gemini/Cohere parsers, OpenAI-compatible base type (Groq, Mistral, DeepSeek, Together, Fireworks, Perplexity, OpenRouter, xAI, Cerebras, SambaNova), path-prefix routing |
| `internal/meter/` | Cost calculation engine, model pricing table, tiktoken-go fallback |
| `internal/ledger/` | Storage interface, SQLite (modernc.org/sqlite, CGO-free) + Postgres impls, multi-tenant queries |
| `internal/budget/` | Budget enforcement middleware, circuit breaker |
| `internal/agent/` | Agent session tracking, loop/ghost detection — THE key differentiator |
| `internal/mcp/` | MCP call interception (stdio wrapper + SSE proxy) |
| `internal/config/` | YAML config loading (viper) |
| `internal/otel/` | OpenTelemetry metrics + Prometheus `/metrics` endpoint |
| `internal/tenant/` | Multi-tenancy: header/config-based tenant resolution |
| `internal/alert/` | Alerting: Slack + webhook notifiers with rate-limited deduplication |
| `internal/ratelimit/` | Request rate limiting: sliding window counters per API key |
| `internal/admin/` | Admin REST API: runtime budget rule CRUD, API key listing, config persistence |
| `internal/dashboard/` | Embedded web dashboard (React/Preact via go:embed) |

### Key Design Principles

- **Never block responses** — async cost recording via buffered channel (10K) + worker goroutines
- **Minimal parsing** — only extract `usage` fields, use `encoding/json` into small structs
- **API keys pass through untouched** — only store `sha256(key[:8] + key[-4:])` fingerprints
- **SQLite for dev, Postgres for prod** — same `Ledger` interface
- **Single binary** — embed web assets + SQL migrations via `go:embed`

## Agent Identification (Headers)

```
X-Agent-Id: code-reviewer          # which agent
X-Agent-Session: sess_abc123       # which execution run
X-Agent-User: user@example.com     # who triggered it
X-Agent-Task: "Review PR #456"     # human-readable description
```

Headers are stripped before forwarding to upstream. Without headers, falls back to API-key-level tracking.

## Implementation Phases

1. **Core Proxy** — proxy + token counting + cost ledger + CLI
2. **Budget Enforcement** — per-key budgets, circuit breaker, streaming-aware
3. **Agent Attribution** — session tracking, loop detection, ghost detection
4. **Observability** — OpenTelemetry, Prometheus, embedded web dashboard
5. **MCP Integration** — stdio wrapper + SSE proxy for MCP tool cost metering
6. **Polish & Launch** — Docker, GoReleaser, Helm, GitHub Action, docs
7. **Multi-Provider** — Groq, Mistral, DeepSeek, Gemini, Cohere with OpenAI-compatible base + path-prefix routing
8. **Postgres** — PostgreSQL storage backend with connection pooling
9. **Multi-Tenancy** — tenant isolation via header/config-based resolution, tenant-scoped budgets
10. **Alerting** — Slack + webhook notifications with rate-limited deduplication
11. **Rate Limiting** — sliding window per-key request throttling + Homebrew tap
12. **Admin API** — runtime budget rule CRUD, API key spend listing, config persistence

## Dependencies

```
github.com/spf13/cobra
github.com/spf13/viper
github.com/pkoukk/tiktoken-go
github.com/pressly/goose/v3
github.com/oklog/ulid/v2
github.com/sony/gobreaker
modernc.org/sqlite
github.com/lib/pq
go.opentelemetry.io/otel
```

## Build & Run

```bash
make build              # → bin/agentledger
make test               # → go test -race -cover ./...
make lint               # → golangci-lint run
make dev                # → go run ./cmd/agentledger serve
make docker             # → docker build
```

## Reference Docs

Detailed research and planning docs are in `references/` (gitignored):
- `references/IMPLEMENTATION_PLAN.md` — full implementation plan with all phases
- `references/COMPETITIVE_RESEARCH.md` — deep LiteLLM analysis, positioning patterns
- `references/MARKET_RESEARCH.md` — AI cost horror stories, stats, framework downloads
