package admin

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/budget"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

// Handler serves the admin REST API.
type Handler struct {
	store     *Store
	ledger    ledger.Ledger
	budgetMgr *budget.Manager
	token     string // admin authentication token
	blocklist *Blocklist
	logger    *slog.Logger
}

// NewHandler creates an admin API handler.
func NewHandler(store *Store, l ledger.Ledger, budgetMgr *budget.Manager, token string, blocklist *Blocklist, logger *slog.Logger) *Handler {
	return &Handler{
		store:     store,
		ledger:    l,
		budgetMgr: budgetMgr,
		token:     token,
		blocklist: blocklist,
		logger:    logger,
	}
}

// RegisterRoutes registers admin API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/budgets/rules", h.requireAuth(h.handleListRules))
	mux.HandleFunc("POST /api/admin/budgets/rules", h.requireAuth(h.handleCreateRule))
	mux.HandleFunc("DELETE /api/admin/budgets/rules", h.requireAuth(h.handleDeleteRule))
	mux.HandleFunc("GET /api/admin/api-keys", h.requireAuth(h.handleListAPIKeys))
	mux.HandleFunc("GET /api/admin/api-keys/blocked", h.requireAuth(h.handleListBlocked))
	mux.HandleFunc("POST /api/admin/api-keys/block", h.requireAuth(h.handleBlockKey))
	mux.HandleFunc("DELETE /api/admin/api-keys/block", h.requireAuth(h.handleUnblockKey))
	mux.HandleFunc("GET /api/admin/providers", h.requireAuth(h.handleListProviders))
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.token == "" {
			writeAdminError(w, http.StatusForbidden, "admin API not configured")
			return
		}
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + h.token
		if subtle.ConstantTimeCompare([]byte(auth), []byte(expected)) != 1 {
			writeAdminError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}
		if r.Method != http.MethodGet {
			if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
				writeAdminError(w, http.StatusForbidden, "missing required header")
				return
			}
		}
		next(w, r)
	}
}

// handleListRules returns budget rules from runtime config (DB overlay) or YAML default.
func (h *Handler) handleListRules(w http.ResponseWriter, r *http.Request) {
	var rules []budget.Rule
	if err := h.store.GetJSON(r.Context(), "budget_rules", &rules); err != nil {
		h.logger.Error("listing budget rules", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeAdminJSON(w, rules)
}

// handleCreateRule adds a budget rule to the runtime config.
func (h *Handler) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule budget.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid rule: "+err.Error())
		return
	}

	// Load existing rules.
	var rules []budget.Rule //nolint:prealloc
	_ = h.store.GetJSON(r.Context(), "budget_rules", &rules)
	rules = append(rules, rule)

	if err := h.store.SetJSON(r.Context(), "budget_rules", rules); err != nil {
		h.logger.Error("saving budget rules", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Hot-reload into budget manager.
	if h.budgetMgr != nil {
		h.budgetMgr.UpdateRules(rules)
	}

	w.WriteHeader(http.StatusCreated)
	writeAdminJSON(w, rule)
}

// handleDeleteRule removes a budget rule by API key pattern.
func (h *Handler) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeAdminError(w, http.StatusBadRequest, "pattern query parameter required")
		return
	}

	var rules []budget.Rule
	_ = h.store.GetJSON(r.Context(), "budget_rules", &rules)

	var filtered []budget.Rule
	found := false
	for _, rule := range rules {
		if rule.APIKeyPattern == pattern {
			found = true
			continue
		}
		filtered = append(filtered, rule)
	}

	if !found {
		writeAdminError(w, http.StatusNotFound, "rule not found")
		return
	}

	if err := h.store.SetJSON(r.Context(), "budget_rules", filtered); err != nil {
		h.logger.Error("deleting budget rule", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if h.budgetMgr != nil {
		h.budgetMgr.UpdateRules(filtered)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleBlockKey adds an API key pattern to the blocklist.
func (h *Handler) handleBlockKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pattern string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Pattern == "" {
		writeAdminError(w, http.StatusBadRequest, "pattern is required")
		return
	}

	var patterns []string //nolint:prealloc
	_ = h.store.GetJSON(r.Context(), "blocked_keys", &patterns)
	patterns = append(patterns, req.Pattern)

	if err := h.store.SetJSON(r.Context(), "blocked_keys", patterns); err != nil {
		h.logger.Error("blocking API key", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if h.blocklist != nil {
		h.blocklist.Refresh()
	}

	w.WriteHeader(http.StatusCreated)
	writeAdminJSON(w, map[string]string{"pattern": req.Pattern})
}

// handleUnblockKey removes an API key pattern from the blocklist.
func (h *Handler) handleUnblockKey(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		writeAdminError(w, http.StatusBadRequest, "pattern query parameter required")
		return
	}

	var patterns []string
	_ = h.store.GetJSON(r.Context(), "blocked_keys", &patterns)

	var filtered []string
	found := false
	for _, p := range patterns {
		if p == pattern {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}

	if !found {
		writeAdminError(w, http.StatusNotFound, "pattern not found")
		return
	}

	if err := h.store.SetJSON(r.Context(), "blocked_keys", filtered); err != nil {
		h.logger.Error("unblocking API key", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if h.blocklist != nil {
		h.blocklist.Refresh()
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListBlocked returns the list of blocked API key patterns.
func (h *Handler) handleListBlocked(w http.ResponseWriter, r *http.Request) {
	var patterns []string
	if err := h.store.GetJSON(r.Context(), "blocked_keys", &patterns); err != nil {
		h.logger.Error("listing blocked keys", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if patterns == nil {
		patterns = []string{}
	}
	writeAdminJSON(w, patterns)
}

// handleListAPIKeys returns known API key hashes with their spend.
func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	entries, err := h.ledger.QueryCosts(r.Context(), ledger.CostFilter{
		Since:   monthStart,
		Until:   now,
		GroupBy: "key",
	})
	if err != nil {
		h.logger.Error("listing API keys", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	type apiKeyEntry struct {
		APIKeyHash   string  `json:"api_key_hash"`
		Requests     int     `json:"requests"`
		TotalCostUSD float64 `json:"total_cost_usd"`
	}

	var result []apiKeyEntry
	for _, e := range entries {
		result = append(result, apiKeyEntry{
			APIKeyHash:   e.APIKeyHash,
			Requests:     e.Requests,
			TotalCostUSD: e.TotalCostUSD,
		})
	}

	writeAdminJSON(w, result)
}

// handleListProviders returns the status of configured providers.
func (h *Handler) handleListProviders(w http.ResponseWriter, r *http.Request) {
	// Return from runtime config if available.
	var providers map[string]bool
	if err := h.store.GetJSON(r.Context(), "providers_enabled", &providers); err != nil {
		h.logger.Error("listing providers", "error", err)
		writeAdminError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if providers == nil {
		providers = make(map[string]bool)
	}
	writeAdminJSON(w, providers)
}

func writeAdminJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func writeAdminError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
