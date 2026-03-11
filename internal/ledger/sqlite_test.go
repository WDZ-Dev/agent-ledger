package ledger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *SQLite {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")
	db, err := NewSQLite(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRecordAndQuery(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now()

	records := []*UsageRecord{
		{
			ID: "01", Timestamp: now, Provider: "openai", Model: "gpt-4o-mini",
			APIKeyHash: "abc123", InputTokens: 100, OutputTokens: 50,
			TotalTokens: 150, CostUSD: 0.001, StatusCode: 200,
		},
		{
			ID: "02", Timestamp: now, Provider: "openai", Model: "gpt-4o",
			APIKeyHash: "abc123", InputTokens: 200, OutputTokens: 100,
			TotalTokens: 300, CostUSD: 0.005, StatusCode: 200,
		},
		{
			ID: "03", Timestamp: now, Provider: "anthropic", Model: "claude-sonnet-4",
			APIKeyHash: "def456", InputTokens: 500, OutputTokens: 200,
			TotalTokens: 700, CostUSD: 0.01, StatusCode: 200,
		},
	}

	for _, r := range records {
		if err := db.RecordUsage(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	// Query all by model.
	entries, err := db.QueryCosts(ctx, CostFilter{
		Since:   now.Add(-time.Hour),
		Until:   now.Add(time.Hour),
		GroupBy: "model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Highest cost first.
	if entries[0].Model != "claude-sonnet-4" {
		t.Errorf("expected claude-sonnet-4 first (highest cost), got %q", entries[0].Model)
	}

	// Query by provider.
	entries, err = db.QueryCosts(ctx, CostFilter{
		Since:   now.Add(-time.Hour),
		Until:   now.Add(time.Hour),
		GroupBy: "provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 provider groups, got %d", len(entries))
	}
}

func TestGetTotalSpend(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now()

	records := []*UsageRecord{
		{ID: "01", Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: "key1", CostUSD: 0.10},
		{ID: "02", Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: "key1", CostUSD: 0.25},
		{ID: "03", Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: "key2", CostUSD: 0.50},
	}
	for _, r := range records {
		if err := db.RecordUsage(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	total, err := db.GetTotalSpend(ctx, "key1", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if total != 0.35 {
		t.Errorf("expected 0.35, got %f", total)
	}
}

func TestEmptyQuery(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now()

	entries, err := db.QueryCosts(ctx, CostFilter{
		Since: now.Add(-time.Hour), Until: now.Add(time.Hour), GroupBy: "model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestMigrationsCreateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep")
	dsn := filepath.Join(dir, "test.db")

	db, err := NewSQLite(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}
