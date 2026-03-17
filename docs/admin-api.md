# Admin API

Manage budget rules and view API key usage at runtime without restarting the proxy.

## Enable

```yaml
admin:
  enabled: true
  token: "your-secret-admin-token"
```

## Authentication

All admin endpoints require a Bearer token:

```bash
curl -H "Authorization: Bearer your-secret-admin-token" \
  http://localhost:8787/api/admin/budgets/rules
```

## Endpoints

### Budget Rules

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/budgets/rules` | List all budget rules |
| `POST` | `/api/admin/budgets/rules` | Create a budget rule |
| `DELETE` | `/api/admin/budgets/rules?pattern=...` | Delete a rule by pattern |

#### Create a Rule

```bash
curl -X POST http://localhost:8787/api/admin/budgets/rules \
  -H "Authorization: Bearer your-secret-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "api_key_pattern": "sk-proj-dev-*",
    "daily_limit_usd": 5.0,
    "monthly_limit_usd": 50.0,
    "action": "block"
  }'
```

#### Delete a Rule

```bash
curl -X DELETE "http://localhost:8787/api/admin/budgets/rules?pattern=sk-proj-dev-*" \
  -H "Authorization: Bearer your-secret-admin-token"
```

### API Keys

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/api-keys` | List API key hashes with monthly spend |

### Providers

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/providers` | List provider status |

## Persistence

Runtime rules take effect immediately and persist across restarts. They are stored in the database and take precedence over YAML config rules.
