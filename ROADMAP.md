# AgentLedger — Future Roadmap

Phases 1–12 are complete. This document captures potential future features, roughly prioritized by value.

---

## Per-Tenant Budgets

**Problem:** Budgets are currently per-API-key. Organizations sharing an instance with tenants enabled can't set spend limits at the team/org level.

**Scope:**
- Extend `BudgetRuleConfig` with optional `TenantID` field
- `budget.Manager.Check()` looks up tenant-scoped spend via `GetTotalSpendByTenant()` when a tenant is present
- Admin API: CRUD tenant budget rules at `/api/admin/tenants/{id}/budget`
- Dashboard: per-tenant budget visualization

**Complexity:** Medium — the query plumbing exists (`GetTotalSpendByTenant`), but the budget manager needs a second lookup path and the admin API needs new endpoints.

---

## OpenAI Responses API

**Problem:** OpenAI's newer Responses API (`/v1/responses`) uses a different request/response format than Chat Completions. Agents using this endpoint get proxied but not metered.

**Scope:**
- New parser in `openai_compat.go` or a separate `openai_responses.go`
- `Match()` detects `/v1/responses` path
- `ParseRequest()`: extract `model`, `instructions`, `input` fields
- `ParseResponse()`: extract `usage.input_tokens`, `output_tokens`, `reasoning_tokens`
- Streaming: SSE events with typed `response.*` events; usage in `response.completed`

**Complexity:** Medium — similar structure to existing parsers but different field names and streaming event types.

---

## Response Caching

**Problem:** Identical prompts sent by multiple agents (or retries) hit the API and incur cost every time.

**Scope:**
- New `internal/cache/` package with pluggable backends (in-memory LRU, Redis)
- Cache key: hash of `(model, messages, temperature, tools)` — must exclude non-deterministic fields
- Cache-Control semantics: `X-AgentLedger-Cache: hit/miss` response header
- Config: `cache.enabled`, `cache.ttl`, `cache.max_size_mb`, `cache.backend`
- Budget integration: cached responses cost $0 and don't count toward limits
- Metrics: `agentledger_cache_hits_total`, `agentledger_cache_misses_total`

**Complexity:** High — cache invalidation is hard; need to handle streaming responses, tool calls, and non-deterministic outputs carefully.

---

## WebSocket Live Dashboard

**Problem:** Dashboard polls every 30 seconds. No real-time visibility into active spend.

**Scope:**
- New `/api/dashboard/ws` endpoint using `gorilla/websocket` or `nhooyr.io/websocket`
- Server pushes events: `cost_recorded`, `session_started`, `session_ended`, `budget_warning`, `loop_detected`
- Frontend: live-updating cost counter, session activity feed, toast notifications for alerts
- Fallback: keep polling for browsers that don't support WebSocket

**Complexity:** Medium — straightforward WebSocket hub pattern, but needs careful connection lifecycle management.

---

## CSV/JSON Export

**Problem:** No way to export cost data for external analysis, accounting, or compliance.

**Scope:**
- CLI: `agentledger export --format csv --last 30d --by model > costs.csv`
- Dashboard API: `GET /api/dashboard/costs?format=csv` returns downloadable CSV
- Fields: timestamp, provider, model, api_key_hash, agent_id, session_id, tenant_id, input_tokens, output_tokens, cost_usd
- Support date range, group-by, and tenant filters

**Complexity:** Low — query plumbing exists; just needs a CSV encoder and CLI flag.

---

## API Key Rotation & Revocation

**Problem:** No way to block a compromised API key at the proxy level without changing upstream provider settings.

**Scope:**
- Admin API: `POST /api/admin/api-keys/block` with key hash or glob pattern
- Blocked keys get 403 immediately (before budget check or upstream call)
- Stored in `admin_config` table as a blocklist
- Optional: key alias mapping (friendly names for key hashes in dashboard)
- Optional: key rotation tracking — detect when a key hash changes for the same agent

**Complexity:** Low — blocklist check is a simple lookup in the proxy hot path.

---

## Postgres Testcontainer CI

**Problem:** Postgres integration tests require a running Postgres instance. CI only runs SQLite tests.

**Scope:**
- Add `testcontainers-go` to test dependencies
- Create shared `testhelpers.NewPostgresContainer(t)` that spins up a Postgres container and returns a DSN
- Modify `postgres_test.go` to use the container when `POSTGRES_TEST_DSN` is not set
- Add a CI workflow step that runs `go test -tags integration ./internal/ledger/`

**Complexity:** Low — `testcontainers-go` handles all the Docker lifecycle.

---

## Grafana Dashboard Template

**Problem:** Users running Prometheus + Grafana have to build dashboards from scratch.

**Scope:**
- JSON dashboard template in `deploy/grafana/agentledger.json`
- Panels: total spend (gauge), spend rate (graph), requests by provider (pie), top models by cost (table), active sessions (stat), alert history (log)
- Variables: `$interval`, `$provider`, `$model`
- README section with import instructions

**Complexity:** Low — no code changes, just a JSON template.

---

## Plugin / Middleware System

**Problem:** Users can't extend AgentLedger without forking. Custom logic (auth, routing, transformations) requires code changes.

**Scope:**
- Define `Middleware` interface: `func(next http.Handler) http.Handler`
- Config-driven middleware chain: `middlewares: [auth, transform, custom]`
- Built-in middlewares: request logging, header injection, response transformation
- Optional: Go plugin support (`plugin.Open`) for compiled extensions
- Optional: Lua/Wasm scripting for lightweight custom logic

**Complexity:** Very high — plugin systems are notoriously hard to get right. Start with the middleware interface and built-in options; defer scripting.

---

## Priority Matrix

| Feature | Value | Complexity | Recommendation |
|---------|-------|------------|----------------|
| CSV/JSON Export | High | Low | Do first |
| API Key Revocation | High | Low | Do first |
| Per-Tenant Budgets | High | Medium | Do second |
| Postgres Testcontainer CI | Medium | Low | Do second |
| OpenAI Responses API | High | Medium | Do second |
| Grafana Dashboard | Medium | Low | Do anytime |
| WebSocket Dashboard | Medium | Medium | Do later |
| Response Caching | High | High | Do later |
| Plugin System | Medium | Very High | Do much later |
