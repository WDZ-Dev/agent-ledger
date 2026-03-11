package agent

import (
	"log/slog"
	"testing"
	"time"
)

func testDetector(loopThreshold int, loopWindowMins int, ghostMaxAgeMins int) *Detector {
	cfg := Config{
		LoopThreshold:   loopThreshold,
		LoopWindowMins:  loopWindowMins,
		GhostMaxAgeMins: ghostMaxAgeMins,
		GhostMinCalls:   50,
		GhostMinCostUSD: 1.0,
	}
	return NewDetector(cfg, slog.Default())
}

func TestCheckLoop_NoLoop(t *testing.T) {
	d := testDetector(5, 5, 0)

	calls := []CallRecord{
		{Timestamp: time.Now(), Path: "/v1/chat/completions"},
		{Timestamp: time.Now(), Path: "/v1/chat/completions"},
		{Timestamp: time.Now(), Path: "/v1/embeddings"},
	}

	alert := d.CheckLoop(calls, "/v1/chat/completions", "sess1", "agent1")
	if alert != nil {
		t.Error("expected no alert for 2 calls (threshold is 5)")
	}
}

func TestCheckLoop_Detected(t *testing.T) {
	d := testDetector(3, 5, 0)

	now := time.Now()
	calls := []CallRecord{
		{Timestamp: now.Add(-2 * time.Minute), Path: "/v1/chat/completions"},
		{Timestamp: now.Add(-1 * time.Minute), Path: "/v1/chat/completions"},
		{Timestamp: now, Path: "/v1/chat/completions"},
	}

	alert := d.CheckLoop(calls, "/v1/chat/completions", "sess1", "agent1")
	if alert == nil {
		t.Fatal("expected loop alert")
	}
	if alert.Type != "loop_detected" {
		t.Errorf("alert type = %q, want loop_detected", alert.Type)
	}
	if alert.SessionID != "sess1" {
		t.Errorf("session_id = %q", alert.SessionID)
	}
}

func TestCheckLoop_OutsideWindow(t *testing.T) {
	d := testDetector(3, 5, 0)

	now := time.Now()
	calls := []CallRecord{
		{Timestamp: now.Add(-10 * time.Minute), Path: "/v1/chat/completions"},
		{Timestamp: now.Add(-8 * time.Minute), Path: "/v1/chat/completions"},
		{Timestamp: now.Add(-6 * time.Minute), Path: "/v1/chat/completions"},
	}

	alert := d.CheckLoop(calls, "/v1/chat/completions", "sess1", "agent1")
	if alert != nil {
		t.Error("expected no alert — calls are outside the 5 minute window")
	}
}

func TestCheckLoop_Disabled(t *testing.T) {
	d := testDetector(0, 5, 0) // threshold=0 disables

	calls := []CallRecord{
		{Timestamp: time.Now(), Path: "/v1/chat/completions"},
	}

	alert := d.CheckLoop(calls, "/v1/chat/completions", "sess1", "agent1")
	if alert != nil {
		t.Error("expected no alert when loop detection is disabled")
	}
}

func TestCheckGhost_Detected(t *testing.T) {
	d := testDetector(0, 0, 60)

	s := &Session{
		ID:           "sess1",
		AgentID:      "agent1",
		StartedAt:    time.Now().Add(-2 * time.Hour),
		Status:       StatusActive,
		CallCount:    100,
		TotalCostUSD: 5.0,
	}

	alert := d.CheckGhost(s)
	if alert == nil {
		t.Fatal("expected ghost alert")
	}
	if alert.Type != "ghost_detected" {
		t.Errorf("alert type = %q, want ghost_detected", alert.Type)
	}
}

func TestCheckGhost_TooYoung(t *testing.T) {
	d := testDetector(0, 0, 60)

	s := &Session{
		ID:           "sess1",
		AgentID:      "agent1",
		StartedAt:    time.Now().Add(-30 * time.Minute), // only 30min old
		Status:       StatusActive,
		CallCount:    100,
		TotalCostUSD: 5.0,
	}

	alert := d.CheckGhost(s)
	if alert != nil {
		t.Error("expected no alert — session is too young")
	}
}

func TestCheckGhost_TooFewCalls(t *testing.T) {
	d := testDetector(0, 0, 60)

	s := &Session{
		ID:           "sess1",
		AgentID:      "agent1",
		StartedAt:    time.Now().Add(-2 * time.Hour),
		Status:       StatusActive,
		CallCount:    10, // below min of 50
		TotalCostUSD: 5.0,
	}

	alert := d.CheckGhost(s)
	if alert != nil {
		t.Error("expected no alert — too few calls")
	}
}

func TestCheckGhost_TooLowCost(t *testing.T) {
	d := testDetector(0, 0, 60)

	s := &Session{
		ID:           "sess1",
		AgentID:      "agent1",
		StartedAt:    time.Now().Add(-2 * time.Hour),
		Status:       StatusActive,
		CallCount:    100,
		TotalCostUSD: 0.50, // below min of $1.0
	}

	alert := d.CheckGhost(s)
	if alert != nil {
		t.Error("expected no alert — cost too low")
	}
}

func TestCheckGhost_Disabled(t *testing.T) {
	d := testDetector(0, 0, 0) // disabled

	s := &Session{
		ID:           "sess1",
		AgentID:      "agent1",
		StartedAt:    time.Now().Add(-2 * time.Hour),
		Status:       StatusActive,
		CallCount:    100,
		TotalCostUSD: 5.0,
	}

	alert := d.CheckGhost(s)
	if alert != nil {
		t.Error("expected no alert when ghost detection is disabled")
	}
}
