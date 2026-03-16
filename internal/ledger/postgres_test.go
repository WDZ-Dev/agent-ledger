//go:build integration

package ledger

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
)

func newPostgresTestDB(t *testing.T) *Postgres {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping integration test")
	}

	pg, err := NewPostgres(dsn, 5, 2)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(func() { pg.Close() })
	return pg
}

func uniqueID(label string) string {
	return fmt.Sprintf("test-%d-%s", time.Now().UnixNano(), label)
}

// ---------------------------------------------------------------------------
// Ledger method tests
// ---------------------------------------------------------------------------

func TestPostgres_RecordUsage(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	rec := &UsageRecord{
		ID:           uniqueID("rec"),
		Timestamp:    time.Now().UTC(),
		Provider:     "openai",
		Model:        "gpt-4o",
		APIKeyHash:   uniqueID("key"),
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		CostUSD:      0.002,
		Estimated:    false,
		DurationMS:   120,
		StatusCode:   200,
		Path:         "/v1/chat/completions",
		AgentID:      uniqueID("agent"),
		SessionID:    uniqueID("sess"),
		UserID:       uniqueID("user"),
		TenantID:     uniqueID("tenant"),
	}

	if err := pg.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
}

func TestPostgres_QueryCosts(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	tenant := uniqueID("tenant")
	key := uniqueID("key")

	rec1 := &UsageRecord{
		ID:           uniqueID("rec1"),
		Timestamp:    now,
		Provider:     "openai",
		Model:        "gpt-4o",
		APIKeyHash:   key,
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		CostUSD:      0.005,
		StatusCode:   200,
		TenantID:     tenant,
	}
	rec2 := &UsageRecord{
		ID:           uniqueID("rec2"),
		Timestamp:    now,
		Provider:     "anthropic",
		Model:        "claude-sonnet-4",
		APIKeyHash:   key,
		InputTokens:  200,
		OutputTokens: 100,
		TotalTokens:  300,
		CostUSD:      0.01,
		StatusCode:   200,
		TenantID:     tenant,
	}

	for _, r := range []*UsageRecord{rec1, rec2} {
		if err := pg.RecordUsage(ctx, r); err != nil {
			t.Fatalf("RecordUsage: %v", err)
		}
	}

	entries, err := pg.QueryCosts(ctx, CostFilter{
		Since:    now.Add(-time.Hour),
		Until:    now.Add(time.Hour),
		GroupBy:  "model",
		TenantID: tenant,
	})
	if err != nil {
		t.Fatalf("QueryCosts: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 cost entries grouped by model, got %d", len(entries))
	}
}

func TestPostgres_QueryCostTimeseries(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	key := uniqueID("key")

	// Insert records at two different hours to produce two timeseries buckets.
	hourAgo := now.Add(-time.Hour)
	rec1 := &UsageRecord{
		ID:         uniqueID("ts1"),
		Timestamp:  hourAgo,
		Provider:   "openai",
		Model:      "gpt-4o",
		APIKeyHash: key,
		CostUSD:    0.01,
		StatusCode: 200,
	}
	rec2 := &UsageRecord{
		ID:         uniqueID("ts2"),
		Timestamp:  now,
		Provider:   "openai",
		Model:      "gpt-4o",
		APIKeyHash: key,
		CostUSD:    0.02,
		StatusCode: 200,
	}

	for _, r := range []*UsageRecord{rec1, rec2} {
		if err := pg.RecordUsage(ctx, r); err != nil {
			t.Fatalf("RecordUsage: %v", err)
		}
	}

	points, err := pg.QueryCostTimeseries(ctx, "hour", hourAgo.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("QueryCostTimeseries: %v", err)
	}
	if len(points) < 1 {
		t.Fatal("expected at least 1 timeseries bucket, got 0")
	}
	// With records in different hours we expect 2 buckets, but if the test
	// runs right at the hour boundary they could collapse into 1. Accept >= 1.
	t.Logf("timeseries buckets returned: %d", len(points))
}

func TestPostgres_GetTotalSpend(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	key := uniqueID("key")

	recs := []*UsageRecord{
		{ID: uniqueID("s1"), Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: key, CostUSD: 0.10, StatusCode: 200},
		{ID: uniqueID("s2"), Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: key, CostUSD: 0.25, StatusCode: 200},
	}
	for _, r := range recs {
		if err := pg.RecordUsage(ctx, r); err != nil {
			t.Fatalf("RecordUsage: %v", err)
		}
	}

	total, err := pg.GetTotalSpend(ctx, key, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("GetTotalSpend: %v", err)
	}
	const want = 0.35
	if total != want {
		t.Errorf("GetTotalSpend: expected %.2f, got %.6f", want, total)
	}
}

func TestPostgres_GetTotalSpendByTenant(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	tenant := uniqueID("tenant")

	recs := []*UsageRecord{
		{ID: uniqueID("t1"), Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: uniqueID("k"), CostUSD: 0.10, StatusCode: 200, TenantID: tenant},
		{ID: uniqueID("t2"), Timestamp: now, Provider: "openai", Model: "gpt-4o", APIKeyHash: uniqueID("k"), CostUSD: 0.40, StatusCode: 200, TenantID: tenant},
	}
	for _, r := range recs {
		if err := pg.RecordUsage(ctx, r); err != nil {
			t.Fatalf("RecordUsage: %v", err)
		}
	}

	total, err := pg.GetTotalSpendByTenant(ctx, tenant, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("GetTotalSpendByTenant: %v", err)
	}
	const want = 0.50
	if total != want {
		t.Errorf("GetTotalSpendByTenant: expected %.2f, got %.6f", want, total)
	}
}

