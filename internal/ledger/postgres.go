package ledger

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/lib/pq" // register postgres driver
	"github.com/pressly/goose/v3"

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
)

//go:embed migrations/postgres/*.sql
var embedPostgresMigrations embed.FS

// Postgres implements the Ledger interface using PostgreSQL.
type Postgres struct {
	db *sql.DB
}

// NewPostgres connects to a PostgreSQL database and runs migrations.
func NewPostgres(dsn string, maxOpen, maxIdle int) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres: %w", err)
	}

	if maxOpen > 0 {
		db.SetMaxOpenConns(maxOpen)
	}
	if maxIdle > 0 {
		db.SetMaxIdleConns(maxIdle)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	if err := runPostgresMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Postgres{db: db}, nil
}

func runPostgresMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedPostgresMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations/postgres"); err != nil {
		return fmt.Errorf("running postgres migrations: %w", err)
	}
	return nil
}

func (p *Postgres) RecordUsage(ctx context.Context, record *UsageRecord) error {
	const q = `INSERT INTO usage_records (
		id, timestamp, provider, model, api_key_hash,
		input_tokens, output_tokens, total_tokens, cost_usd, estimated,
		duration_ms, status_code, path, agent_id, session_id, user_id, tenant_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := p.db.ExecContext(ctx, q,
		record.ID, record.Timestamp.UTC(), record.Provider, record.Model, record.APIKeyHash,
		record.InputTokens, record.OutputTokens, record.TotalTokens, record.CostUSD, record.Estimated,
		record.DurationMS, record.StatusCode, record.Path, record.AgentID, record.SessionID, record.UserID,
		record.TenantID,
	)
	if err != nil {
		return fmt.Errorf("inserting usage record: %w", err)
	}
	return nil
}

func (p *Postgres) QueryCosts(ctx context.Context, filter CostFilter) ([]CostEntry, error) {
	groupCol := "model"
	switch filter.GroupBy {
	case "provider": //nolint:goconst
		groupCol = "provider"
	case "key":
		groupCol = "api_key_hash"
	case "agent":
		groupCol = "agent_id"
	case "session":
		groupCol = "session_id"
	}

	where := "timestamp >= $1 AND timestamp <= $2"
	args := []any{filter.Since.UTC(), filter.Until.UTC()}
	if filter.TenantID != "" {
		args = append(args, filter.TenantID)
		where += fmt.Sprintf(" AND tenant_id = $%d", len(args))
	}

	q := fmt.Sprintf(`SELECT
		provider, model, api_key_hash, agent_id, session_id,
		COUNT(*) as requests,
		COALESCE(SUM(input_tokens), 0),
		COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(cost_usd), 0)
	FROM usage_records
	WHERE %s
	GROUP BY %s
	ORDER BY SUM(cost_usd) DESC`, where, groupCol)

	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying costs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []CostEntry
	for rows.Next() {
		var e CostEntry
		if err := rows.Scan(&e.Provider, &e.Model, &e.APIKeyHash, &e.AgentID, &e.SessionID,
			&e.Requests, &e.InputTokens, &e.OutputTokens, &e.TotalCostUSD); err != nil {
			return nil, fmt.Errorf("scanning cost entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (p *Postgres) QueryCostTimeseries(ctx context.Context, interval string, since, until time.Time) ([]TimeseriesPoint, error) {
	bucket := "date_trunc('hour', timestamp)"
	if interval == "day" {
		bucket = "date_trunc('day', timestamp)"
	}

	q := fmt.Sprintf(`SELECT
		%s as bucket,
		COALESCE(SUM(cost_usd), 0),
		COUNT(*)
	FROM usage_records
	WHERE timestamp >= $1 AND timestamp <= $2
	GROUP BY bucket
	ORDER BY bucket ASC`, bucket)

	rows, err := p.db.QueryContext(ctx, q, since.UTC(), until.UTC())
	if err != nil {
		return nil, fmt.Errorf("querying cost timeseries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var points []TimeseriesPoint
	for rows.Next() {
		var pt TimeseriesPoint
		if err := rows.Scan(&pt.Timestamp, &pt.CostUSD, &pt.Requests); err != nil {
			return nil, fmt.Errorf("scanning timeseries point: %w", err)
		}
		points = append(points, pt)
	}
	return points, rows.Err()
}

func (p *Postgres) GetTotalSpend(ctx context.Context, apiKeyHash string, since, until time.Time) (float64, error) {
	const q = `SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records
		WHERE api_key_hash = $1 AND timestamp >= $2 AND timestamp <= $3`

	var total float64
	if err := p.db.QueryRowContext(ctx, q, apiKeyHash, since.UTC(), until.UTC()).Scan(&total); err != nil {
		return 0, fmt.Errorf("querying total spend: %w", err)
	}
	return total, nil
}

func (p *Postgres) GetTotalSpendByTenant(ctx context.Context, tenantID string, since, until time.Time) (float64, error) {
	const q = `SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3`

	var total float64
	if err := p.db.QueryRowContext(ctx, q, tenantID, since.UTC(), until.UTC()).Scan(&total); err != nil {
		return 0, fmt.Errorf("querying tenant spend: %w", err)
	}
	return total, nil
}

// UpsertSession inserts or updates an agent session record.
func (p *Postgres) UpsertSession(ctx context.Context, sess *agent.Session) error {
	const q = `INSERT INTO agent_sessions (
		id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT(id) DO UPDATE SET
		ended_at = EXCLUDED.ended_at,
		status = EXCLUDED.status,
		call_count = EXCLUDED.call_count,
		total_cost_usd = EXCLUDED.total_cost_usd,
		total_tokens = EXCLUDED.total_tokens`

	var endedAt *time.Time
	if sess.EndedAt != nil {
		endedAt = sess.EndedAt
	}

	_, err := p.db.ExecContext(ctx, q,
		sess.ID, sess.AgentID, sess.UserID, sess.Task,
		sess.StartedAt.UTC(), endedAt, sess.Status,
		sess.CallCount, sess.TotalCostUSD, sess.TotalTokens,
	)
	if err != nil {
		return fmt.Errorf("upserting session: %w", err)
	}
	return nil
}

// GetSession retrieves a single agent session by ID.
func (p *Postgres) GetSession(ctx context.Context, id string) (*agent.Session, error) {
	const q = `SELECT id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	FROM agent_sessions WHERE id = $1`

	var sess agent.Session
	var endedAt sql.NullTime
	err := p.db.QueryRowContext(ctx, q, id).Scan(
		&sess.ID, &sess.AgentID, &sess.UserID, &sess.Task,
		&sess.StartedAt, &endedAt, &sess.Status,
		&sess.CallCount, &sess.TotalCostUSD, &sess.TotalTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}
	if endedAt.Valid {
		sess.EndedAt = &endedAt.Time
	}
	return &sess, nil
}

// ListActiveSessions returns all sessions with status "active".
func (p *Postgres) ListActiveSessions(ctx context.Context) ([]agent.Session, error) {
	const q = `SELECT id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	FROM agent_sessions WHERE status = 'active'`

	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing active sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sessions []agent.Session
	for rows.Next() {
		var sess agent.Session
		var endedAt sql.NullTime
		if err := rows.Scan(
			&sess.ID, &sess.AgentID, &sess.UserID, &sess.Task,
			&sess.StartedAt, &endedAt, &sess.Status,
			&sess.CallCount, &sess.TotalCostUSD, &sess.TotalTokens,
		); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// DB returns the underlying database connection for use by other packages
// (e.g., admin config store).
func (p *Postgres) DB() *sql.DB {
	return p.db
}

func (p *Postgres) Close() error {
	return p.db.Close()
}
