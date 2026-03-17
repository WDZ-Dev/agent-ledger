package admin_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/WDZ-Dev/agent-ledger/internal/admin"
	"github.com/WDZ-Dev/agent-ledger/internal/budget"
)

// setupTestDB creates an in-memory SQLite database with the admin_config table.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE admin_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestStore_GetSetDelete(t *testing.T) {
	db := setupTestDB(t)
	s := admin.NewStore(db)
	ctx := context.Background()

	// Get non-existent key returns empty.
	val, err := s.Get(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("expected empty, got %q", val)
	}

	// Set a value.
	if setErr := s.Set(ctx, "foo", "bar"); setErr != nil {
		t.Fatal(setErr)
	}
	val, err = s.Get(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if val != "bar" {
		t.Fatalf("expected bar, got %q", val)
	}

	// Upsert.
	if upsertErr := s.Set(ctx, "foo", "baz"); upsertErr != nil {
		t.Fatal(upsertErr)
	}
	val, err = s.Get(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if val != "baz" {
		t.Fatalf("expected baz, got %q", val)
	}

	// Delete.
	if delErr := s.Delete(ctx, "foo"); delErr != nil {
		t.Fatal(delErr)
	}
	val, err = s.Get(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("expected empty after delete, got %q", val)
	}
}

func TestStore_JSON(t *testing.T) {
	db := setupTestDB(t)
	s := admin.NewStore(db)
	ctx := context.Background()

	rules := []budget.Rule{
		{APIKeyPattern: "sk-test-*", DailyLimitUSD: 10.0, Action: "block"},
	}

	if err := s.SetJSON(ctx, "budget_rules", rules); err != nil {
		t.Fatal(err)
	}

	var got []budget.Rule
	if err := s.GetJSON(ctx, "budget_rules", &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].APIKeyPattern != "sk-test-*" {
		t.Fatalf("unexpected rules: %+v", got)
	}
}

func TestStore_ListAll(t *testing.T) {
	db := setupTestDB(t)
	s := admin.NewStore(db)
	ctx := context.Background()

	_ = s.Set(ctx, "a", "1")
	_ = s.Set(ctx, "b", "2")

	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all["a"] != "1" || all["b"] != "2" {
		t.Fatalf("unexpected entries: %+v", all)
	}
}

func TestHandler_RequiresAuth(t *testing.T) {
	db := setupTestDB(t)
	store := admin.NewStore(db)
	handler := admin.NewHandler(store, nil, nil, "secret-token", nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// No auth header.
	req := httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Wrong token.
	req = httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Correct token.
	req = httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_CRUDRules(t *testing.T) {
	db := setupTestDB(t)
	store := admin.NewStore(db)
	handler := admin.NewHandler(store, nil, nil, "token", nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	auth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer token")
	}

	// List — initially empty.
	req := httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	auth(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}

	// Create a rule.
	rule := budget.Rule{APIKeyPattern: "sk-prod-*", DailyLimitUSD: 50.0, Action: "block"}
	body, _ := json.Marshal(rule)
	req = httptest.NewRequest("POST", "/api/admin/budgets/rules", bytes.NewReader(body))
	auth(req)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	// List — should have 1 rule.
	req = httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	auth(req)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var rules []budget.Rule
	_ = json.NewDecoder(rec.Body).Decode(&rules)
	if len(rules) != 1 || rules[0].APIKeyPattern != "sk-prod-*" {
		t.Fatalf("expected 1 rule, got %+v", rules)
	}

	// Delete.
	req = httptest.NewRequest("DELETE", "/api/admin/budgets/rules?pattern=sk-prod-*", nil)
	auth(req)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rec.Code)
	}

	// List — should be empty again.
	req = httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	auth(req)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	rules = nil
	_ = json.NewDecoder(rec.Body).Decode(&rules)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %+v", rules)
	}
}

func TestHandler_DeleteNonExistent(t *testing.T) {
	db := setupTestDB(t)
	store := admin.NewStore(db)
	handler := admin.NewHandler(store, nil, nil, "token", nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("DELETE", "/api/admin/budgets/rules?pattern=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_NoToken(t *testing.T) {
	db := setupTestDB(t)
	store := admin.NewStore(db)
	handler := admin.NewHandler(store, nil, nil, "", nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/admin/budgets/rules", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when no token configured, got %d", rec.Code)
	}
}
