package mcp

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

func TestHTTPProxy_ForwardsAndRecords(t *testing.T) {
	// Fake upstream MCP server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "tools/call") {
			t.Error("upstream did not receive tools/call request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	}))
	defer upstream.Close()

	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pricer := NewPricer([]PricingRule{
		{Server: "unknown", Tool: "read_file", CostPerCall: 0.01},
	})
	rec := ledger.NewRecorder(store, 100, 1, logger)

	proxy := NewHTTPProxy(upstream.URL, pricer, rec, logger)

	// Create request.
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Id", "test-agent")
	req.Header.Set("X-Agent-Session", "test-sess")

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	// Verify response was forwarded.
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Error("response body not forwarded")
	}

	// Close recorder to flush.
	rec.Close()

	records := store.getRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec0 := records[0]
	if rec0.Provider != "mcp" {
		t.Errorf("provider = %q, want %q", rec0.Provider, "mcp")
	}
	if rec0.AgentID != "test-agent" {
		t.Errorf("agent_id = %q, want %q", rec0.AgentID, "test-agent")
	}
	if rec0.SessionID != "test-sess" {
		t.Errorf("session_id = %q, want %q", rec0.SessionID, "test-sess")
	}
}

func TestHTTPProxy_MethodNotAllowed(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	proxy := NewHTTPProxy("http://localhost:9999", NewPricer(nil), rec, logger)

	req := httptest.NewRequest(http.MethodGet, "/mcp/", nil)
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHTTPProxy_UpstreamError(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	// Point to a non-existent upstream.
	proxy := NewHTTPProxy("http://127.0.0.1:1", NewPricer(nil), rec, logger)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rr.Code)
	}
}

func TestHTTPProxy_NonToolCallPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"read"}]}}`))
	}))
	defer upstream.Close()

	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)

	proxy := NewHTTPProxy(upstream.URL, NewPricer(nil), rec, logger)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	rec.Close()

	// No tool call → no records.
	records := store.getRecords()
	if len(records) != 0 {
		t.Errorf("expected 0 records for tools/list, got %d", len(records))
	}
}

func TestHTTPProxy_PathTraversal(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	proxy := NewHTTPProxy("http://localhost:9999", NewPricer(nil), rec, logger)

	// Attempt path traversal.
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp/../../../etc/passwd", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("path traversal: status = %d, want 400", rr.Code)
	}
}
