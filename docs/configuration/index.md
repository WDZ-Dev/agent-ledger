# Configuration Overview

AgentLedger works out of the box with sensible defaults. All configuration is optional — the proxy starts with OpenAI and Anthropic enabled, SQLite storage, and the dashboard on.

## Config File Locations

AgentLedger looks for config in these locations (in order):

1. Path passed via `--config` / `-c` flag
2. `./agentledger.yaml`
3. `./configs/agentledger.yaml`
4. `~/.config/agentledger/agentledger.yaml`
5. `/etc/agentledger/agentledger.yaml`

## Minimal Config

No config file is needed for basic usage. To customize:

```yaml
listen: ":8787"

providers:
  openai:
    upstream: "https://api.openai.com"
    enabled: true
  anthropic:
    upstream: "https://api.anthropic.com"
    enabled: true

storage:
  driver: "sqlite"
  dsn: "data/agentledger.db"
```

## Environment Variable Overrides

All settings can be overridden with environment variables prefixed `AGENTLEDGER_`:

```bash
AGENTLEDGER_LISTEN=":9090"
AGENTLEDGER_STORAGE_DSN="/tmp/ledger.db"
AGENTLEDGER_LOG_LEVEL="debug"
```

## Full Reference

See [Full Reference](reference.md) for every configuration option with descriptions and defaults.
