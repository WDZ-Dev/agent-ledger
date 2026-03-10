# AgentLedger

> **"Know what your agents cost."** — Meter. Budget. Control.

A Go-based open-source reverse proxy that provides real-time cost attribution, budget enforcement, and financial observability for AI agents.

## Quick Context

- **What:** Transparent reverse proxy between AI agents and LLM APIs (OpenAI, Anthropic)
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
Agents → AgentLedger (Go proxy :8787) → LLM APIs (OpenAI, Anthropic)
              ↓
         SQLite/Postgres ledger
              ↓
         Dashboard + Prometheus + CLI
```

### Core Packages

| Package | Purpose |
|---------|---------|
| `cmd/agentledger/` | CLI entrypoint (cobra): `serve`, `costs`, `version` |
| `internal/proxy/` | Core reverse proxy (`httputil.ReverseProxy`), SSE streaming, middleware chain |
| `internal/provider/` | Provider interface + OpenAI/Anthropic parsers, auto-detection from request |
| `internal/meter/` | Cost calculation engine, model pricing table, tiktoken-go fallback |
| `internal/ledger/` | Storage interface, SQLite (modernc.org/sqlite, CGO-free) + Postgres impls |
| `internal/budget/` | Budget enforcement middleware, circuit breaker |
| `internal/agent/` | Agent session tracking, loop/ghost detection — THE key differentiator |
| `internal/mcp/` | MCP call interception (stdio wrapper + SSE proxy) |
| `internal/config/` | YAML config loading (viper) |
| `internal/otel/` | OpenTelemetry metrics + Prometheus `/metrics` endpoint |
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

1. **Core Proxy** (Week 1-2) — proxy + token counting + cost ledger + CLI
2. **Budget Enforcement** (Week 3-4) — per-key budgets, circuit breaker, streaming-aware
3. **Agent Attribution** (Week 5-6) — session tracking, loop detection, ghost detection
4. **Observability** (Week 7-8) — OpenTelemetry, Prometheus, embedded web dashboard
5. **MCP Integration** (Week 9-10) — stdio wrapper + SSE proxy for MCP tool cost metering
6. **Polish & Launch** (Week 11-12) — Docker, GoReleaser, Helm, GitHub Action, docs

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
