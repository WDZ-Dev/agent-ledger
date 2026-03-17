# Docker Deployment

## Quick Start

```bash
docker run --rm -p 8787:8787 ghcr.io/wdz-dev/agent-ledger:latest
```

## With Persistent Storage

```bash
docker run -d \
  --name agentledger \
  -p 8787:8787 \
  -v agentledger-data:/data \
  ghcr.io/wdz-dev/agent-ledger:latest
```

## Docker Compose

A `docker-compose.yml` is included in the `deploy/` directory:

```bash
cd deploy && docker compose up
```

This starts AgentLedger with persistent volume storage and health checks.

## Custom Configuration

Mount a config file:

```bash
docker run -d \
  --name agentledger \
  -p 8787:8787 \
  -v agentledger-data:/data \
  -v ./agentledger.yaml:/etc/agentledger/agentledger.yaml \
  ghcr.io/wdz-dev/agent-ledger:latest
```

Or use environment variables:

```bash
docker run -d \
  --name agentledger \
  -p 8787:8787 \
  -e AGENTLEDGER_LISTEN=":8787" \
  -e AGENTLEDGER_STORAGE_DSN="/data/agentledger.db" \
  -e AGENTLEDGER_LOG_LEVEL="info" \
  ghcr.io/wdz-dev/agent-ledger:latest
```

## With PostgreSQL

For production deployments with multiple replicas, use PostgreSQL instead of SQLite:

```yaml
# docker-compose.yml
services:
  agentledger:
    image: ghcr.io/wdz-dev/agent-ledger:latest
    ports:
      - "8787:8787"
    environment:
      AGENTLEDGER_STORAGE_DRIVER: "postgres"
      AGENTLEDGER_STORAGE_DSN: "postgres://user:pass@postgres:5432/agentledger?sslmode=disable"
    depends_on:
      - postgres

  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: agentledger
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

## Health Check

The proxy responds to health checks at:

```
GET http://localhost:8787/healthz
```
