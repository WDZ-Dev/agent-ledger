package agent

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// stubSessionStore implements SessionStore for testing.
type stubSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func newStubStore() *stubSessionStore {
	return &stubSessionStore{sessions: make(map[string]*Session)}
}

func (s *stubSessionStore) UpsertSession(_ context.Context, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *sess
	s.sessions[sess.ID] = &cp
	return nil
}

func (s *stubSessionStore) GetSession(_ context.Context, id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, context.Canceled // simulate not found
	}
	return sess, nil
}

func (s *stubSessionStore) ListActiveSessions(_ context.Context) ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Session
	for _, sess := range s.sessions {
		if sess.Status == StatusActive {
			out = append(out, *sess)
		}
	}
	return out, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTrackCall_CreatesSession(t *testing.T) {
	store := newStubStore()
	cfg := Config{
		SessionTimeoutMins: 30,
		LoopThreshold:      0,
		GhostMaxAgeMins:    0,
	}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	alert := tracker.TrackCall("sess1", "agent1", "user1", "test task", "gpt-4o", "/v1/chat/completions")
	if alert != nil {
		t.Error("expected no alert on first call")
	}

	// Call again — should increment.
	tracker.TrackCall("sess1", "agent1", "user1", "test task", "gpt-4o", "/v1/chat/completions")

	tracker.mu.RLock()
	ts := tracker.sessions["sess1"]
	tracker.mu.RUnlock()

	if ts == nil {
		t.Fatal("session not found")
	}
	if ts.session.CallCount != 2 {
		t.Errorf("call_count = %d, want 2", ts.session.CallCount)
	}
	if ts.session.AgentID != "agent1" {
		t.Errorf("agent_id = %q", ts.session.AgentID)
	}
	if ts.session.Task != "test task" {
		t.Errorf("task = %q", ts.session.Task)
	}
}

func TestTrackCall_LoopDetection(t *testing.T) {
	store := newStubStore()
	cfg := Config{
		SessionTimeoutMins: 30,
		LoopThreshold:      3,
		LoopWindowMins:     5,
		LoopAction:         "block",
	}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	// 3 calls to the same path should trigger loop.
	for i := 0; i < 3; i++ {
		alert := tracker.TrackCall("sess1", "agent1", "user1", "task", "gpt-4o", "/v1/chat/completions")
		if i < 2 && alert != nil {
			t.Errorf("unexpected alert on call %d", i+1)
		}
		if i == 2 && alert == nil {
			t.Error("expected loop alert on call 3")
		}
	}
}

func TestTrackCall_NoLoopDifferentPaths(t *testing.T) {
	store := newStubStore()
	cfg := Config{
		SessionTimeoutMins: 30,
		LoopThreshold:      3,
		LoopWindowMins:     5,
	}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	paths := []string{"/v1/chat/completions", "/v1/embeddings", "/v1/models"}
	for _, p := range paths {
		alert := tracker.TrackCall("sess1", "agent1", "user1", "task", "gpt-4o", p)
		if alert != nil {
			t.Errorf("unexpected alert for path %s", p)
		}
	}
}

func TestRecordCost(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 30}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	tracker.TrackCall("sess1", "agent1", "user1", "task", "gpt-4o", "/v1/chat/completions")
	tracker.RecordCost("sess1", 0.05, 100)
	tracker.RecordCost("sess1", 0.03, 50)

	tracker.mu.RLock()
	ts := tracker.sessions["sess1"]
	tracker.mu.RUnlock()

	if ts.session.TotalCostUSD != 0.08 {
		t.Errorf("total_cost = %.2f, want 0.08", ts.session.TotalCostUSD)
	}
	if ts.session.TotalTokens != 150 {
		t.Errorf("total_tokens = %d, want 150", ts.session.TotalTokens)
	}
}

func TestRecordCost_UnknownSession(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 30}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	// Should not panic.
	tracker.RecordCost("nonexistent", 1.0, 100)
}

func TestEndSession(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 30}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	tracker.TrackCall("sess1", "agent1", "user1", "task", "gpt-4o", "/v1/chat/completions")
	tracker.EndSession("sess1")

	tracker.mu.RLock()
	ts := tracker.sessions["sess1"]
	tracker.mu.RUnlock()

	if ts.session.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", ts.session.Status)
	}
	if ts.session.EndedAt == nil {
		t.Error("ended_at should be set")
	}
}

