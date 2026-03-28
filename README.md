# AgentLedger

**Know what your agents cost.** Meter. Budget. Control.

AgentLedger is a reverse proxy that sits between your AI agents and LLM providers, tracking every token, calculating costs, and enforcing budgets — all without changing a single line of your application code.

<p align="center">
  <img src="promo/demo.gif" alt="AgentLedger Demo" width="960">
</p>

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
```

See the [provider documentation](https://wdz-dev.github.io/agent-ledger/docs/features/providers/) for all 15 supported providers.

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

## Features

See the [full feature documentation](https://wdz-dev.github.io/agent-ledger/docs/features/providers/) for detailed configuration of all features including cost tracking, budget enforcement, agent sessions, MCP metering, multi-tenancy, alerting, rate limiting, and the admin API.

## Configuration

See [`configs/agentledger.example.yaml`](configs/agentledger.example.yaml) for the full configuration reference, or visit the [configuration docs](https://wdz-dev.github.io/agent-ledger/docs/configuration/).

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

```bash
make build    # Build binary to bin/agentledger
make test     # Run all tests with race detection
make dev      # Build and run with example config
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Run checks (`make check`)
4. Commit your changes
5. Open a pull request

## License

Source Available — free for personal and non-commercial use. Commercial/enterprise use requires a license. See [LICENSE](LICENSE) for details, or contact [wdzdevgroup@gmail.com](mailto:wdzdevgroup@gmail.com) for pricing.
