# Circuit Breaker

Protects against upstream provider failures. After a configurable number of consecutive 5xx responses, the circuit opens and rejects requests immediately — preventing wasted spend on failing APIs.

## Configuration

```yaml
circuit_breaker:
  max_failures: 5           # consecutive 5xx before opening
  timeout_secs: 30          # seconds before half-open retry
```

## States

| State | Behavior |
|-------|----------|
| **Closed** | Normal operation — requests forwarded to upstream |
| **Open** | Circuit tripped — requests rejected immediately with `503` |
| **Half-Open** | After `timeout_secs`, one request is allowed through to test recovery |

If the half-open request succeeds, the circuit closes. If it fails, the circuit opens again.

## When to Use

The circuit breaker is useful when:

- An upstream provider is experiencing an outage
- You want to fail fast rather than wait for timeouts
- You want to prevent agents from burning budget on requests that will fail

Omit the `circuit_breaker` section to disable.
