package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

type stubLedger struct {
	costs      []ledger.CostEntry
	timeseries []ledger.TimeseriesPoint
}

func (s *stubLedger) RecordUsage(_ context.Context, _ *ledger.UsageRecord) error { return nil }
func (s *stubLedger) QueryCosts(_ context.Context, _ ledger.CostFilter) ([]ledger.CostEntry, error) {
	return s.costs, nil
}
func (s *stubLedger) GetTotalSpend(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (s *stubLedger) GetTotalSpendByTenant(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (s *stubLedger) QueryCostTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]ledger.TimeseriesPoint, error) {
	return s.timeseries, nil
}
func (s *stubLedger) QueryRecentExpensive(_ context.Context, _, _ time.Time, _ string, _ int) ([]ledger.ExpensiveRequest, error) {
	return nil, nil
}
func (s *stubLedger) QueryErrorStats(_ context.Context, _, _ time.Time, _ string) (*ledger.ErrorStats, error) {
	return &ledger.ErrorStats{}, nil
}
func (s *stubLedger) Close() error { return nil }

func TestHandleSummary(t *testing.T) {
	store := &stubLedger{
		costs: []ledger.CostEntry{
			{Model: "gpt-4o-mini", Requests: 10, TotalCostUSD: 0.50},
			{Model: "claude-sonnet-4-6", Requests: 5, TotalCostUSD: 1.20},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/summary", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["today_requests"] != float64(15) {
		t.Errorf("today_requests = %v, want 15", resp["today_requests"])
	}
	if resp["active_sessions"] != float64(0) {
		t.Errorf("active_sessions = %v, want 0", resp["active_sessions"])
	}
}

func TestHandleTimeseries(t *testing.T) {
	store := &stubLedger{
		timeseries: []ledger.TimeseriesPoint{
			{Timestamp: time.Now(), CostUSD: 0.50, Requests: 10},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/timeseries?interval=hour&hours=24", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var points []ledger.TimeseriesPoint
	if err := json.NewDecoder(w.Body).Decode(&points); err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Errorf("expected 1 point, got %d", len(points))
	}
}

func TestHandleCosts(t *testing.T) {
	store := &stubLedger{
		costs: []ledger.CostEntry{
			{Model: "gpt-4o-mini", Requests: 10, InputTokens: 100, OutputTokens: 50, TotalCostUSD: 0.50},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/costs?group_by=model", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var entries []ledger.CostEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestHandleCostsWithTenant(t *testing.T) {
	store := &stubLedger{
		costs: []ledger.CostEntry{
			{Model: "gpt-4o-mini", Requests: 3, TotalCostUSD: 0.15},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/costs?group_by=model&tenant=alpha", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleSummaryWithTenant(t *testing.T) {
	store := &stubLedger{
		costs: []ledger.CostEntry{
			{Model: "gpt-4o-mini", Requests: 5, TotalCostUSD: 0.25},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/summary?tenant=beta", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleTimeseriesWithTenant(t *testing.T) {
	store := &stubLedger{
		timeseries: []ledger.TimeseriesPoint{
			{Timestamp: time.Now(), CostUSD: 0.10, Requests: 2},
		},
	}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/timeseries?tenant=gamma", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleSessionsWithoutTracker(t *testing.T) {
	store := &stubLedger{}
	h := NewHandler(store, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/dashboard/sessions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestStaticHandler(t *testing.T) {
	handler := StaticHandler()

	// Should serve index.html for root.
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("root status = %d, want 200", w.Code)
	}

	// Should serve style.css.
	req = httptest.NewRequest("GET", "/style.css", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("style.css status = %d, want 200", w.Code)
	}
}
