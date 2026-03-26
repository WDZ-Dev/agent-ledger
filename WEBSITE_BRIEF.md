# AgentLedger Marketing Website Brief

> Give this file to Claude Code. It contains everything needed to build a marketing website for AgentLedger.

---

## Product Overview

**Name:** AgentLedger
**Tagline:** "Know what your agents cost."
**Subtitle:** Meter. Budget. Control.
**One-liner:** Open-source reverse proxy that gives you real-time cost attribution, budget enforcement, and financial observability for AI agents — without changing a single line of code.
**License:** Source Available (free for non-commercial use; commercial license required for enterprise)
**Language:** Go — ships as a single binary with zero runtime dependencies
**Repo:** https://github.com/WDZ-Dev/agent-ledger

---

## Target Audience

1. **Engineering teams** running AI agents in production (coding agents, support bots, data pipelines)
2. **Platform/DevOps engineers** responsible for AI infrastructure cost management
3. **Startups and enterprises** with multiple teams sharing LLM API keys
4. **Individual developers** who want visibility into what their agents actually cost

Pain points to hit:
- "We had a $2,000 weekend because an agent got stuck in a loop"
- "We have no idea which agent or team is responsible for our OpenAI bill"
- "LiteLLM is too heavy, too slow, and paywalls the features we need"
- "We need budget guardrails before giving agents to our whole team"

---

## Key Differentiators (vs LiteLLM — primary competitor)

| | AgentLedger | LiteLLM |
|---|---|---|
| **Architecture** | Go single binary, sub-10ms overhead | Python, documented memory leaks |
| **Cost model** | Per-agent-execution tracking | Per-key/user/team only |
| **Loop detection** | Built-in, zero-config | Not available |
| **Ghost agent detection** | Built-in | Not available |
| **Pre-flight estimation** | Rejects requests that would bust budget *before* the API call | Post-hoc only |
| **Budget enforcement** | Free, included | Enterprise paywall |
| **Audit logs** | Free, included | Enterprise paywall |
| **SSO** | Free, included | Enterprise paywall |
| **Setup** | `brew install` + one env var | Python + pip + database server |
| **Dependencies** | Zero (embedded SQLite, embedded dashboard) | PostgreSQL required, Redis recommended |

**Positioning statement:** AgentLedger is the cost control layer that LiteLLM charges enterprise pricing for — except it's free, faster, and built for agents from day one.

---

## Features

### Core (Hero Section)

1. **Zero Code Changes** — Set `OPENAI_BASE_URL=http://localhost:8787/v1` and you're done. Works with any OpenAI/Anthropic SDK.

2. **Real-Time Cost Tracking** — Every request metered, every token counted, every dollar attributed. 83+ models with up-to-date pricing.

3. **Budget Enforcement** — Daily and monthly limits per API key, per agent, or per tenant. Soft warnings at thresholds, hard blocks at limits.

4. **Pre-Flight Estimation** — Calculates worst-case cost *before* forwarding to the API. Rejects requests that would exceed your budget.

5. **Agent Session Tracking** — Group multi-call agent runs into sessions. See exactly what each execution costs.

6. **Loop Detection** — Automatically detects and stops runaway agents stuck in infinite loops. Zero configuration needed.

7. **Ghost Agent Detection** — Finds agents that are still running but haven't made a request in a while. Prevents silent cost bleed.

### Infrastructure

8. **15 LLM Providers** — OpenAI, Anthropic, Azure OpenAI, Google Gemini, Groq, Mistral, DeepSeek, Cohere, Together AI, Fireworks AI, Perplexity, OpenRouter, xAI (Grok), Cerebras, SambaNova

9. **Embedded Dashboard** — Real-time web UI for cost visibility. No external tools needed.

10. **Prometheus + Grafana** — OpenTelemetry metrics, `/metrics` endpoint, pre-built Grafana dashboard template.

11. **Multi-Tenancy** — Isolate costs by team or org. Tenant-scoped budgets and dashboard filtering.

12. **Alerting** — Slack and webhook notifications for budget warnings, loop detection, and anomalies.

13. **Rate Limiting** — Per-key request throttling with sliding window counters.

14. **Admin API** — Runtime budget rule management without restarts. API key blocklist with glob patterns.

15. **MCP Tool Metering** — Track costs of MCP (Model Context Protocol) tool calls alongside LLM usage.

16. **CSV/JSON Export** — Export cost data for accounting, compliance, or external analysis.

