package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/budget"
	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
)

// mockStore implements ledger.Ledger for testing.
type mockStore struct {
	records    []*ledger.UsageRecord
	totalSpend float64
}

func (m *mockStore) RecordUsage(_ context.Context, record *ledger.UsageRecord) error {
	m.records = append(m.records, record)
	return nil
}

func (m *mockStore) QueryCosts(_ context.Context, _ ledger.CostFilter) ([]ledger.CostEntry, error) {
	return nil, nil
}

func (m *mockStore) GetTotalSpend(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return m.totalSpend, nil
}

func (m *mockStore) Close() error { return nil }

func setupTestProxy(t *testing.T, upstream *httptest.Server) (*Proxy, *ledger.Recorder, *mockStore) {
	t.Helper()
	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	t.Cleanup(func() { rec.Close() })

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{
			Upstream: upstream.URL,
			Enabled:  true,
		},
		Anthropic: config.ProviderConfig{
			Upstream: upstream.URL,
			Enabled:  true,
		},
	})

	m := meter.New()
	p := New(reg, m, rec, nil, nil, logger)
	return p, rec, store
}

func TestHealthEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	p, _, _ := setupTestProxy(t, upstream)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("health check status = %d, want 200", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("health status = %q, want %q", resp["status"], "ok")
	}
}

func TestNoProviderMatched(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	p, _, _ := setupTestProxy(t, upstream)

	req := httptest.NewRequest("GET", "/unknown/path", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
}

func TestNonStreamingOpenAI(t *testing.T) {
	respBody := `{
		"id":"chatcmpl-abc","model":"gpt-4o-mini","object":"chat.completion",
		"choices":[{"message":{"content":"Hello"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
	}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, respBody)
	}))
	defer upstream.Close()

	p, rec, store := setupTestProxy(t, upstream)

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	// Wait for async recording.
	rec.Close()

	if len(store.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(store.records))
	}

	r := store.records[0]
	if r.Provider != "openai" {
		t.Errorf("provider = %q", r.Provider)
	}
	if r.Model != "gpt-4o-mini" {
		t.Errorf("model = %q", r.Model)
	}
	if r.InputTokens != 10 {
		t.Errorf("input_tokens = %d", r.InputTokens)
	}
	if r.OutputTokens != 5 {
		t.Errorf("output_tokens = %d", r.OutputTokens)
	}
	if r.CostUSD == 0 {
		t.Error("cost should be > 0")
	}
	if r.APIKeyHash == "" {
		t.Error("api_key_hash should not be empty")
	}
}

func TestStreamingOpenAI(t *testing.T) {
	ssePayload := "data: {\"id\":\"chatcmpl-abc\",\"model\":\"gpt-4o-mini\",\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n" +
		"data: {\"id\":\"chatcmpl-abc\",\"model\":\"gpt-4o-mini\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
		"data: [DONE]\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ssePayload)
	}))
	defer upstream.Close()

	p, rec, store := setupTestProxy(t, upstream)

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"stream":true}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// The SSE data should be passed through to the client.
	if !strings.Contains(w.Body.String(), "[DONE]") {
		t.Error("response should contain SSE data")
	}

	rec.Close()

	if len(store.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(store.records))
	}

	r := store.records[0]
	if r.InputTokens != 10 {
		t.Errorf("input_tokens = %d, want 10", r.InputTokens)
	}
	if r.OutputTokens != 5 {
		t.Errorf("output_tokens = %d, want 5", r.OutputTokens)
	}
}

func TestAgentHeadersStripped(t *testing.T) {
	var receivedHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"gpt-4o-mini","usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer upstream.Close()

	p, rec, _ := setupTestProxy(t, upstream)
	defer rec.Close()

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test")
	req.Header.Set("X-Agent-Id", "test-agent")
	req.Header.Set("X-Agent-Session", "sess_123")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if receivedHeaders.Get("X-Agent-Id") != "" {
		t.Error("X-Agent-Id should be stripped before forwarding")
	}
	if receivedHeaders.Get("X-Agent-Session") != "" {
		t.Error("X-Agent-Session should be stripped before forwarding")
	}
	if receivedHeaders.Get("Authorization") == "" {
		t.Error("Authorization should be preserved")
	}
}

func TestBudgetBlocks429(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"gpt-4o-mini","usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer upstream.Close()

	store := &mockStore{totalSpend: 100.0}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	t.Cleanup(func() { rec.Close() })

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()

	budgetMgr := budget.NewManager(store, budget.Config{
		Default: budget.Rule{
			DailyLimitUSD: 10.0,
			Action:        "block",
		},
	}, logger)

	p := New(reg, m, rec, budgetMgr, nil, logger)

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}

	var errResp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatal(err)
	}
	errObj, _ := errResp["error"].(map[string]any)
	if errObj["type"] != "budget_exceeded" {
		t.Errorf("error type = %v, want budget_exceeded", errObj["type"])
	}
}

func TestBudgetWarningHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"gpt-4o-mini","usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer upstream.Close()

	// Spend is 85% of daily limit — should trigger soft warning.
	store := &mockStore{totalSpend: 8.5}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	t.Cleanup(func() { rec.Close() })

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()

	budgetMgr := budget.NewManager(store, budget.Config{
		Default: budget.Rule{
			DailyLimitUSD: 10.0,
			SoftLimitPct:  0.8,
			Action:        "block",
		},
	}, logger)

	p := New(reg, m, rec, budgetMgr, nil, logger)

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200 (soft limit = warn, not block)", w.Code)
	}

	warning := w.Header().Get("X-AgentLedger-Budget-Warning")
	if warning == "" {
		t.Error("expected X-AgentLedger-Budget-Warning header")
	}
}

func TestPreflightRejectsExpensiveRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("request should not reach upstream")
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	// $0.50 remaining daily budget. gpt-4o-mini output is $0.60/MTok,
	// so 1M max_tokens would cost $0.60 — exceeds remaining.
	store := &mockStore{totalSpend: 9.50}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	t.Cleanup(func() { rec.Close() })

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()

	budgetMgr := budget.NewManager(store, budget.Config{
		Default: budget.Rule{
			DailyLimitUSD: 10.0,
			Action:        "block",
		},
	}, logger)

	p := New(reg, m, rec, budgetMgr, nil, logger)

	// max_tokens=1000000 → worst-case output cost = $0.60 > $0.50 remaining
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"max_tokens":1000000}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (pre-flight rejection)", w.Code)
	}

	var errResp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatal(err)
	}
	errObj, _ := errResp["error"].(map[string]any)
	if errObj["type"] != "budget_exceeded" {
		t.Errorf("error type = %v, want budget_exceeded", errObj["type"])
	}
	if errObj["estimated_cost"] == nil {
		t.Error("expected estimated_cost in pre-flight error")
	}
}

func TestPreflightAllowsCheapRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"gpt-4o-mini","usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`)
	}))
	defer upstream.Close()

	// $5.00 remaining. max_tokens=100 of gpt-4o-mini = $0.00006 — well under.
	store := &mockStore{totalSpend: 5.0}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	t.Cleanup(func() { rec.Close() })

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()

	budgetMgr := budget.NewManager(store, budget.Config{
		Default: budget.Rule{
			DailyLimitUSD: 10.0,
			Action:        "block",
		},
	}, logger)

	p := New(reg, m, rec, budgetMgr, nil, logger)

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"max_tokens":100}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200 (cheap request should pass pre-flight)", w.Code)
	}
}
