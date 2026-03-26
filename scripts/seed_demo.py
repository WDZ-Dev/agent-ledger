#!/usr/bin/env python3
"""Seed the AgentLedger SQLite database with realistic demo data for dashboard screenshots/videos."""

import sqlite3
import random
import uuid
import os
import sys
from datetime import datetime, timedelta, timezone

DB_PATH = os.environ.get("AGENTLEDGER_DB", "data/agentledger.db")

# --- Models with realistic pricing (USD per million tokens) ---
MODELS = {
    # (provider, model): (input_per_mtok, output_per_mtok, weight)
    ("openai", "gpt-4.1"): (2.00, 8.00, 25),
    ("openai", "gpt-4.1-mini"): (0.40, 1.60, 35),
    ("openai", "gpt-4o"): (2.50, 10.00, 15),
    ("openai", "o3-mini"): (1.10, 4.40, 8),
    ("openai", "gpt-3.5-turbo"): (0.50, 1.50, 5),
    ("anthropic", "claude-sonnet-4"): (3.00, 15.00, 30),
    ("anthropic", "claude-haiku-4.5"): (1.00, 5.00, 20),
    ("anthropic", "claude-opus-4"): (15.00, 75.00, 5),
    ("groq", "llama-3.3-70b-versatile"): (0.59, 0.79, 15),
    ("groq", "llama-3.1-8b-instant"): (0.05, 0.08, 10),
    ("mistral", "mistral-large-latest"): (2.00, 6.00, 8),
    ("deepseek", "deepseek-chat"): (0.14, 0.28, 12),
    ("deepseek", "deepseek-reasoner"): (0.55, 2.19, 6),
    ("gemini", "gemini-2.5-pro"): (1.25, 10.00, 10),
    ("gemini", "gemini-2.5-flash"): (0.15, 0.60, 12),
    ("cohere", "command-r-plus"): (2.50, 10.00, 4),
    ("xai", "grok-3"): (3.00, 15.00, 3),
    ("perplexity", "sonar-pro"): (3.00, 15.00, 3),
    ("together", "meta-llama/Llama-3.3-70B-Instruct-Turbo"): (0.88, 0.88, 5),
}

# --- Agents ---
AGENTS = [
    {"id": "code-reviewer", "tasks": ["Review PR #142", "Review PR #187", "Review PR #201", "Review PR #256", "Review PR #312"]},
    {"id": "bug-triager", "tasks": ["Triage issue #891", "Triage issue #904", "Triage issue #923", "Classify bug reports batch 14"]},
    {"id": "doc-writer", "tasks": ["Generate API docs", "Update README", "Write migration guide v3", "Document new endpoints"]},
    {"id": "test-generator", "tasks": ["Generate tests for auth module", "Generate tests for billing", "Cover edge cases for parser"]},
    {"id": "data-analyst", "tasks": ["Analyze Q1 metrics", "Weekly report generation", "Revenue forecast model", "Churn analysis"]},
    {"id": "chat-assistant", "tasks": ["Customer support session", "Onboarding flow", "FAQ responses", "Help desk ticket #4421"]},
    {"id": "deploy-bot", "tasks": ["Pre-deploy checks", "Rollback analysis", "Canary validation", "Infra audit"]},
    {"id": "security-scanner", "tasks": ["Dependency audit", "SAST scan review", "CVE assessment", "Secrets detection"]},
]

# --- API key hashes (fake but realistic-looking) ---
API_KEYS = [
    "sk-proj-abc...7f2a",
    "sk-proj-dev...3e1b",
    "sk-proj-prod...9d4c",
    "sk-ant-team1...2a8f",
    "sk-ant-team2...6c3d",
    "sk-proj-staging...1b7e",
]

USERS = ["danial@example.com", "zubair@example.com", "wasay@example.com", "bot@ci.internal"]

PATHS = [
    "/v1/chat/completions",
    "/v1/messages",
    "/groq/v1/chat/completions",
    "/mistral/v1/chat/completions",
    "/deepseek/v1/chat/completions",
    "/gemini/v1beta/models/gemini-2.5-pro:generateContent",
    "/cohere/v2/chat",
    "/xai/v1/chat/completions",
    "/perplexity/v1/chat/completions",
    "/together/v1/chat/completions",
]


def get_path_for_provider(provider):
    mapping = {
        "openai": "/v1/chat/completions",
        "anthropic": "/v1/messages",
        "groq": "/groq/v1/chat/completions",
        "mistral": "/mistral/v1/chat/completions",
        "deepseek": "/deepseek/v1/chat/completions",
        "gemini": "/gemini/v1beta/models/gemini-2.5-pro:generateContent",
        "cohere": "/cohere/v2/chat",
        "xai": "/xai/v1/chat/completions",
        "perplexity": "/perplexity/v1/chat/completions",
        "together": "/together/v1/chat/completions",
    }
    return mapping.get(provider, "/v1/chat/completions")