17. **Circuit Breaker** — Automatic upstream failure protection. Don't burn money on failed requests.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Your AI Agents                           │
│  (Coding assistants, support bots, data pipelines, etc.)        │
└─────────────────────┬───────────────────────────────────────────┘
                      │  OPENAI_BASE_URL=http://localhost:8787/v1
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      AgentLedger Proxy                          │
│                      (Go binary, :8787)                         │
│                                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │ Budget   │ │ Rate     │ │ Loop     │ │ Cost              │  │
│  │ Enforce  │ │ Limiter  │ │ Detector │ │ Calculator        │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │ Session  │ │ Tenant   │ │ Alert    │ │ Pre-flight        │  │
│  │ Tracker  │ │ Resolver │ │ Notifier │ │ Estimator         │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘  │
└──────────┬──────────────────────────────────┬───────────────────┘
           │                                  │
           ▼                                  ▼
┌─────────────────────┐          ┌────────────────────────────────┐
│  SQLite / Postgres   │          │  LLM Providers                │
│  (cost ledger)       │          │  OpenAI, Anthropic, Gemini,   │
│                      │          │  Groq, Mistral, DeepSeek,     │
│  Dashboard + CLI     │          │  Cohere, Azure, + 7 more      │
│  Prometheus /metrics │          └────────────────────────────────┘
│  Slack / Webhooks    │
└─────────────────────┘
```

---

## Quick Start (for website code examples)

### Install

```bash
# Homebrew
brew install wdz-dev/tap/agentledger

# Or download binary
curl -sSL https://github.com/WDZ-Dev/agent-ledger/releases/latest/download/agentledger_$(uname -s)_$(uname -m).tar.gz | tar xz

# Or Docker
docker run --rm -p 8787:8787 ghcr.io/wdz-dev/agent-ledger:latest

# Or from source
go install github.com/WDZ-Dev/agent-ledger/cmd/agentledger@latest
```

### Start the proxy

```bash
agentledger serve
```

### Point your agents at it (zero code changes)

```bash
# OpenAI SDK
export OPENAI_BASE_URL=http://localhost:8787/v1

# Anthropic SDK
export ANTHROPIC_BASE_URL=http://localhost:8787/anthropic

# Any other provider
export GROQ_BASE_URL=http://localhost:8787/groq/openai
export MISTRAL_BASE_URL=http://localhost:8787/mistral
```

### See what you're spending

```bash
# CLI
agentledger costs --last 24h --by agent

# Output:
# AGENT            REQUESTS  INPUT TOKENS  OUTPUT TOKENS  COST (USD)
# code-reviewer    47        125,340       42,100         $1.23
# pr-summarizer    12        8,200         3,400          $0.08
# data-pipeline    156       890,000       245,000        $8.41
# TOTAL            215       1,023,540     290,500        $9.72
```

### Set budgets

```yaml
# agentledger.yaml
budget:
  default:
    daily_limit_usd: 50.00
    monthly_limit_usd: 500.00
    soft_limit_pct: 0.8        # warn at 80%
    action: block              # block when exceeded
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      daily_limit_usd: 5.00   # tight limit for dev keys
    - tenant_id: "team-ml"
      daily_limit_usd: 200.00 # team-level budget
```

### Agent identification (optional but powerful)

```python
# Python SDK example — add headers to get per-agent tracking
import openai

