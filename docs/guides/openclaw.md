# OpenClaw + AgentLedger

Track what your OpenClaw agents cost. Set budgets. Kill runaway spend.

## The problem

OpenClaw agents run autonomously — managing email, scheduling meetings, browsing the web, executing code. Every action triggers LLM calls. A single agent session can make dozens of calls without any human in the loop.

Without cost visibility:

- A loop burns $50 before you notice
- Ghost agents keep running after you close the chat
- You check your provider dashboard the next morning and wonder what happened

## The fix: 60 seconds

### 1. Install AgentLedger

```bash
brew install wdz-dev/tap/agentledger
```

Or grab a binary from [GitHub Releases](https://github.com/WDZ-Dev/agent-ledger/releases).

### 2. Start the proxy

```bash
agentledger serve
```

AgentLedger is now listening on `localhost:8787`.

### 3. Configure OpenClaw

Add AgentLedger as a custom provider in `~/.openclaw/openclaw.json`:

#### OpenAI models through AgentLedger

```json
{
  "models": {
    "providers": {
      "agentledger-openai": {
        "baseUrl": "http://localhost:8787/v1",
        "apiKey": "${OPENAI_API_KEY}",
        "api": "openai-completions",
        "models": [
          { "id": "gpt-4o", "name": "GPT-4o", "contextWindow": 128000, "maxTokens": 16384 },
          { "id": "gpt-4.1", "name": "GPT-4.1", "contextWindow": 1047576, "maxTokens": 32768 },
          { "id": "gpt-4.1-mini", "name": "GPT-4.1 Mini", "contextWindow": 1047576, "maxTokens": 32768 }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": ["agentledger-openai/gpt-4o"]
    }
  }
}
```

#### Anthropic models through AgentLedger

```json
{
  "models": {
    "providers": {
      "agentledger-anthropic": {
        "baseUrl": "http://localhost:8787",
        "apiKey": "${ANTHROPIC_API_KEY}",
        "api": "anthropic-messages",
        "models": [
          { "id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4", "contextWindow": 200000, "maxTokens": 8192 },
          { "id": "claude-opus-4-20250514", "name": "Claude Opus 4", "contextWindow": 200000, "maxTokens": 32768 },
          { "id": "claude-haiku-4-5-20251001", "name": "Claude Haiku 4.5", "contextWindow": 200000, "maxTokens": 8192 }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": ["agentledger-anthropic/claude-sonnet-4-20250514"]
    }
  }
}
```

#### Multiple providers (recommended)

Combine both in one config. OpenClaw will use whichever model is configured per-agent:

```json
{
  "models": {
    "providers": {
      "agentledger-openai": {
        "baseUrl": "http://localhost:8787/v1",
        "apiKey": "${OPENAI_API_KEY}",
        "api": "openai-completions",
        "models": [
          { "id": "gpt-4o", "name": "GPT-4o", "contextWindow": 128000, "maxTokens": 16384 },
          { "id": "gpt-4.1-mini", "name": "GPT-4.1 Mini", "contextWindow": 1047576, "maxTokens": 32768 }
        ]
      },
      "agentledger-anthropic": {
        "baseUrl": "http://localhost:8787",
        "apiKey": "${ANTHROPIC_API_KEY}",
        "api": "anthropic-messages",
        "models": [
          { "id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4", "contextWindow": 200000, "maxTokens": 8192 }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": [
        "agentledger-openai/gpt-4.1-mini",
        "agentledger-anthropic/claude-sonnet-4-20250514"
      ]
    }
  }
}
```

Apply the config:

```bash
openclaw gateway config.apply --file ~/.openclaw/openclaw.json
```

### 4. Open the dashboard

Browse to [http://localhost:8787](http://localhost:8787). You'll see every LLM call your OpenClaw agents make — model, tokens, cost, latency — in real time.

### 5. Check costs from the CLI

```bash
agentledger costs
```

```
PROVIDER   MODEL                     REQUESTS   INPUT TOKENS   OUTPUT TOKENS   COST (USD)
--------   -----                     --------   ------------   -------------   ----------
openai     gpt-4.1-mini              87         174000         43500           $0.1392
anthropic  claude-sonnet-4-20250514  23         46000          11500           $0.3105
--------   -----                     --------   ------------   -------------   ----------
TOTAL                                110        220000         55000           $0.4497
```

## Set budget limits

OpenClaw agents are autonomous. They don't ask before making the next call. Set a safety net:

```yaml
# agentledger.yaml
budgets:
  default:
    daily_limit_usd: 10.0
    monthly_limit_usd: 100.0
    soft_limit_pct: 0.8
    action: "block"
```

- At **80%** of the limit, AgentLedger adds a warning header to responses
- At **100%**, requests are rejected with `429` — the agent stops spending

### Per-key limits

Give different OpenClaw API keys different budgets:

```yaml
budgets:
  rules:
    - api_key_pattern: "sk-proj-personal-*"
      daily_limit_usd: 5.0
    - api_key_pattern: "sk-proj-work-*"
      daily_limit_usd: 50.0
```

## Detect runaway agents

### Loop detection

If an OpenClaw agent gets stuck repeating the same action:

```yaml
agent:
  loop_threshold: 20
  loop_window_mins: 5
  loop_action: "block"    # or "warn"
```

AgentLedger detects the loop and blocks it after 20 repetitive calls in 5 minutes.

### Ghost detection

If an agent keeps running after you've closed the chat:

```yaml
agent:
  ghost_max_age_mins: 60
  ghost_min_calls: 50
  ghost_min_cost_usd: 1.0
```

## Get alerts

Get notified in Slack when something goes wrong:

```yaml
alerts:
  slack:
    webhook_url: "https://hooks.slack.com/services/..."
  cooldown_mins: 5
```

Alert types: `budget_warning`, `budget_exceeded`, `loop_detected`, `ghost_detected`.

## How it works

```
OpenClaw agent
    ↓ (sends normal API request)
AgentLedger proxy (localhost:8787)
    ↓ (meters tokens, checks budget, records cost)
OpenAI / Anthropic / Groq / ...
    ↓ (response)
AgentLedger proxy
    ↓ (forwards response unchanged)
OpenClaw agent
```

AgentLedger is a transparent reverse proxy. It doesn't modify requests or responses. Your API keys pass through untouched. The agent doesn't know it's there.

## FAQ

**Does this slow down my agents?**
No. AgentLedger adds ~0.1ms per request. Cost recording is fully async.

**Do I need to change my OpenClaw skills or agents?**
No. Just change the provider config. Everything else works the same.

**What if AgentLedger is down?**
Your agents will fail to reach the LLM. This is a feature — if your cost proxy is down, you probably want to know before racking up unmetered spend.

**Does it work with OpenClaw's multi-key rotation?**
Yes. AgentLedger tracks costs per API key fingerprint, so key rotation works transparently.

**Can I run this on a server?**
Yes. Deploy with Docker or Kubernetes and point your OpenClaw gateway at the remote address instead of localhost.
