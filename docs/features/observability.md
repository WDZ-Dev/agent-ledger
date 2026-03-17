# Observability

AgentLedger exports OpenTelemetry metrics via a Prometheus endpoint for integration with your existing monitoring stack.

## Prometheus Endpoint

Metrics are exposed at:

```
http://localhost:8787/metrics
```

## Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `agentledger_requests_total` | Counter | Total proxied requests by provider, model, status |
| `agentledger_request_duration_ms` | Histogram | Request latency |
| `agentledger_input_tokens_total` | Counter | Total input tokens |
| `agentledger_output_tokens_total` | Counter | Total output tokens |
| `agentledger_cost_usd_total` | Counter | Total cost in USD |
| `agentledger_sessions_active` | Gauge | Currently active agent sessions |
| `agentledger_loop_detected_total` | Counter | Loop detection events |
| `agentledger_ghost_detected_total` | Counter | Ghost detection events |
| `agentledger_rate_limited_total` | Counter | Rate-limited requests |
| `agentledger_mcp_calls_total` | Counter | MCP tool calls |

## Grafana

A pre-built Grafana dashboard template is included at `deploy/grafana/agentledger.json`. Import it into your Grafana instance for panels covering:

- Total spend (gauge)
- Spend rate over time (graph)
- Requests by provider (pie chart)
- Top models by cost (table)
- Active sessions (stat)

## Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: "agentledger"
    static_configs:
      - targets: ["localhost:8787"]
```
