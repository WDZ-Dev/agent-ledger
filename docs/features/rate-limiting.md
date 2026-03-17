# Rate Limiting

Throttle request volume per API key with sliding window counters. Budget enforcement limits spend; rate limiting limits request frequency.

## Configuration

```yaml
rate_limits:
  default:
    requests_per_minute: 60
    requests_per_hour: 1000
  rules:
    - api_key_pattern: "sk-proj-dev-*"
      requests_per_minute: 10
```

## How It Works

Rate limits use an in-memory sliding window counter keyed by API key hash. When a limit is exceeded, the request is rejected immediately with:

- HTTP status: `429 Too Many Requests`
- `Retry-After` header with seconds until the window resets

## Per-Key Rules

Rules use the same glob pattern matching as [budget rules](budgets.md). Rules are evaluated in order — the first match wins. If no rule matches, the default applies.

## Metrics

Rate-limited requests are tracked via the `agentledger_rate_limited_total` Prometheus metric.
