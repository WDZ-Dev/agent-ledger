---
hide:
  - toc
---

<div class="al-landing" markdown>

<div class="al-hero" markdown>

<div class="al-logo-mark">
<svg viewBox="0 0 80 80" fill="none" xmlns="http://www.w3.org/2000/svg" width="64" height="64">
  <defs><clipPath id="al-hero-clip"><circle cx="40" cy="40" r="12"/></clipPath></defs>
  <path d="M 52,40 Q 61,45 70,40" stroke="#3fb950" stroke-width="1.8" stroke-linecap="round"/>
  <path d="M 48.5,48.5 Q 51.3,58.3 61.2,61.2" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.8"/>
  <path d="M 40,52 Q 35,61 40,70" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.8"/>
  <path d="M 31.5,48.5 Q 20.6,50.3 18.8,61.2" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.6"/>
  <path d="M 28,40 Q 19,35 10,40" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.6"/>
  <path d="M 31.5,31.5 Q 29.6,20.6 18.8,18.8" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.7"/>
  <path d="M 40,28 Q 45,19 40,10" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.7"/>
  <path d="M 48.5,31.5 Q 59.3,29.6 61.2,18.8" stroke="#388bfd" stroke-width="1.8" stroke-linecap="round" opacity="0.9"/>
  <circle cx="70" cy="40" r="2.8" fill="#3fb950"/>
  <circle cx="61.2" cy="61.2" r="2.5" fill="#388bfd" opacity="0.7"/>
  <circle cx="40" cy="70" r="2.5" fill="#388bfd" opacity="0.7"/>
  <circle cx="18.8" cy="61.2" r="2.5" fill="#388bfd" opacity="0.55"/>
  <circle cx="10" cy="40" r="2.5" fill="#388bfd" opacity="0.55"/>
  <circle cx="18.8" cy="18.8" r="2.5" fill="#388bfd" opacity="0.65"/>
  <circle cx="40" cy="10" r="2.5" fill="#388bfd" opacity="0.65"/>
  <circle cx="61.2" cy="18.8" r="2.5" fill="#388bfd" opacity="0.85"/>
  <circle cx="40" cy="40" r="12.5" fill="#0a0e14"/>
  <g clip-path="url(#al-hero-clip)">
    <line x1="28" y1="48" x2="52" y2="48" stroke="#388bfd" stroke-width="0.8" opacity="0.45"/>
    <rect x="32" y="38" width="4" height="10" rx="0.8" fill="#388bfd" opacity="0.6"/>
    <rect x="38" y="42" width="4" height="6" rx="0.8" fill="#3fb950"/>
    <rect x="44" y="34" width="4" height="14" rx="0.8" fill="#388bfd" opacity="0.92"/>
    <line x1="28" y1="40" x2="52" y2="40" stroke="#e6edf3" stroke-width="0.9" stroke-dasharray="2 1.5" opacity="0.55"/>
    <line x1="30.5" y1="37.5" x2="30.5" y2="42.5" stroke="#e6edf3" stroke-width="1" stroke-linecap="round" opacity="0.55"/>
  </g>
  <circle cx="40" cy="40" r="12.5" fill="none" stroke="#388bfd" stroke-width="2"/>
</svg>
</div>

# AgentLedger

**Know what your agents cost.**{ .al-tagline }

The open-source reverse proxy that gives you real-time cost tracking, budget enforcement, and financial observability for every AI agent call — without changing a single line of code.

</div>

<div class="al-divider"></div>

<div class="al-features" markdown>

<div class="al-feature" markdown>
<div class="al-feature-icon">$</div>

### Per-agent cost tracking
Every LLM call attributed to the agent, session, and user that triggered it. Not just per-key — per-execution.
</div>

<div class="al-feature" markdown>
<div class="al-feature-icon">//</div>

### Budget enforcement
Set daily and monthly limits. Requests that would exceed your budget get blocked before they hit the API.
</div>

<div class="al-feature" markdown>
<div class="al-feature-icon">~</div>

### Loop & ghost detection
Automatically detect runaway agents stuck in loops and ghost processes silently burning tokens.
</div>

<div class="al-feature" markdown>
<div class="al-feature-icon">&gt;</div>

### Zero code changes
Point your existing SDK at the proxy with one environment variable. Works with OpenAI, Anthropic, and 13 more providers.
</div>

</div>

<div class="al-divider"></div>

<div class="al-stats" markdown>

<div class="al-stat" markdown>
**15**{ .al-stat-number }

LLM providers
</div>

<div class="al-stat" markdown>
**83+**{ .al-stat-number }

Models with built-in pricing
</div>

<div class="al-stat" markdown>
**<10ms**{ .al-stat-number }

Proxy overhead
</div>

<div class="al-stat" markdown>
**0**{ .al-stat-number }

Dependencies
</div>

</div>

<div class="al-divider"></div>

<div class="al-how" markdown>

### How it works

```
Your agents ──→ AgentLedger proxy ──→ LLM APIs
                     │
                     ├── Track costs per agent
                     ├── Enforce budgets
                     ├── Detect loops
                     └── Dashboard + alerts
```

One binary. One env var. Full visibility.

</div>

<div class="al-divider"></div>

<div class="al-bottom-cta" markdown>

### Get started

```bash
brew install wdz-dev/tap/agentledger
agentledger serve
```

Then point your agent at the proxy:

```bash
export OPENAI_BASE_URL=http://localhost:8787/v1
```

[Quick Start](getting-started/quickstart.md){ .al-btn }
[Installation](getting-started/installation.md){ .al-btn .al-btn--secondary }

</div>

</div>
