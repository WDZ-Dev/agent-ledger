# Quick Start

## 1. Start the proxy

```bash
# With defaults (listens on :8787, SQLite storage)
agentledger serve

# Or with a config file
agentledger serve -c agentledger.yaml
```

## 2. Point your agents at it

=== "Python (OpenAI)"

    ```bash
    export OPENAI_BASE_URL=http://localhost:8787/v1
    ```

    ```python
    import openai
    client = openai.OpenAI()  # picks up OPENAI_BASE_URL automatically
    ```

=== "Node.js"

    ```javascript
    const openai = new OpenAI({ baseURL: 'http://localhost:8787/v1' });
    ```

=== "Claude Code"

    ```bash
    export ANTHROPIC_BASE_URL=http://localhost:8787
    ```

=== "curl"

    ```bash
    curl http://localhost:8787/v1/chat/completions \
      -H "Authorization: Bearer $OPENAI_API_KEY" \
      -H "Content-Type: application/json" \
      -d '{"model": "gpt-4.1-mini", "messages": [{"role": "user", "content": "Hello"}]}'
    ```

For other providers, use path-prefix routing:

```bash
# Groq
curl http://localhost:8787/groq/v1/chat/completions

# Mistral
curl http://localhost:8787/mistral/v1/chat/completions

# DeepSeek
curl http://localhost:8787/deepseek/v1/chat/completions

# Gemini
curl http://localhost:8787/gemini/v1beta/models/gemini-2.5-pro:generateContent

# Cohere
curl http://localhost:8787/cohere/v2/chat
```

## 3. Check your costs

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

## 4. Set budgets (optional)

```yaml
# agentledger.yaml
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

When a limit is hit, the agent receives a `429` with a clear JSON error — no surprise charges.

## 5. Add agent tracking (optional)

Add headers to get per-agent cost attribution:

```python
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

Headers are stripped before forwarding to the provider. Without them, costs are tracked at the API-key level.

## 6. View the dashboard

Open [http://localhost:8787/](http://localhost:8787/) for real-time cost breakdowns, session views, and spending trends.

## Next steps

- [Configuration reference](../configuration/reference.md) — full YAML config
- [Providers](../features/providers.md) — all 15 supported providers
- [Budget enforcement](../features/budgets.md) — per-key limits, pre-flight estimation
- [Agent tracking](../features/agent-tracking.md) — sessions, loop detection, ghost detection
