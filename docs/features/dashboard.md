# Dashboard

AgentLedger includes an embedded web dashboard for real-time cost visibility. No external tools needed.

## Accessing the Dashboard

The dashboard is served at the proxy's root URL:

```
http://localhost:8787/
```

Enabled by default. To disable:

```yaml
dashboard:
  enabled: false
```

## Features

- **Summary cards** — today's spend, month's spend, request count, avg cost per request, active sessions, error rate
- **Cost over time** — line/area chart with selectable time ranges (30 min to 30 days)
- **Spend by provider** — doughnut chart showing cost distribution across providers
- **Cost breakdown** — table grouped by model, provider, agent, or session
- **Most expensive requests** — top 10 costliest individual API calls
- **Active sessions** — live view of running agent sessions with cost and status
- **Error breakdown** — 429s, 5xx errors, avg latency
- **API key usage** — spend per API key hash
- **Budget rules** — view and manage rules (requires admin token)

## Multi-Tenant Filtering

All dashboard views support tenant filtering. Enter a tenant ID in the filter bar to see costs for a specific team or organization.

The dashboard REST API endpoints also accept `?tenant=` query parameters.

## Dashboard API

The dashboard exposes these REST endpoints:

| Endpoint | Description |
|----------|-------------|
| `GET /api/dashboard/summary` | Summary cards data |
| `GET /api/dashboard/timeseries` | Cost over time chart data |
| `GET /api/dashboard/costs` | Cost breakdown table data |
| `GET /api/dashboard/sessions` | Active sessions list |
| `GET /api/dashboard/expensive` | Most expensive requests |
| `GET /api/dashboard/stats` | Error stats and avg cost |
| `GET /api/dashboard/export` | Export as CSV or JSON |
