package agent

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Session represents an agent execution run — a group of related LLM calls.
type Session struct {
	ID           string
	AgentID      string
	UserID       string
	Task         string
	StartedAt    time.Time
	EndedAt      *time.Time
	Status       string // "active", "completed", "killed"
	CallCount    int
	TotalCostUSD float64
	TotalTokens  int
}

// CallRecord tracks a single request within a session for loop detection.
type CallRecord struct {
	Timestamp time.Time
	Model     string
	Path      string
}

// SessionStore persists agent sessions.
type SessionStore interface {
	UpsertSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	ListActiveSessions(ctx context.Context) ([]Session, error)
}

// Config holds agent tracking settings.
type Config struct {
	SessionTimeoutMins int     `mapstructure:"session_timeout_mins"`
	LoopThreshold      int     `mapstructure:"loop_threshold"`
	LoopWindowMins     int     `mapstructure:"loop_window_mins"`
	LoopAction         string  `mapstructure:"loop_action"`
	GhostMaxAgeMins    int     `mapstructure:"ghost_max_age_mins"`
	GhostMinCalls      int     `mapstructure:"ghost_min_calls"`
	GhostMinCostUSD    float64 `mapstructure:"ghost_min_cost_usd"`
}

// Tracker manages agent session lifecycle, loop detection, and ghost detection.
type Tracker struct {
	store    SessionStore
	detector *Detector
	logger   *slog.Logger
	cfg      Config

	mu       sync.RWMutex
	sessions map[string]*trackedSession

	done   chan struct{}
	closed sync.Once
}

type trackedSession struct {
	session      Session
	calls        []CallRecord
	dirty        bool
	ghostAlerted bool
}

const (
	flushInterval = 10 * time.Second

	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusKilled    = "killed"
)

// NewTracker creates a session tracker with background flush and ghost detection.
// The background goroutine only starts when tracking features are configured.
func NewTracker(store SessionStore, cfg Config, logger *slog.Logger) *Tracker {
	t := &Tracker{
		store:    store,
		detector: NewDetector(cfg, logger),
		logger:   logger,
		cfg:      cfg,
		sessions: make(map[string]*trackedSession),
		done:     make(chan struct{}),
	}
	if t.Enabled() {
		go t.backgroundLoop()
	}
	return t
}

// Enabled returns true if any agent tracking features are configured.
func (t *Tracker) Enabled() bool {
	return t.cfg.LoopThreshold > 0 || t.cfg.GhostMaxAgeMins > 0
}

// TrackCall records a request in a session and runs loop detection.
// Returns an Alert if a loop is detected.
func (t *Tracker) TrackCall(sessionID, agentID, userID, task, model, path string) *Alert {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts, ok := t.sessions[sessionID]
	if !ok {
		ts = &trackedSession{
			session: Session{
				ID:        sessionID,
				AgentID:   agentID,
				UserID:    userID,
				Task:      task,
				StartedAt: time.Now(),
				Status:    StatusActive,
			},
		}
		t.sessions[sessionID] = ts
		t.logger.Info("session started",
			"session_id", sessionID,
			"agent_id", agentID,
		)
	}

	call := CallRecord{
		Timestamp: time.Now(),
		Model:     model,
		Path:      path,
	}
	ts.calls = append(ts.calls, call)
	ts.session.CallCount++
	ts.dirty = true

	// Trim call history older than the loop window to bound memory.
	if t.detector.loopWindow > 0 && len(ts.calls) > t.cfg.LoopThreshold*10 {
		cutoff := time.Now().Add(-t.detector.loopWindow)
		trimIdx := 0
		for trimIdx < len(ts.calls) && ts.calls[trimIdx].Timestamp.Before(cutoff) {
			trimIdx++
		}
		if trimIdx > 0 {
			ts.calls = ts.calls[trimIdx:]
		}
	}

	// Loop detection runs under the same lock to avoid racing on ts.calls.
	if t.cfg.LoopThreshold > 0 {
		alert := t.detector.CheckLoop(ts.calls, path, sessionID, agentID)
		if alert != nil {
			t.logger.Warn("loop detected",
				"session_id", sessionID,
				"agent_id", agentID,
				"path", path,
			)
			return alert
		}
	}

	return nil
}

// RecordCost updates session totals after response cost is calculated.
func (t *Tracker) RecordCost(sessionID string, costUSD float64, tokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts, ok := t.sessions[sessionID]
	if !ok {
		return
	}
	ts.session.TotalCostUSD += costUSD
	ts.session.TotalTokens += tokens
	ts.dirty = true
}

// EndSession marks a session as completed.
func (t *Tracker) EndSession(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts, ok := t.sessions[sessionID]
	if !ok {
		return
	}
	now := time.Now()
	ts.session.EndedAt = &now
	ts.session.Status = StatusCompleted
	ts.dirty = true

	t.logger.Info("session ended",
		"session_id", sessionID,
		"agent_id", ts.session.AgentID,
		"calls", ts.session.CallCount,
		"cost_usd", ts.session.TotalCostUSD,
	)
}

// ShouldBlock returns true if loop detection should block requests.
func (t *Tracker) ShouldBlock() bool {
	return t.cfg.LoopAction == "block"
}

// Close flushes all sessions and stops the background goroutine.
func (t *Tracker) Close() {
	t.closed.Do(func() {
		close(t.done)
		t.flush()
	})
}

func (t *Tracker) backgroundLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.flush()
			t.expireIdleSessions()
			t.detectGhosts()
		case <-t.done:
			return
		}
	}
}

func (t *Tracker) flush() {
	t.mu.Lock()
	var toFlush []Session
	for _, ts := range t.sessions {
		if ts.dirty {
			toFlush = append(toFlush, ts.session)
			ts.dirty = false
		}
	}
	// Remove completed/killed sessions from memory after flushing.
	for id, ts := range t.sessions {
		if ts.session.Status != StatusActive && !ts.dirty {
			delete(t.sessions, id)
		}
	}
	t.mu.Unlock()

	ctx := context.Background()
	for i := range toFlush {
		if err := t.store.UpsertSession(ctx, &toFlush[i]); err != nil {
			t.logger.Error("flushing session", "error", err, "session_id", toFlush[i].ID)
		}
	}
}

func (t *Tracker) expireIdleSessions() {
	if t.cfg.SessionTimeoutMins <= 0 {
		return
	}
	timeout := time.Duration(t.cfg.SessionTimeoutMins) * time.Minute

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for _, ts := range t.sessions {
		if ts.session.Status != StatusActive {
			continue
		}
		lastActivity := ts.session.StartedAt
		if len(ts.calls) > 0 {
			lastActivity = ts.calls[len(ts.calls)-1].Timestamp
		}
		if now.Sub(lastActivity) > timeout {
			ts.session.EndedAt = &now
			ts.session.Status = StatusCompleted
			ts.dirty = true
			t.logger.Info("session timed out",
				"session_id", ts.session.ID,
				"agent_id", ts.session.AgentID,
			)
		}
	}
}

func (t *Tracker) detectGhosts() {
	if t.cfg.GhostMaxAgeMins <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, ts := range t.sessions {
		if ts.session.Status != StatusActive || ts.ghostAlerted {
			continue
		}
		alert := t.detector.CheckGhost(&ts.session)
		if alert != nil {
			ts.ghostAlerted = true
			t.logger.Warn("ghost agent detected",
				"session_id", ts.session.ID,
				"agent_id", ts.session.AgentID,
				"calls", ts.session.CallCount,
				"cost_usd", ts.session.TotalCostUSD,
				"age", time.Since(ts.session.StartedAt).String(),
			)
		}
	}
}
