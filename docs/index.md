---
hide:
  - navigation
  - toc
  - footer
---

<div class="al-landing" markdown>

<div class="al-hero" markdown>

<div class="al-logo-mark">AL</div>

# AgentLedger

**Know what your agents cost.**{ .al-tagline }

The open-source reverse proxy that gives you real-time cost tracking, budget enforcement, and financial observability for every AI agent call — without changing a single line of code.

<div class="al-waitlist-form" markdown>

<form class="al-form">
  <input type="email" name="email" placeholder="you@company.com" required class="al-input" />
  <button type="submit" class="al-btn">Join the waitlist</button>
</form>

<p class="al-form-note">Be the first to know when we launch. No spam.</p>

</div>

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

<div class="al-usecase" markdown>

### Your OpenClaw agent spent $47 last night.

OpenClaw agents run autonomously — managing email, scheduling meetings, browsing the web. Every action triggers LLM calls. Without visibility, costs spiral silently.

AgentLedger sits between OpenClaw and the LLM providers. Set a $10 daily budget and AgentLedger kills runaway spend before it hits your bill.

<div class="al-usecase-steps" markdown>

<div class="al-usecase-step" markdown>
**1. Install**

```
brew install wdz-dev/tap/agentledger
agentledger serve
```
</div>

<div class="al-usecase-step" markdown>
**2. Point OpenClaw at it**

```json
{
  "baseUrl": "http://localhost:8787/v1",
  "api": "openai-completions"
}
```
</div>

<div class="al-usecase-step" markdown>
**3. See everything**

Open `localhost:8787` — every call, every dollar, every agent session. Set budgets. Get alerts. Sleep at night.
</div>

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

### Get early access

<form class="al-form">
  <input type="email" name="email" placeholder="you@company.com" required class="al-input" />
  <button type="submit" class="al-btn">Join the waitlist</button>
</form>

</div>

</div>