client = openai.OpenAI(
    base_url="http://localhost:8787/v1",
    default_headers={
        "X-Agent-Id": "code-reviewer",
        "X-Agent-Session": f"sess_{run_id}",
        "X-Agent-User": "alice@company.com",
        "X-Agent-Task": "Review PR #456",
    }
)
```

---

## Supported Providers (full list)

| Provider | Type | Models | Default Upstream |
|----------|------|--------|-----------------|
| OpenAI | Native | GPT-5.x, GPT-4.1, GPT-4o, o3/o4, GPT-3.5 | api.openai.com |
| Anthropic | Native | Claude Opus 4.6, Sonnet 4.6, Haiku 4.5, Claude 3.x | api.anthropic.com |
| Azure OpenAI | Custom | All Azure-hosted OpenAI models | *.openai.azure.com |
| Google Gemini | Custom | Gemini 2.5 Pro/Flash, 2.0, 1.5 | generativelanguage.googleapis.com |
| Cohere | Custom | Command R+, Command R, Command Light | api.cohere.com |
| Groq | OpenAI-compat | Llama 3.3 70B, Mixtral, Gemma | api.groq.com |
| Mistral | OpenAI-compat | Large, Small, Codestral, Nemo | api.mistral.ai |
| DeepSeek | OpenAI-compat | DeepSeek Chat, Reasoner | api.deepseek.com |
| Together AI | OpenAI-compat | Llama, Qwen, DeepSeek | api.together.xyz |
| Fireworks AI | OpenAI-compat | Llama, Qwen | api.fireworks.ai |
| Perplexity | OpenAI-compat | Sonar Pro, Sonar, Reasoning | api.perplexity.ai |
| OpenRouter | OpenAI-compat | 200+ models via routing | openrouter.ai |
| xAI (Grok) | OpenAI-compat | Grok 3, Grok 3 Mini, Grok 2 | api.x.ai |
| Cerebras | OpenAI-compat | Llama 3.3 70B, Llama 3.1 8B | api.cerebras.ai |
| SambaNova | OpenAI-compat | Llama 3.3 70B, Llama 3.1 8B | api.sambanova.ai |

**83+ models** with built-in, up-to-date pricing tables (including GPT-5 family and Claude 4.5/4.6).

---

## Deployment Options

| Method | Command |
|--------|---------|
| **Homebrew** | `brew install wdz-dev/tap/agentledger` |
| **Binary** | Download from GitHub Releases (Linux, macOS, Windows — amd64 + arm64) |
| **Docker** | `docker run -p 8787:8787 ghcr.io/wdz-dev/agent-ledger:latest` |
| **Docker Compose** | `docker compose -f deploy/docker-compose.yml up` |
| **Kubernetes** | `helm install agentledger deploy/helm/agentledger` |
| **From source** | `go install github.com/WDZ-Dev/agent-ledger/cmd/agentledger@latest` |

---

## Stats & Numbers (for social proof / hero section)

- **15** LLM providers supported
- **83+** models with built-in pricing
- **Sub-10ms** proxy overhead
- **Zero** runtime dependencies
- **Zero** code changes required
- **Free** for personal and non-commercial use
- **6** platforms (Linux/macOS/Windows × amd64/arm64)

---

## Website Structure Suggestion

### Pages

1. **Hero / Landing** — Tagline, one-liner, architecture diagram, quick start, "star on GitHub" CTA
2. **Features** — Grid of all features with icons and short descriptions
3. **Providers** — Logo grid of all 15 supported providers
4. **Pricing** — Single card: "Free. Forever. All features included." Comparison table vs LiteLLM.
5. **Docs / Quick Start** — Installation, configuration, agent headers, CLI usage
6. **GitHub CTA** — Star button, contribution guide link

### Design Direction

- **Dark mode by default** (developer tool aesthetic)
- **Monospace accents** for code/terminal elements
- **Color palette:** Dark navy/slate background, green/teal accent (money/cost theme), white text
- **Terminal-style animations** for the quick start section (typing effect showing install → start → costs)
- **Minimal, fast, no bloat** — mirrors the product's philosophy
- **Responsive** — must work well on mobile

### Tone

- Direct, no-bullshit, technically credible
- Speak to engineers, not executives
- Show don't tell — code examples over marketing prose
- Confidence without arrogance: "We built what we needed. You probably need it too."

---

## Comparison Section Copy

### "Why not just check the OpenAI dashboard?"

The OpenAI dashboard shows you total org spend. It doesn't tell you:
- Which agent execution caused the $47 spike at 3am
- That your coding assistant is stuck in a loop burning $12/minute
- That the intern's test script has been running for 6 hours
- How much each team is spending vs their budget

AgentLedger answers all of these. In real-time. For every provider.

### "Why not LiteLLM?"

LiteLLM is a Python proxy with 50+ contributors and 15K+ stars. It's mature. It's also:
- **Slow** — Python + async overhead, documented memory leaks in production
- **Heavy** — requires PostgreSQL, Redis recommended, complex deployment
- **Paywalled** — budget tracking, audit logs, SSO, and guardrails require Enterprise ($$$)
- **Not agent-aware** — tracks costs per key or user, not per agent execution

AgentLedger is a single Go binary. Install it. Set one env var. Done.

---

## Social Proof Ideas (if applicable)

- GitHub stars count
- "Used in production by X teams"
- Cost savings testimonials: "Caught a $X runaway agent in the first week"
- Performance benchmarks: requests/sec, p99 latency overhead

---

## Technical Details for the Developer

- Use a static site generator (Next.js, Astro, or plain HTML/CSS/JS)
- Host on Vercel, Netlify, or GitHub Pages
- Keep it fast — no heavy frameworks, minimal JS
- Code syntax highlighting for YAML and bash examples
- Copy-to-clipboard on all code blocks
- GitHub repo link in header/nav
- Consider a Mintlify or Docusaurus setup if docs need to be more extensive
