-- +goose Up
CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    api_key_hash TEXT NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0.0,
    estimated BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL DEFAULT 0,
    path TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_usage_records_timestamp ON usage_records(timestamp);
CREATE INDEX idx_usage_records_api_key_hash ON usage_records(api_key_hash);
CREATE INDEX idx_usage_records_model ON usage_records(model);
CREATE INDEX idx_usage_records_provider ON usage_records(provider);

-- +goose Down
DROP TABLE IF EXISTS usage_records;
