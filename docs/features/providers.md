# Providers

AgentLedger supports 15 LLM providers with 83+ models and built-in pricing.

## Supported Providers

| Provider | Routing | Type | Models |
|----------|---------|------|--------|
| OpenAI | `/v1/` (default) | Native | GPT-4.1, GPT-4.1-mini, GPT-4o, o3, o4-mini, GPT-3.5-turbo |
| Anthropic | `/v1/messages` | Native | Claude Opus 4, Sonnet 4, Haiku 4, Claude 3.5/3.x |
| Azure OpenAI | `/azure/` | Custom | All Azure-hosted OpenAI models |
| Google Gemini | `/gemini/` | Custom | Gemini 2.5 Pro/Flash, 2.0, 1.5 |
| Cohere | `/cohere/` | Custom | Command R+, Command R, Command Light |
| Groq | `/groq/v1/` | OpenAI-compat | Llama 3.3 70B, Mixtral, Gemma |
| Mistral | `/mistral/v1/` | OpenAI-compat | Large, Small, Codestral, Nemo |
| DeepSeek | `/deepseek/v1/` | OpenAI-compat | DeepSeek Chat, Reasoner |
| Together AI | `/together/v1/` | OpenAI-compat | Llama, Qwen, DeepSeek |
| Fireworks AI | `/fireworks/v1/` | OpenAI-compat | Llama, Qwen |
| Perplexity | `/perplexity/v1/` | OpenAI-compat | Sonar Pro, Sonar, Reasoning |
| OpenRouter | `/openrouter/v1/` | OpenAI-compat | 200+ models via routing |
| xAI (Grok) | `/xai/v1/` | OpenAI-compat | Grok 3, Grok 3 Mini, Grok 2 |
| Cerebras | `/cerebras/v1/` | OpenAI-compat | Llama 3.3 70B, Llama 3.1 8B |
| SambaNova | `/sambanova/v1/` | OpenAI-compat | Llama 3.3 70B, Llama 3.1 8B |

## How Routing Works

**OpenAI** is the default — requests to `/v1/chat/completions` route to OpenAI.

**Anthropic** is detected by the `/v1/messages` path.

**All other providers** use path-prefix routing. A request to `/groq/v1/chat/completions` is routed to Groq. The prefix is stripped before forwarding, so Groq's API sees `/v1/chat/completions`.

## Provider Types

- **Native** — OpenAI and Anthropic have dedicated parsers for their specific API formats.
- **OpenAI-compatible** — Groq, Mistral, DeepSeek, Together, Fireworks, Perplexity, OpenRouter, xAI, Cerebras, and SambaNova all use the OpenAI `/v1/chat/completions` format. They share a common parser.
- **Custom** — Gemini and Cohere have unique API formats and get dedicated parsers.

## Configuration

OpenAI and Anthropic are enabled by default. Additional providers go in `providers.extra`:

```yaml
providers:
  openai:
    upstream: "https://api.openai.com"
    enabled: true
  anthropic:
    upstream: "https://api.anthropic.com"
    enabled: true
  extra:
    groq:
      type: "openai"
      upstream: "https://api.groq.com/openai"
      path_prefix: "/groq"
      enabled: true
```

See the [full reference](../configuration/reference.md) for all provider entries.

## API Key Handling

API keys pass through to the upstream provider untouched. AgentLedger never stores raw keys — it creates a SHA-256 fingerprint from the first 8 and last 4 characters for attribution and reporting.

## Model Matching

Versioned model names (e.g., `gpt-4o-2024-11-20`) are matched via longest prefix to the pricing table. If a model isn't found, costs are recorded with a fallback of $0 and a warning is logged.