func TestShouldBlock(t *testing.T) {
	store := newStubStore()

	cfg1 := Config{LoopAction: "block"}
	t1 := NewTracker(store, cfg1, nil, testLogger())
	defer t1.Close()
	if !t1.ShouldBlock() {
		t.Error("should block when action=block")
	}

	cfg2 := Config{LoopAction: "warn"}
	t2 := NewTracker(store, cfg2, nil, testLogger())
	defer t2.Close()
	if t2.ShouldBlock() {
		t.Error("should not block when action=warn")
	}
}

func TestEnabled(t *testing.T) {
	store := newStubStore()

	// Nothing enabled
	t1 := NewTracker(store, Config{}, nil, testLogger())
	defer t1.Close()
	if t1.Enabled() {
		t.Error("should not be enabled with zero config")
	}

	// Loop detection enabled
	t2 := NewTracker(store, Config{LoopThreshold: 10}, nil, testLogger())
	defer t2.Close()
	if !t2.Enabled() {
		t.Error("should be enabled with loop threshold")
	}

	// Ghost detection enabled
	t3 := NewTracker(store, Config{GhostMaxAgeMins: 60}, nil, testLogger())
	defer t3.Close()
	if !t3.Enabled() {
		t.Error("should be enabled with ghost max age")
	}
}

func TestFlush_PersistsToStore(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 30}
	tracker := NewTracker(store, cfg, nil, testLogger())

	tracker.TrackCall("sess1", "agent1", "user1", "task", "gpt-4o", "/v1/chat/completions")
	tracker.RecordCost("sess1", 0.10, 200)

	// Flush by closing.
	tracker.Close()

	store.mu.Lock()
	sess, ok := store.sessions["sess1"]
	store.mu.Unlock()

	if !ok {
		t.Fatal("session not persisted to store")
	}
	if sess.CallCount != 1 {
		t.Errorf("call_count = %d, want 1", sess.CallCount)
	}
	if sess.TotalCostUSD != 0.10 {
		t.Errorf("total_cost = %.2f, want 0.10", sess.TotalCostUSD)
	}
}

func TestExpireIdleSessions(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 1} // 1 minute timeout
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	// Create a session with a call that's 2 minutes old.
	tracker.mu.Lock()
	tracker.sessions["sess1"] = &trackedSession{
		session: Session{
			ID:        "sess1",
			AgentID:   "agent1",
			StartedAt: time.Now().Add(-5 * time.Minute),
			Status:    StatusActive,
			CallCount: 1,
		},
		calls: []CallRecord{
			{Timestamp: time.Now().Add(-2 * time.Minute), Path: "/v1/chat/completions"},
		},
	}
	tracker.mu.Unlock()

	tracker.expireIdleSessions()

	tracker.mu.RLock()
	ts := tracker.sessions["sess1"]
	tracker.mu.RUnlock()

	if ts.session.Status != StatusCompleted {
		t.Errorf("status = %q, want completed (should be expired)", ts.session.Status)
	}
}

func TestTrackerEvictsOldestSession(t *testing.T) {
	store := newStubStore()
	cfg := Config{SessionTimeoutMins: 30}
	tracker := NewTracker(store, cfg, nil, testLogger())
	defer tracker.Close()

	// Fill to max capacity.
	tracker.mu.Lock()
	for i := 0; i < maxActiveSessions; i++ {
		id := "sess-" + time.Now().Add(time.Duration(i)*time.Millisecond).Format("150405.000000")
		tracker.sessions[id] = &trackedSession{
			session: Session{
				ID:        id,
				Status:    StatusActive,
				StartedAt: time.Now(),
			},
			calls: []CallRecord{
				{Timestamp: time.Now()},
			},
		}
	}
	// Add one old session that should be evicted.
	tracker.sessions["oldest"] = &trackedSession{
		session: Session{
			ID:        "oldest",
			Status:    StatusActive,
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
		calls: []CallRecord{
			{Timestamp: time.Now().Add(-1 * time.Hour)},
		},
	}
	tracker.mu.Unlock()

	// This should evict the oldest session to make room.
	tracker.TrackCall("new-session", "agent1", "user1", "task", "gpt-4o", "/v1/chat/completions")

	tracker.mu.RLock()
	_, oldestExists := tracker.sessions["oldest"]
	_, newExists := tracker.sessions["new-session"]
	tracker.mu.RUnlock()

	if oldestExists {
		t.Error("oldest session should have been evicted")
	}
	if !newExists {
		t.Error("new session should have been created")
	}
}
