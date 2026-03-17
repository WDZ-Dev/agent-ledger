package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

// Handler serves the dashboard REST API.
type Handler struct {
	ledger  ledger.Ledger
	tracker *agent.Tracker
}

// NewHandler creates a dashboard API handler.
func NewHandler(l ledger.Ledger, tracker *agent.Tracker) *Handler {
	return &Handler{ledger: l, tracker: tracker}
}

// RegisterRoutes registers dashboard API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard/summary", h.handleSummary)
	mux.HandleFunc("GET /api/dashboard/timeseries", h.handleTimeseries)
	mux.HandleFunc("GET /api/dashboard/costs", h.handleCosts)
	mux.HandleFunc("GET /api/dashboard/sessions", h.handleSessions)
	mux.HandleFunc("GET /api/dashboard/export", h.handleExport)
	mux.HandleFunc("GET /api/dashboard/expensive", h.handleExpensive)
	mux.HandleFunc("GET /api/dashboard/stats", h.handleStats)
}

func (h *Handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	tenantID := r.URL.Query().Get("tenant")

	// Get today's costs by model.
	todayCosts, err := h.ledger.QueryCosts(r.Context(), ledger.CostFilter{
		Since:    dayStart,
		Until:    now,
		GroupBy:  "model",
		TenantID: tenantID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var todaySpend float64
	var todayRequests int
	for _, e := range todayCosts {
		todaySpend += e.TotalCostUSD
		todayRequests += e.Requests
	}

	// Get month's costs.
	monthCosts, err := h.ledger.QueryCosts(r.Context(), ledger.CostFilter{
		Since:    monthStart,
		Until:    now,
		GroupBy:  "model",
		TenantID: tenantID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var monthSpend float64
	for _, e := range monthCosts {
		monthSpend += e.TotalCostUSD
	}

	// Active sessions count.
	var activeSessions int
	if h.tracker != nil {
		activeSessions = h.tracker.ActiveSessionCount()
	}

	writeJSON(w, map[string]any{
		"today_spend_usd": todaySpend,
		"month_spend_usd": monthSpend,
		"today_requests":  todayRequests,
		"active_sessions": activeSessions,
	})
}

func (h *Handler) handleTimeseries(w http.ResponseWriter, r *http.Request) {
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "hour"
	}

	hoursF, _ := strconv.ParseFloat(r.URL.Query().Get("hours"), 64)
	if hoursF <= 0 {
		hoursF = 24
	}

	tenantID := r.URL.Query().Get("tenant")

	now := time.Now().UTC()
	since := now.Add(-time.Duration(hoursF * float64(time.Hour)))

	points, err := h.ledger.QueryCostTimeseries(r.Context(), interval, since, now, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, points)
}

func (h *Handler) handleCosts(w http.ResponseWriter, r *http.Request) {
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "model"
	}

	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 24
	}

	tenantID := r.URL.Query().Get("tenant")

	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	entries, err := h.ledger.QueryCosts(r.Context(), ledger.CostFilter{
		Since:    since,
		Until:    now,
		GroupBy:  groupBy,
		TenantID: tenantID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, entries)
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if h.tracker == nil {
		writeJSON(w, []any{})
		return
	}

	sessions := h.tracker.ListSessions()
	writeJSON(w, sessions)
}

func (h *Handler) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "model"
	}

	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 720 // 30 days
	}

	tenantID := r.URL.Query().Get("tenant")

	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	entries, err := h.ledger.QueryCosts(r.Context(), ledger.CostFilter{
		Since:    since,
		Until:    now,
		GroupBy:  groupBy,
		TenantID: tenantID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="agentledger-costs.csv"`)

		cw := csv.NewWriter(w)
		defer cw.Flush()

		header := []string{
			"provider", "model", "api_key_hash", "agent_id", "session_id",
			"requests", "input_tokens", "output_tokens", "cost_usd",
		}
		if err := cw.Write(header); err != nil {
			return
		}

		for _, e := range entries {
			record := []string{
				e.Provider,
				e.Model,
				e.APIKeyHash,
				e.AgentID,
				e.SessionID,
				strconv.Itoa(e.Requests),
				strconv.FormatInt(e.InputTokens, 10),
				strconv.FormatInt(e.OutputTokens, 10),
				fmt.Sprintf("%.6f", e.TotalCostUSD),
			}
			if err := cw.Write(record); err != nil {
				return
			}
		}
	default:
		writeJSON(w, entries)
	}
}

func (h *Handler) handleExpensive(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 168
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	tenantID := r.URL.Query().Get("tenant")
	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	results, err := h.ledger.QueryRecentExpensive(r.Context(), since, now, tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, results)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 24
	}
	tenantID := r.URL.Query().Get("tenant")
	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	stats, err := h.ledger.QueryErrorStats(r.Context(), since, now, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, stats)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
