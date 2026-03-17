# Agent Session Tracking

AgentLedger groups multi-call agent runs into sessions, enabling per-execution cost attribution. This is the key differentiator — most tools track cost per API key or per user. AgentLedger tracks cost per agent execution.

## Agent Headers

Tag requests with agent metadata using HTTP headers:

```
X-Agent-Id: code-reviewer
X-Agent-Session: sess_abc123
X-Agent-User: user@example.com
X-Agent-Task: "Review PR #456"
```

All agent headers are stripped before forwarding to the upstream provider.

### Python Example

```python
client = openai.OpenAI(
    base_url="http://localhost:8787/v1",
    default_headers={
        "X-Agent-Id": "code-reviewer",
        "X-Agent-Session": f"sess_{run_id}",
        "X-Agent-User": "alice@company.com",
        "X-Agent-Task": "Review PR #456",
    }
)
```

### Fallback

Without agent headers, costs are tracked at the API-key level. Headers are optional but recommended for per-agent visibility.

## Loop Detection

Automatically detects runaway agents making repetitive calls to the same endpoint.

```yaml
agent:
  loop_threshold: 20        # same path N times in window = loop
  loop_window_mins: 5       # sliding window
  loop_action: "warn"       # "warn" logs + alert, "block" returns 429
```

When a loop is detected:

- **warn**: logs a warning and sends an alert (if [alerting](alerting.md) is configured)
- **block**: returns `429 Too Many Requests` to stop the agent

Set `loop_threshold: 0` to disable.

## Ghost Agent Detection

Finds agents that are still running but may have been forgotten — burning tokens silently.

```yaml
agent:
  ghost_max_age_mins: 60    # sessions older than this are candidates
  ghost_min_calls: 50       # minimum calls before flagging
  ghost_min_cost_usd: 1.0   # minimum spend before flagging
```

A session is flagged as a ghost when all three thresholds are met: it's been running longer than `ghost_max_age_mins`, has made more than `ghost_min_calls`, and has spent more than `ghost_min_cost_usd`.

Set `ghost_max_age_mins: 0` to disable.

## Session Lifecycle

```yaml
agent:
  session_timeout_mins: 30  # auto-expire idle sessions
```

Sessions are automatically expired after `session_timeout_mins` of inactivity. Active sessions are visible in the dashboard and via the sessions API endpoint.
