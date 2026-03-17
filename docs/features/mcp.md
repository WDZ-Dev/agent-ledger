# MCP Tool Metering

Track costs of MCP (Model Context Protocol) tool calls alongside LLM usage. Two modes are available.

## HTTP Proxy Mode

Forward JSON-RPC requests to an upstream MCP server, metering each tool call:

```yaml
mcp:
  enabled: true
  upstream: "http://localhost:3000"
  pricing:
    - server: "filesystem"
      tool: "read_file"
      cost_per_call: 0.01
    - server: "filesystem"
      tool: ""                  # wildcard: all tools on this server
      cost_per_call: 0.005
    - server: "github"
      tool: ""
      cost_per_call: 0.02
```

## Stdio Wrapper Mode

Wrap any MCP server process and intercept tool calls via stdio:

```bash
agentledger mcp-wrap -- npx @modelcontextprotocol/server-filesystem /tmp
```

This launches the MCP server as a child process, intercepts JSON-RPC messages on stdin/stdout, records tool call costs, and forwards everything transparently.

## Pricing Rules

Rules are matched in order. The first matching `server` + `tool` combination wins.

| server | tool | Matches |
|--------|------|---------|
| `"filesystem"` | `"read_file"` | Exact match: filesystem server, read_file tool |
| `"filesystem"` | `""` | Wildcard: any tool on the filesystem server |
| `""` | `""` | Catch-all: any server, any tool |

## Viewing MCP Costs

MCP tool costs appear alongside LLM costs in:

- The CLI: `agentledger costs`
- The dashboard
- Prometheus metrics: `agentledger_mcp_calls_total`
- Export: `agentledger export`
