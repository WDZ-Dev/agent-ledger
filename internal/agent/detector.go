package agent

import (
	"fmt"
	"log/slog"
	"time"
)

// Alert represents a detected anomaly in agent behavior.
type Alert struct {
	Type      string // "loop_detected", "ghost_detected"
	SessionID string
	AgentID   string
	Message   string
}

// Detector checks for loop and ghost agent patterns.
type Detector struct {
	loopThreshold int
	loopWindow    time.Duration
	ghostMaxAge   time.Duration
	ghostMinCalls int
	ghostMinCost  float64
	logger        *slog.Logger
}

// NewDetector creates a pattern detector from config.
func NewDetector(cfg Config, logger *slog.Logger) *Detector {
	return &Detector{
		loopThreshold: cfg.LoopThreshold,
		loopWindow:    time.Duration(cfg.LoopWindowMins) * time.Minute,
		ghostMaxAge:   time.Duration(cfg.GhostMaxAgeMins) * time.Minute,
		ghostMinCalls: cfg.GhostMinCalls,
		ghostMinCost:  cfg.GhostMinCostUSD,
		logger:        logger,
	}
}

// CheckLoop examines the call history for repetitive patterns.
// A loop is detected when the same path appears >= threshold times
// within the sliding window.
func (d *Detector) CheckLoop(calls []CallRecord, currentPath, sessionID, agentID string) *Alert {
	if d.loopThreshold <= 0 || d.loopWindow <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-d.loopWindow)
	count := 0
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Timestamp.Before(cutoff) {
			break
		}
		if calls[i].Path == currentPath {
			count++
		}
	}

	if count >= d.loopThreshold {
		return &Alert{
			Type:      "loop_detected",
			SessionID: sessionID,
			AgentID:   agentID,
			Message: fmt.Sprintf("path %q called %d times in %s (threshold: %d)",
				currentPath, count, d.loopWindow, d.loopThreshold),
		}
	}

	return nil
}

// CheckGhost examines a session for ghost agent patterns — sessions that
// have been running too long with too many calls and too much spend
// without being explicitly ended.
func (d *Detector) CheckGhost(s *Session) *Alert {
	if d.ghostMaxAge <= 0 {
		return nil
	}

	age := time.Since(s.StartedAt)
	if age < d.ghostMaxAge {
		return nil
	}
	if s.CallCount < d.ghostMinCalls {
		return nil
	}
	if s.TotalCostUSD < d.ghostMinCost {
		return nil
	}

	return &Alert{
		Type:      "ghost_detected",
		SessionID: s.ID,
		AgentID:   s.AgentID,
		Message: fmt.Sprintf("session active %s, %d calls, $%.2f spent",
			age.Truncate(time.Minute), s.CallCount, s.TotalCostUSD),
	}
}
