package mcp

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

// recordingLedger captures records for test assertions.
type recordingLedger struct {
	mu      sync.Mutex
	records []*ledger.UsageRecord
	count   atomic.Int64
}

func (r *recordingLedger) RecordUsage(_ context.Context, rec *ledger.UsageRecord) error {
	r.mu.Lock()
	r.records = append(r.records, rec)
	r.mu.Unlock()
	r.count.Add(1)
	return nil
}

func (r *recordingLedger) QueryCosts(_ context.Context, _ ledger.CostFilter) ([]ledger.CostEntry, error) {
	return nil, nil
}

func (r *recordingLedger) GetTotalSpend(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (r *recordingLedger) GetTotalSpendByTenant(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}

func (r *recordingLedger) QueryCostTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]ledger.TimeseriesPoint, error) {
	return nil, nil
}
func (r *recordingLedger) QueryRecentExpensive(_ context.Context, _, _ time.Time, _ string, _ int) ([]ledger.ExpensiveRequest, error) {
	return nil, nil
}
func (r *recordingLedger) QueryErrorStats(_ context.Context, _, _ time.Time, _ string) (*ledger.ErrorStats, error) {
	return &ledger.ErrorStats{}, nil
}
func (r *recordingLedger) QueryRecentSessions(_ context.Context, _, _ time.Time, _ string, _ int) ([]ledger.SessionRecord, error) {
	return nil, nil
}
func (r *recordingLedger) QueryLatencyPercentiles(_ context.Context, _, _ time.Time, _ string) (*ledger.LatencyStats, error) {
	return &ledger.LatencyStats{}, nil
}
func (r *recordingLedger) QueryTokenTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]ledger.TokenTimeseriesPoint, error) {
	return nil, nil
}

func (r *recordingLedger) Close() error { return nil }

func (r *recordingLedger) getRecords() []*ledger.UsageRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*ledger.UsageRecord, len(r.records))
	copy(out, r.records)
	return out
}

func newTestInterceptor(store *recordingLedger) *Interceptor {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pricer := NewPricer([]PricingRule{
		{Server: "filesystem", Tool: "read_file", CostPerCall: 0.01},
		{Server: "filesystem", Tool: "", CostPerCall: 0.005},
	})
	rec := ledger.NewRecorder(store, 100, 1, logger)
	return NewInterceptor("filesystem", pricer, rec, "agent-1", "sess-1", "user-1", logger)
}

func TestInterceptor_ToolCallRecorded(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	// Client sends tools/call
	reqMsg := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/tmp"}}}`)
	interceptor.HandleMessage(reqMsg, true, nil)

	// Server responds
	respMsg := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"file contents"}]}}`)
	interceptor.HandleMessage(respMsg, false, nil)

	// Close recorder to flush
	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0]
	if rec.Provider != "mcp" {
		t.Errorf("provider = %q, want %q", rec.Provider, "mcp")
	}
	if rec.Model != "filesystem:read_file" {
		t.Errorf("model = %q, want %q", rec.Model, "filesystem:read_file")
	}
	if rec.CostUSD != 0.01 {
		t.Errorf("cost = %f, want 0.01", rec.CostUSD)
	}
	if rec.AgentID != "agent-1" {
		t.Errorf("agent_id = %q, want %q", rec.AgentID, "agent-1")
	}
	if rec.Path != "tools/call" {
		t.Errorf("path = %q, want %q", rec.Path, "tools/call")
	}
	if rec.StatusCode != 200 {
		t.Errorf("status_code = %d, want 200", rec.StatusCode)
	}
	if rec.DurationMS < 0 {
		t.Errorf("duration_ms = %d, want >= 0", rec.DurationMS)
	}
}

func TestInterceptor_ErrorResponse(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	reqMsg := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_file","arguments":{}}}`)
	interceptor.HandleMessage(reqMsg, true, nil)

	respMsg := []byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"invalid path"}}`)
	interceptor.HandleMessage(respMsg, false, nil)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].StatusCode != 500 {
		t.Errorf("status_code = %d, want 500", records[0].StatusCode)
	}
}

func TestInterceptor_WildcardPricing(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	reqMsg := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_directory","arguments":{}}}`)
	interceptor.HandleMessage(reqMsg, true, nil)

	respMsg := []byte(`{"jsonrpc":"2.0","id":3,"result":{}}`)
	interceptor.HandleMessage(respMsg, false, nil)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].CostUSD != 0.005 {
		t.Errorf("cost = %f, want 0.005 (wildcard)", records[0].CostUSD)
	}
}

func TestInterceptor_AgentContextOverride(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	ctx := &AgentContext{
		AgentID:   "http-agent",
		SessionID: "http-sess",
		UserID:    "http-user",
	}

	reqMsg := []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"read_file","arguments":{}}}`)
	interceptor.HandleMessage(reqMsg, true, ctx)

	respMsg := []byte(`{"jsonrpc":"2.0","id":4,"result":{}}`)
	interceptor.HandleMessage(respMsg, false, ctx)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].AgentID != "http-agent" {
		t.Errorf("agent_id = %q, want %q", records[0].AgentID, "http-agent")
	}
	if records[0].SessionID != "http-sess" {
		t.Errorf("session_id = %q, want %q", records[0].SessionID, "http-sess")
	}
}

func TestInterceptor_UnknownIDIgnored(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	// Response with no matching request
	respMsg := []byte(`{"jsonrpc":"2.0","id":99,"result":{}}`)
	interceptor.HandleMessage(respMsg, false, nil)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestInterceptor_NotificationIgnored(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	// Notification (no id)
	msg := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	interceptor.HandleMessage(msg, true, nil)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestInterceptor_InitializeExtractsServerName(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	// Simulate initialize response (not matched to inflight)
	respMsg := []byte(`{"jsonrpc":"2.0","id":1,"result":{"serverInfo":{"name":"my-server","version":"2.0"}}}`)
	interceptor.HandleMessage(respMsg, false, nil)

	if interceptor.ServerName() != "my-server" {
		t.Errorf("server name = %q, want %q", interceptor.ServerName(), "my-server")
	}
}

func TestInterceptor_ConcurrentCalls(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reqMsg := []byte(`{"jsonrpc":"2.0","id":` + string(rune('0'+id%10)) + `,"method":"tools/call","params":{"name":"read_file","arguments":{}}}`)
			interceptor.HandleMessage(reqMsg, true, nil)
		}(i)
	}
	wg.Wait()

	// No panics = pass. The inflight map is properly synchronized.
}

func TestInterceptor_NonJSONIgnored(t *testing.T) {
	store := &recordingLedger{}
	interceptor := newTestInterceptor(store)

	interceptor.HandleMessage([]byte("this is not json"), true, nil)
	interceptor.HandleMessage([]byte(""), false, nil)

	interceptor.recorder.Close()

	records := store.getRecords()
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
