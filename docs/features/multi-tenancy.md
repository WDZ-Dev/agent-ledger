# Multi-Tenancy

Isolate costs, budgets, and dashboards by team or organization.

## Configuration

Enable tenancy and map API keys to tenants using glob patterns:

```yaml
tenants:
  enabled: true
  key_mappings:
    - api_key_pattern: "sk-proj-team-alpha-*"
      tenant_id: "alpha"
    - api_key_pattern: "sk-proj-team-beta-*"
      tenant_id: "beta"
```

## Header-Based Tenancy

Set the tenant per-request via header:

```
X-AgentLedger-Tenant: alpha
```

Header-based tenancy takes precedence over config-based key mapping.

## What's Isolated

- **Costs** — each tenant's spend is tracked separately
- **Budgets** — tenant-scoped budget rules (see [budgets](budgets.md))
- **Dashboard** — filter by tenant in the web UI
- **API endpoints** — all cost and stats endpoints accept `?tenant=` filter

## Without Tenancy

When tenancy is disabled (default), all costs are tracked globally. Existing behavior is unchanged — tenancy is fully opt-in.