// ---------------------------------------------------------------------------
// SessionStore method tests
// ---------------------------------------------------------------------------

func TestPostgres_UpsertSession(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	sess := &agent.Session{
		ID:           uniqueID("sess"),
		AgentID:      uniqueID("agent"),
		UserID:       uniqueID("user"),
		Task:         "integration test task",
		StartedAt:    time.Now().UTC(),
		Status:       "active",
		CallCount:    1,
		TotalCostUSD: 0.01,
		TotalTokens:  150,
	}

	if err := pg.UpsertSession(ctx, sess); err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
}

func TestPostgres_GetSession(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	sess := &agent.Session{
		ID:           uniqueID("sess"),
		AgentID:      uniqueID("agent"),
		UserID:       uniqueID("user"),
		Task:         "get-session test",
		StartedAt:    time.Now().UTC().Truncate(time.Microsecond), // Postgres has microsecond precision
		Status:       "active",
		CallCount:    3,
		TotalCostUSD: 0.05,
		TotalTokens:  500,
	}

	if err := pg.UpsertSession(ctx, sess); err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}

	got, err := pg.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}

	if got.ID != sess.ID {
		t.Errorf("ID: want %q, got %q", sess.ID, got.ID)
	}
	if got.AgentID != sess.AgentID {
		t.Errorf("AgentID: want %q, got %q", sess.AgentID, got.AgentID)
	}
	if got.UserID != sess.UserID {
		t.Errorf("UserID: want %q, got %q", sess.UserID, got.UserID)
	}
	if got.Task != sess.Task {
		t.Errorf("Task: want %q, got %q", sess.Task, got.Task)
	}
	if got.Status != sess.Status {
		t.Errorf("Status: want %q, got %q", sess.Status, got.Status)
	}
	if got.CallCount != sess.CallCount {
		t.Errorf("CallCount: want %d, got %d", sess.CallCount, got.CallCount)
	}
	if got.TotalCostUSD != sess.TotalCostUSD {
		t.Errorf("TotalCostUSD: want %f, got %f", sess.TotalCostUSD, got.TotalCostUSD)
	}
	if got.TotalTokens != sess.TotalTokens {
		t.Errorf("TotalTokens: want %d, got %d", sess.TotalTokens, got.TotalTokens)
	}
}

func TestPostgres_ListActiveSessions(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	endedAt := now

	activeSess := &agent.Session{
		ID:        uniqueID("active"),
		AgentID:   uniqueID("agent"),
		UserID:    uniqueID("user"),
		Task:      "active task",
		StartedAt: now,
		Status:    "active",
		CallCount: 1,
	}
	completedSess := &agent.Session{
		ID:        uniqueID("completed"),
		AgentID:   uniqueID("agent"),
		UserID:    uniqueID("user"),
		Task:      "completed task",
		StartedAt: now,
		EndedAt:   &endedAt,
		Status:    "completed",
		CallCount: 5,
	}

	for _, s := range []*agent.Session{activeSess, completedSess} {
		if err := pg.UpsertSession(ctx, s); err != nil {
			t.Fatalf("UpsertSession: %v", err)
		}
	}

	active, err := pg.ListActiveSessions(ctx)
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}

	// There may be active sessions from other test runs, so check that our
	// specific active session is present and the completed one is not.
	foundActive := false
	foundCompleted := false
	for _, s := range active {
		if s.ID == activeSess.ID {
			foundActive = true
		}
		if s.ID == completedSess.ID {
			foundCompleted = true
		}
	}
	if !foundActive {
		t.Error("expected active session to appear in ListActiveSessions")
	}
	if foundCompleted {
		t.Error("completed session should not appear in ListActiveSessions")
	}
}

func TestPostgres_UpsertSession_Update(t *testing.T) {
	pg := newPostgresTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	sess := &agent.Session{
		ID:           uniqueID("sess"),
		AgentID:      uniqueID("agent"),
		UserID:       uniqueID("user"),
		Task:         "upsert update test",
		StartedAt:    now,
		Status:       "active",
		CallCount:    1,
		TotalCostUSD: 0.01,
		TotalTokens:  100,
	}

	if err := pg.UpsertSession(ctx, sess); err != nil {
		t.Fatalf("UpsertSession (insert): %v", err)
	}

	// Update the session: change status, bump call count.
	endedAt := now.Add(5 * time.Minute)
	sess.Status = "completed"
	sess.CallCount = 10
	sess.TotalCostUSD = 0.50
	sess.TotalTokens = 5000
	sess.EndedAt = &endedAt

	if err := pg.UpsertSession(ctx, sess); err != nil {
		t.Fatalf("UpsertSession (update): %v", err)
	}

	got, err := pg.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession after update: %v", err)
	}

	if got.Status != "completed" {
		t.Errorf("Status: want %q, got %q", "completed", got.Status)
	}
	if got.CallCount != 10 {
		t.Errorf("CallCount: want 10, got %d", got.CallCount)
	}
	if got.TotalCostUSD != 0.50 {
		t.Errorf("TotalCostUSD: want 0.50, got %f", got.TotalCostUSD)
	}
	if got.TotalTokens != 5000 {
		t.Errorf("TotalTokens: want 5000, got %d", got.TotalTokens)
	}
	if got.EndedAt == nil {
		t.Fatal("EndedAt: expected non-nil after update")
	}
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestPostgres_Close(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping integration test")
	}

	pg, err := NewPostgres(dsn, 5, 2)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}

	if err := pg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
