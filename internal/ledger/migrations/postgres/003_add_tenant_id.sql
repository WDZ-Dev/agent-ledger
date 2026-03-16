-- +goose Up
ALTER TABLE usage_records ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE agent_sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_usage_records_tenant_id ON usage_records(tenant_id);
CREATE INDEX idx_sessions_tenant_id ON agent_sessions(tenant_id);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_tenant_id;
DROP INDEX IF EXISTS idx_usage_records_tenant_id;
ALTER TABLE agent_sessions DROP COLUMN tenant_id;
ALTER TABLE usage_records DROP COLUMN tenant_id;
