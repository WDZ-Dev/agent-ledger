# Roadmap

## Completed

- [x] **Phase 1: Core Proxy** — Reverse proxy, token metering, cost calculation, SQLite storage, CLI
- [x] **Phase 2: Budget Enforcement** — Per-key budgets, pre-flight estimation, circuit breaker
- [x] **Phase 3: Agent Attribution** — Session tracking, loop detection, ghost agent detection
- [x] **Phase 4: Observability** — OpenTelemetry metrics, Prometheus endpoint, web dashboard
- [x] **Phase 5: MCP Integration** — Meter MCP tool calls alongside LLM costs
- [x] **Phase 6: Polish & Launch** — Docker, GoReleaser, Helm chart, docs
- [x] **Phase 7: Multi-Provider** — Groq, Mistral, DeepSeek, Gemini, Cohere with path-prefix routing
- [x] **Phase 8: Postgres** — Production-grade PostgreSQL storage backend
- [x] **Phase 9: Multi-Tenancy** — Tenant isolation with header and config-based resolution
- [x] **Phase 10: Alerting** — Slack and webhook notifications with deduplication
- [x] **Phase 11: Rate Limiting** — Per-key request throttling + Homebrew tap
- [x] **Phase 12: Admin API** — Runtime budget rule management

## Future

| Feature | Value | Complexity |
|---------|-------|------------|
| Response Caching | High | High |
| WebSocket Live Dashboard | Medium | Medium |
| Plugin / Middleware System | Medium | Very High |

See [ROADMAP.md](https://github.com/WDZ-Dev/agent-ledger/blob/main/ROADMAP.md) on GitHub for detailed descriptions of each planned feature.
