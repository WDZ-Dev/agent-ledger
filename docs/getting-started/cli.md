# CLI Reference

## `agentledger serve`

Start the reverse proxy.

```bash
agentledger serve [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Path to config file | Auto-detect (see [config overview](../configuration/index.md)) |

## `agentledger costs`

Show a cost report from the ledger.

```bash
agentledger costs [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Path to config file | Auto-detect |
| `--last` | Time window: `1h`, `24h`, `7d`, `30d` | `24h` |
| `--by` | Group by: `model`, `provider`, `key` | `model` |

Example:

```bash
agentledger costs --last 7d --by provider
```

## `agentledger export`

Export cost data as CSV or JSON.

```bash
agentledger export [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Path to config file | Auto-detect |
| `--format` | Output format: `csv`, `json` | `json` |
| `--last` | Time window: `1h`, `24h`, `7d`, `30d` | `30d` |
| `--by` | Group by: `model`, `provider`, `key`, `agent`, `session` | `model` |

## `agentledger mcp-wrap`

Wrap an MCP server process for tool call metering via stdio.

```bash
agentledger mcp-wrap [flags] -- command [args...]
```

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Path to config file | Auto-detect |

Example:

```bash
agentledger mcp-wrap -- npx @modelcontextprotocol/server-filesystem /tmp
```

## `agentledger version`

Print the version string.

```bash
agentledger version
```

## Environment Variables

All config settings can be overridden with environment variables prefixed `AGENTLEDGER_`:

```bash
AGENTLEDGER_LISTEN=":9090"
AGENTLEDGER_STORAGE_DSN="/tmp/ledger.db"
AGENTLEDGER_LOG_LEVEL="debug"
```
