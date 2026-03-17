# AgentLedger

**Know what your agents cost.** Meter. Budget. Control.

AgentLedger is an open-source reverse proxy that gives you real-time cost attribution, budget enforcement, and financial observability for AI agents — without changing a single line of code.

```bash
export OPENAI_BASE_URL=http://localhost:8787/v1
# That's it. Your agents now have cost tracking and budget enforcement.
```

---

## Why AgentLedger?

AI agents make dozens of LLM calls per task. Costs compound fast, loops happen silently, and provider dashboards only show you the damage after the fact.

- **Real-time cost tracking** — every request metered, every token counted
- **Budget enforcement** — daily and monthly limits with automatic blocking
- **Pre-flight estimation** — rejects requests that would exceed your budget before they hit the API
- **Agent session tracking** — group multi-call agent runs into sessions, detect loops and ghost agents
- **15 LLM providers** — OpenAI, Anthropic, Gemini, Groq, Mistral, DeepSeek, Cohere, and more
- **Zero code changes** — works with any OpenAI/Anthropic SDK via base URL override

---

## Architecture

```
  Agents (any SDK) --+
                     |     +--------------------+     OpenAI
  MCP Servers -------+---> |  AgentLedger :8787 | --> Anthropic
  (stdio/HTTP)       |     |                    |     Groq, Mistral
                     |     |  Budget limits     |     DeepSeek, Gemini
                     |     |  Rate limiting     |     Cohere, + 8 more
                     |     |  Token metering    |
                     |     |  Agent sessions    | --> Slack / Webhooks
                     |     |  Cost recording    |
                     |     |                    |
                     |     |  SQLite/Postgres   |
                     |     |  Dashboard         |
                     |     |  Prometheus        |
                     |     +--------------------+
```

---

## At a Glance

| | |
|---|---|
| **Providers** | 15 LLM providers, 83+ models with built-in pricing |
| **Overhead** | Sub-10ms proxy latency (~0.1ms typical) |
| **Dependencies** | Zero — single Go binary with embedded SQLite and dashboard |
| **Setup** | One environment variable, zero code changes |
| **License** | Apache 2.0 — all features free and open-source |
| **Platforms** | Linux, macOS, Windows (amd64 + arm64) |

---

## vs Competitors

| | AgentLedger | Typical LLM proxy |
|---|---|---|
| **Architecture** | Go single binary, sub-10ms overhead | Python runtime, external dependencies |
| **Cost model** | Per-agent-execution tracking | Per-key/user/team only |
| **Loop detection** | Built-in, zero-config | Not available |
| **Ghost agent detection** | Built-in | Not available |
| **Pre-flight estimation** | Rejects before API call | Post-hoc only |
| **Budget enforcement** | Free, included | Often paywalled |
| **Audit logs** | Free, included | Often paywalled |
| **Setup** | `brew install` + one env var | Runtime + package manager + database server |
| **Dependencies** | Zero (embedded SQLite + dashboard) | External database required |

---

## Quick Start

```bash
# Install
brew install wdz-dev/tap/agentledger

# Start the proxy
agentledger serve

# Point your agents at it
export OPENAI_BASE_URL=http://localhost:8787/v1

# Check your costs
agentledger costs
```

[Get started](getting-started/installation.md){ .md-button .md-button--primary }
[View on GitHub](https://github.com/WDZ-Dev/agent-ledger){ .md-button }
