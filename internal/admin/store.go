package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Store provides runtime configuration persistence via the admin_config table.
type Store struct {
	db *sql.DB
}

// NewStore creates an admin config store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Get retrieves a config value by key.
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM admin_config WHERE key = $1", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting admin config %s: %w", key, err)
	}
	return value, nil
}

// Set stores a config value.
func (s *Store) Set(ctx context.Context, key, value string) error {
	const q = `INSERT INTO admin_config (key, value, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
	_, err := s.db.ExecContext(ctx, q, key, value, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("setting admin config %s: %w", key, err)
	}
	return nil
}

// Delete removes a config value.
func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM admin_config WHERE key = $1", key)
	if err != nil {
		return fmt.Errorf("deleting admin config %s: %w", key, err)
	}
	return nil
}

// GetJSON retrieves and unmarshals a JSON config value.
func (s *Store) GetJSON(ctx context.Context, key string, dest any) error {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), dest)
}

// SetJSON marshals and stores a config value as JSON.
func (s *Store) SetJSON(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshalling admin config %s: %w", key, err)
	}
	return s.Set(ctx, key, string(data))
}

// ListAll returns all config entries.
func (s *Store) ListAll(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key, value FROM admin_config")
	if err != nil {
		return nil, fmt.Errorf("listing admin config: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scanning admin config: %w", err)
		}
		result[k] = v
	}
	return result, rows.Err()
}
