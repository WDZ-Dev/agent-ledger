# Alerting

Get notified when budgets are approaching limits or agents are misbehaving.

## Configuration

```yaml
alerts:
  slack:
    webhook_url: "https://hooks.slack.com/services/T00/B00/xxx"
  webhooks:
    - url: "https://api.example.com/alerts"
      headers:
        Authorization: "Bearer token"
  cooldown_mins: 5
```

## Alert Types

| Type | Trigger |
|------|---------|
| `budget_warning` | Spend exceeds soft limit threshold |
| `budget_exceeded` | Spend exceeds hard limit |
| `loop_detected` | Agent making repetitive calls |
| `ghost_detected` | Long-running agent with high spend |

## Slack Notifications

Provide a Slack webhook URL and alerts are posted as formatted messages with severity, details, and timestamps.

## Webhook Notifications

Generic webhook support for any HTTP endpoint. Alerts are sent as JSON POST requests with custom headers for authentication.

## Deduplication

The `cooldown_mins` setting prevents alert spam. Once an alert is sent for a specific key (e.g., a particular API key exceeding its budget), the same alert won't fire again until the cooldown period expires.
