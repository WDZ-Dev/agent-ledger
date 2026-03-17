package ledger

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // register sqlite driver

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
)

//go:embed migrations/sqlite/*.sql
var embedSQLiteMigrations embed.FS

// SQLite implements the Ledger interface using SQLite (CGO-free via modernc.org).
type SQLite struct {
	db *sql.DB
}

// NewSQLite opens or creates a SQLite database at the given path and runs
// any pending migrations.
func NewSQLite(dsn string) (*SQLite, error) {
	// Ensure the directory exists.
	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("creating data dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dsn+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// SQLite handles concurrency best with a single writer.
	db.SetMaxOpenConns(1)

	if err := runSQLiteMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLite{db: db}, nil
}

func runSQLiteMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedSQLiteMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations/sqlite"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

func (s *SQLite) RecordUsage(ctx context.Context, record *UsageRecord) error {
	const q = `INSERT INTO usage_records (
		id, timestamp, provider, model, api_key_hash,
		input_tokens, output_tokens, total_tokens, cost_usd, estimated,
		duration_ms, status_code, path, agent_id, session_id, user_id, tenant_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, q,
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

func (s *SQLite) QueryCosts(ctx context.Context, filter CostFilter) ([]CostEntry, error) {
	groupCol := "model"
	switch filter.GroupBy {
	case "provider":
		groupCol = "provider"
	case "key":
		groupCol = "api_key_hash"
	case "agent":
		groupCol = "agent_id"
	case "session":
		groupCol = "session_id"
	}

	where := "timestamp >= ? AND timestamp <= ?" //nolint:goconst
	args := []any{filter.Since.UTC(), filter.Until.UTC()}
	if filter.TenantID != "" {
		where += " AND tenant_id = ?" //nolint:goconst
		args = append(args, filter.TenantID)
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

	rows, err := s.db.QueryContext(ctx, q, args...)
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

func (s *SQLite) QueryCostTimeseries(ctx context.Context, interval string, since, until time.Time, tenantID string) ([]TimeseriesPoint, error) {
	// Go's time.Time stores as "2006-01-02 15:04:05.999999 +0000 UTC" in SQLite,
	// but strftime only parses ISO8601. Use substr to extract the datetime portion.
	bucket := "strftime('%Y-%m-%d %H:00:00', substr(timestamp, 1, 19))"
	if interval == "day" {
		bucket = "strftime('%Y-%m-%d 00:00:00', substr(timestamp, 1, 19))"
	}

	where := "timestamp >= ? AND timestamp <= ?" //nolint:goconst
	args := []any{since.UTC(), until.UTC()}
	if tenantID != "" {
		where += " AND tenant_id = ?" //nolint:goconst
		args = append(args, tenantID)
	}

	q := fmt.Sprintf(`SELECT
		%s as bucket,
		COALESCE(SUM(cost_usd), 0),
		COUNT(*)
	FROM usage_records
	WHERE %s
	GROUP BY bucket
	ORDER BY bucket ASC`, bucket, where)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying cost timeseries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var points []TimeseriesPoint
	for rows.Next() {
		var p TimeseriesPoint
		var ts string
		if err := rows.Scan(&ts, &p.CostUSD, &p.Requests); err != nil {
			return nil, fmt.Errorf("scanning timeseries point: %w", err)
		}
		p.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *SQLite) GetTotalSpend(ctx context.Context, apiKeyHash string, since, until time.Time) (float64, error) {
	const q = `SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records
		WHERE api_key_hash = ? AND timestamp >= ? AND timestamp <= ?`

	var total float64
	if err := s.db.QueryRowContext(ctx, q, apiKeyHash, since.UTC(), until.UTC()).Scan(&total); err != nil {
		return 0, fmt.Errorf("querying total spend: %w", err)
	}
	return total, nil
}

func (s *SQLite) GetTotalSpendByTenant(ctx context.Context, tenantID string, since, until time.Time) (float64, error) {
	const q = `SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?`

	var total float64
	if err := s.db.QueryRowContext(ctx, q, tenantID, since.UTC(), until.UTC()).Scan(&total); err != nil {
		return 0, fmt.Errorf("querying tenant spend: %w", err)
	}
	return total, nil
}

func (s *SQLite) QueryRecentExpensive(ctx context.Context, since, until time.Time, tenantID string, limit int) ([]ExpensiveRequest, error) {
	where := "timestamp >= ? AND timestamp <= ?" //nolint:goconst //nolint:goconst
	args := []any{since.UTC(), until.UTC()}
	if tenantID != "" {
		where += " AND tenant_id = ?" //nolint:goconst //nolint:goconst
		args = append(args, tenantID)
	}
	args = append(args, limit)

	q := fmt.Sprintf(`SELECT timestamp, provider, model, agent_id,
		input_tokens, output_tokens, cost_usd, duration_ms
	FROM usage_records WHERE %s
	ORDER BY cost_usd DESC LIMIT ?`, where)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying expensive requests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []ExpensiveRequest
	for rows.Next() {
		var r ExpensiveRequest
		if err := rows.Scan(&r.Timestamp, &r.Provider, &r.Model, &r.AgentID,
			&r.InputTokens, &r.OutputTokens, &r.CostUSD, &r.DurationMS); err != nil {
			return nil, fmt.Errorf("scanning expensive request: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *SQLite) QueryErrorStats(ctx context.Context, since, until time.Time, tenantID string) (*ErrorStats, error) {
	where := "timestamp >= ? AND timestamp <= ?" //nolint:goconst //nolint:goconst
	args := []any{since.UTC(), until.UTC()}
	if tenantID != "" {
		where += " AND tenant_id = ?" //nolint:goconst //nolint:goconst
		args = append(args, tenantID)
	}

	q := fmt.Sprintf(`SELECT
		COUNT(*),
		SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END),
		SUM(CASE WHEN status_code = 429 THEN 1 ELSE 0 END),
		SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END),
		COALESCE(AVG(duration_ms), 0),
		COALESCE(AVG(cost_usd), 0)
	FROM usage_records WHERE %s`, where)

	var stats ErrorStats
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(
		&stats.TotalRequests, &stats.ErrorRequests,
		&stats.Count429, &stats.Count5xx,
		&stats.AvgDurationMS, &stats.AvgCostPerReq,
	); err != nil {
		return nil, fmt.Errorf("querying error stats: %w", err)
	}
	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(stats.ErrorRequests) / float64(stats.TotalRequests)
	}
	return &stats, nil
}

// UpsertSession inserts or updates an agent session record.
func (s *SQLite) UpsertSession(ctx context.Context, sess *agent.Session) error {
	const q = `INSERT INTO agent_sessions (
		id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		ended_at = excluded.ended_at,
		status = excluded.status,
		call_count = excluded.call_count,
		total_cost_usd = excluded.total_cost_usd,
		total_tokens = excluded.total_tokens`

	var endedAt *time.Time
	if sess.EndedAt != nil {
		endedAt = sess.EndedAt
	}

	_, err := s.db.ExecContext(ctx, q,
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
func (s *SQLite) GetSession(ctx context.Context, id string) (*agent.Session, error) {
	const q = `SELECT id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	FROM agent_sessions WHERE id = ?`

	var sess agent.Session
	var endedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, q, id).Scan(
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
func (s *SQLite) ListActiveSessions(ctx context.Context) ([]agent.Session, error) {
	const q = `SELECT id, agent_id, user_id, task, started_at, ended_at, status,
		call_count, total_cost_usd, total_tokens
	FROM agent_sessions WHERE status = 'active'`

	rows, err := s.db.QueryContext(ctx, q)
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
func (s *SQLite) DB() *sql.DB {
	return s.db
}

func (s *SQLite) Close() error {
	return s.db.Close()
}
