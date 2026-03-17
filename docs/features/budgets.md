# Budget Enforcement

Set daily and monthly spend limits per API key, per agent, or per tenant. When exceeded, requests are blocked before they reach the LLM provider.

## How It Works

1. Request arrives at the proxy
2. Budget manager checks current spend against limits
3. If over limit: return `429` immediately (no API call, no cost)
4. If approaching limit (soft limit): add warning header, forward request
5. If under limit: forward request normally

## Configuration

```yaml
budgets:
  default:
    daily_limit_usd: 50.0
    monthly_limit_usd: 500.0
    soft_limit_pct: 0.8        # warn at 80%
    action: "block"            # "block" returns 429, "warn" adds header only
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      daily_limit_usd: 5.0
      monthly_limit_usd: 50.0
      action: "block"
    - tenant_id: "alpha"
      daily_limit_usd: 100.0
      monthly_limit_usd: 1000.0
      action: "block"
```

## Block Response

When a limit is hit:

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

HTTP status: `429 Too Many Requests`

## Soft Limits

When `soft_limit_pct` is configured, AgentLedger adds a response header when approaching the threshold:

```
X-AgentLedger-Budget-Warning: daily spend at 82% of limit
```

The request is still forwarded — soft limits are informational only.

## Pre-Flight Estimation

AgentLedger calculates worst-case cost from `max_tokens` before forwarding to the API. If the estimated cost would exceed the remaining budget, the request is rejected immediately — no wasted spend.

## Per-Key Rules

Rules use glob patterns to match API keys:

| Pattern | Matches |
|---------|---------|
| `sk-proj-dev-*` | All keys starting with `sk-proj-dev-` |
| `sk-*` | All keys starting with `sk-` |
| `*` | All keys |

Rules are evaluated in order. The first matching rule wins. If no rule matches, the default applies.

## Runtime Management

Budget rules can be managed at runtime via the [Admin API](../admin-api.md) without restarting the proxy.