def generate_records(num_records=3000, days_back=7):
    """Generate realistic usage records."""
    now = datetime.now(timezone.utc)
    records = []

    # Build weighted model list
    model_choices = []
    model_weights = []
    for key, (inp, out, weight) in MODELS.items():
        model_choices.append(key)
        model_weights.append(weight)

    # Generate agent sessions first
    sessions = []
    for _ in range(80):
        agent = random.choice(AGENTS)
        session_start = now - timedelta(
            hours=random.uniform(0, days_back * 24)
        )
        session_id = f"sess_{uuid.uuid4().hex[:12]}"
        task = random.choice(agent["tasks"])
        user = random.choice(USERS)
        call_count = random.randint(5, 60)
        sessions.append({
            "id": session_id,
            "agent_id": agent["id"],
            "user_id": user,
            "task": task,
            "started_at": session_start,
            "call_count": call_count,
        })

    for i in range(num_records):
        # 40% of records in last 12 hours, rest spread across days_back
        if random.random() < 0.4:
            hours_ago = random.uniform(0, 12)
        else:
            hours_ago = random.uniform(0, days_back * 24)
        timestamp = now - timedelta(hours=hours_ago)

        # Add daily patterns (more activity during work hours US Pacific = UTC-7)
        pacific_hour = (timestamp.hour - 7) % 24
        if 9 <= pacific_hour <= 17:
            pass  # already in work hours
        elif random.random() > 0.3:
            # Shift to work hours (Pacific 9-17 = UTC 16-24/0)
            target_pacific = random.randint(9, 17)
            target_utc = (target_pacific + 7) % 24
            timestamp = timestamp.replace(hour=target_utc)

        # Pick model
        (provider, model) = random.choices(model_choices, weights=model_weights, k=1)[0]
        inp_price, out_price, _ = MODELS[(provider, model)]

        # Token counts (vary by model type)
        if "opus" in model or "gpt-4.1" == model or "gpt-4o" == model or "grok-3" == model:
            input_tokens = random.randint(2000, 30000)
            output_tokens = random.randint(500, 8000)
        elif "haiku" in model or "mini" in model or "flash" in model or "instant" in model or "3.5" in model:
            input_tokens = random.randint(200, 5000)
            output_tokens = random.randint(100, 2000)
        else:
            input_tokens = random.randint(500, 15000)
            output_tokens = random.randint(200, 5000)

        total_tokens = input_tokens + output_tokens
        cost = (input_tokens * inp_price + output_tokens * out_price) / 1_000_000

        # Status code (mostly 200, some errors)
        r = random.random()
        if r < 0.92:
            status = 200
        elif r < 0.96:
            status = 429
        elif r < 0.98:
            status = 500
        else:
            status = 503

        # Duration (ms)
        if "instant" in model or "flash" in model or "haiku" in model:
            duration = random.randint(80, 800)
        elif "opus" in model or "reasoner" in model:
            duration = random.randint(2000, 15000)
        else:
            duration = random.randint(300, 5000)

        # Assign to a session or standalone
        agent_id = ""
        session_id = ""
        user_id = ""
        if random.random() < 0.75:
            session = random.choice(sessions)
            agent_id = session["agent_id"]
            session_id = session["id"]
            user_id = session["user_id"]

        api_key = random.choice(API_KEYS)
        path = get_path_for_provider(provider)

        records.append((
            str(uuid.uuid4()),
            timestamp.strftime("%Y-%m-%d %H:%M:%S +0000 UTC"),
            provider,
            model,
            api_key,
            input_tokens,
            output_tokens,
            total_tokens,
            round(cost, 6),
            0,  # estimated
            duration,
            status,
            path,
            agent_id,
            session_id,
            user_id,
            "",  # tenant_id
        ))

    return records, sessions


def main():
    os.makedirs(os.path.dirname(DB_PATH) or ".", exist_ok=True)

    print(f"Seeding {DB_PATH}...")

    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()

    # Check if table exists (proxy must have run at least once to create schema)
    c.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='usage_records'")
    if not c.fetchone():
        print("ERROR: usage_records table not found. Run 'agentledger serve' once first to create the schema.")
        sys.exit(1)

    # Clear existing data
    c.execute("DELETE FROM usage_records")
    c.execute("DELETE FROM agent_sessions")

    records, sessions = generate_records(num_records=3500, days_back=7)

    # Insert usage records
    c.executemany("""
        INSERT INTO usage_records
        (id, timestamp, provider, model, api_key_hash, input_tokens, output_tokens,
         total_tokens, cost_usd, estimated, duration_ms, status_code, path,
         agent_id, session_id, user_id, tenant_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, records)

    # Insert agent sessions
    now = datetime.now(timezone.utc)
    session_rows = []
    for s in sessions:
        # Calculate session totals from records
        session_cost = sum(r[8] for r in records if r[14] == s["id"])
        session_tokens = sum(r[7] for r in records if r[14] == s["id"])
        session_calls = sum(1 for r in records if r[14] == s["id"])

        ended = s["started_at"] + timedelta(minutes=random.randint(2, 45))
        status = "completed" if ended < now - timedelta(hours=1) else random.choice(["active", "completed"])

        session_rows.append((
            s["id"],
            s["agent_id"],
            s["user_id"],
            s["task"],
            s["started_at"].strftime("%Y-%m-%d %H:%M:%S +0000 UTC"),
            ended.strftime("%Y-%m-%d %H:%M:%S +0000 UTC") if status == "completed" else None,
            status,
            session_calls,
            round(session_cost, 6),
            session_tokens,
        ))

    c.executemany("""
        INSERT INTO agent_sessions
        (id, agent_id, user_id, task, started_at, ended_at, status, call_count, total_cost_usd, total_tokens)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, session_rows)

    conn.commit()

    # Print summary
    total_cost = sum(r[8] for r in records)
    providers = set(r[2] for r in records)
    models = set(r[3] for r in records)
    agents = set(r[13] for r in records if r[13])

    print(f"  {len(records):,} usage records inserted")
    print(f"  {len(session_rows)} agent sessions inserted")
    print(f"  {len(providers)} providers, {len(models)} models")
    print(f"  {len(agents)} unique agents")
    print(f"  ${total_cost:,.2f} total cost")
    print(f"\nDone! Start the proxy and open http://localhost:8787/")

    conn.close()


if __name__ == "__main__":
    main()
