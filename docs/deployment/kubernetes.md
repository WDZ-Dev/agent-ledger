# Kubernetes / Helm

## Install

```bash
helm install agentledger deploy/helm/agentledger
```

## Key Values

| Value | Description | Default |
|-------|-------------|---------|
| `replicaCount` | Number of proxy replicas | `1` |
| `image.repository` | Container image | `ghcr.io/wdz-dev/agent-ledger` |
| `image.tag` | Image tag | Chart appVersion |
| `service.port` | Service port | `8787` |
| `ingress.enabled` | Enable ingress | `false` |
| `persistence.enabled` | Enable PVC for SQLite | `true` |
| `persistence.size` | PVC size | `1Gi` |

## Custom Values

```bash
helm install agentledger deploy/helm/agentledger \
  --set replicaCount=2 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=agentledger.example.com
```

Or with a values file:

```yaml
# values-prod.yaml
replicaCount: 3

ingress:
  enabled: true
  hosts:
    - host: agentledger.example.com
      paths:
        - path: /
          pathType: Prefix

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

```bash
helm install agentledger deploy/helm/agentledger -f values-prod.yaml
```

## SQLite vs PostgreSQL

!!! warning "SQLite with Multiple Replicas"
    SQLite is single-writer. If you run multiple replicas, use PostgreSQL instead.

For PostgreSQL:

```yaml
# values-prod.yaml
env:
  - name: AGENTLEDGER_STORAGE_DRIVER
    value: "postgres"
  - name: AGENTLEDGER_STORAGE_DSN
    value: "postgres://user:pass@postgres:5432/agentledger?sslmode=disable"

persistence:
  enabled: false
```
