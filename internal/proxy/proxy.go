package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
	"github.com/WDZ-Dev/agent-ledger/internal/budget"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	appmetrics "github.com/WDZ-Dev/agent-ledger/internal/otel"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
)

// context keys for passing data between Rewrite and ModifyResponse.
type ctxKey int

const (
	ctxProvider ctxKey = iota
	ctxRequestBody
	ctxRequestMeta
	ctxStartTime
	ctxAPIKeyHash
	ctxAgentID
	ctxSessionID
	ctxUserID
	ctxBudgetResult
	ctxTask
	ctxSessionEnd
)

// Proxy is the core reverse proxy that intercepts LLM API calls,
// meters token usage, and records costs.
type Proxy struct {
	rp       *httputil.ReverseProxy
	registry *provider.Registry
	meter    *meter.Meter
	recorder *ledger.Recorder
	budget   *budget.Manager
	tracker  *agent.Tracker
	metrics  *appmetrics.Metrics
	logger   *slog.Logger
}

// New creates a Proxy wired to the given registry, meter, recorder, and
// optional budget manager, agent tracker, metrics, and transport.
// Pass nil for budgetMgr/tracker/metrics to disable those features.
// Pass nil for transport to use the default pooled transport.
func New(registry *provider.Registry, m *meter.Meter, recorder *ledger.Recorder, budgetMgr *budget.Manager, tracker *agent.Tracker, metrics *appmetrics.Metrics, transport http.RoundTripper, logger *slog.Logger) *Proxy {
	p := &Proxy{
		registry: registry,
		meter:    m,
		recorder: recorder,
		budget:   budgetMgr,
		tracker:  tracker,
		metrics:  metrics,
		logger:   logger,
	}

	if transport == nil {
		transport = &http.Transport{
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
		}
	}

	p.rp = &httputil.ReverseProxy{
		Rewrite:        p.rewrite,
		ModifyResponse: p.modifyResponse,
		ErrorHandler:   p.errorHandler,
		FlushInterval:  -1, // flush immediately for SSE
		Transport:      transport,
	}

	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check — not proxied.
	if r.URL.Path == "/health" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
		return
	}

	prov := p.registry.Detect(r)
	if prov == nil {
		p.logger.Warn("no provider matched", "path", r.URL.Path)
		writeJSONError(w, http.StatusBadGateway, "no provider matched for this request")
		return
	}

	// Read request body for metadata extraction.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.logger.Error("reading request body", "error", err)
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	_ = r.Body.Close()

	// Parse request metadata.
	reqMeta, _ := prov.ParseRequest(body)

	// Extract API key fingerprint.
	apiKey := provider.ExtractAPIKey(r)
	apiKeyHash := provider.HashAPIKey(apiKey)

	// Extract agent headers before stripping.
	agentID, sessionID, userID, task := provider.ExtractAgentHeaders(r)
	sessionEnd := r.Header.Get("X-Agent-Session-End") == "true"

	// Agent session tracking.
	if p.tracker != nil && p.tracker.Enabled() && sessionID != "" {
		if sessionEnd {
			p.tracker.EndSession(sessionID)
		} else {
			model := ""
			if reqMeta != nil {
				model = reqMeta.Model
			}
			alert := p.tracker.TrackCall(sessionID, agentID, userID, task, model, r.URL.Path)
			if alert != nil && alert.Type == "loop_detected" && p.tracker.ShouldBlock() {
				writeJSONError(w, http.StatusTooManyRequests, "loop detected: "+alert.Message)
				return
			}
		}
	}

	// Budget check: reject or warn before forwarding.
	var budgetResult *budget.Result
	if p.budget != nil && p.budget.Enabled() {
		br := p.budget.Check(r.Context(), apiKey, apiKeyHash)
		budgetResult = &br
		if br.Decision == budget.Block {
			p.logger.Warn("budget exceeded",
				"api_key_hash", apiKeyHash,
				"daily_spent", br.DailySpent,
				"daily_limit", br.DailyLimit,
				"monthly_spent", br.MonthlySpent,
				"monthly_limit", br.MonthlyLimit,
			)
			writeBudgetError(w, br)
			return
		}

		// Pre-flight: estimate worst-case cost from max_tokens and reject
		// if it would exceed remaining budget.
		if reqMeta != nil && reqMeta.MaxTokens > 0 {
			worstCase := p.meter.Calculate(reqMeta.Model, 0, reqMeta.MaxTokens)
			if worstCase > 0 {
				dailyRemaining := br.DailyLimit - br.DailySpent
				monthlyRemaining := br.MonthlyLimit - br.MonthlySpent
				if (br.DailyLimit > 0 && worstCase > dailyRemaining) ||
					(br.MonthlyLimit > 0 && worstCase > monthlyRemaining) {
					p.logger.Warn("pre-flight budget rejection",
						"api_key_hash", apiKeyHash,
						"model", reqMeta.Model,
						"max_tokens", reqMeta.MaxTokens,
						"estimated_cost", worstCase,
					)
					writePreflightError(w, br, worstCase)
					return
				}
			}
		}
	}

	// Store everything in context for ModifyResponse.
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxProvider, prov)
	ctx = context.WithValue(ctx, ctxRequestBody, body)
	ctx = context.WithValue(ctx, ctxRequestMeta, reqMeta)
	ctx = context.WithValue(ctx, ctxStartTime, time.Now())
	ctx = context.WithValue(ctx, ctxAPIKeyHash, apiKeyHash)
	ctx = context.WithValue(ctx, ctxAgentID, agentID)
	ctx = context.WithValue(ctx, ctxSessionID, sessionID)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxBudgetResult, budgetResult)
	ctx = context.WithValue(ctx, ctxTask, task)
	ctx = context.WithValue(ctx, ctxSessionEnd, sessionEnd)

	r = r.WithContext(ctx)
	r.Body = io.NopCloser(bytes.NewReader(body))

	p.rp.ServeHTTP(w, r)
}

func (p *Proxy) rewrite(pr *httputil.ProxyRequest) {
	prov, _ := pr.In.Context().Value(ctxProvider).(provider.Provider)
	if prov == nil {
		return
	}

	upstream, err := url.Parse(prov.UpstreamURL())
	if err != nil {
		p.logger.Error("parsing upstream URL", "error", err, "url", prov.UpstreamURL())
		return
	}

	pr.SetURL(upstream)
	pr.Out.Host = upstream.Host

	// Strip agent headers so they don't leak to the provider.
	provider.StripAgentHeaders(pr.Out)

	// Ensure we get uncompressed responses for parsing.
	pr.Out.Header.Del("Accept-Encoding")
}

func (p *Proxy) modifyResponse(resp *http.Response) error {
	ctx := resp.Request.Context()
	prov, _ := ctx.Value(ctxProvider).(provider.Provider)
	reqMeta, _ := ctx.Value(ctxRequestMeta).(*provider.RequestMeta)
	start, _ := ctx.Value(ctxStartTime).(time.Time)
	apiKeyHash, _ := ctx.Value(ctxAPIKeyHash).(string)
	agentID, _ := ctx.Value(ctxAgentID).(string)
	sessionID, _ := ctx.Value(ctxSessionID).(string)
	userID, _ := ctx.Value(ctxUserID).(string)

	if prov == nil {
		return nil
	}

	// Add budget warning header if nearing limit.
	if br, ok := ctx.Value(ctxBudgetResult).(*budget.Result); ok && br != nil && br.Decision == budget.Warn {
		resp.Header.Set("X-AgentLedger-Budget-Warning",
			fmt.Sprintf("daily=%.2f/%.2f monthly=%.2f/%.2f",
				br.DailySpent, br.DailyLimit, br.MonthlySpent, br.MonthlyLimit))
	}

	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStream {
		resp.Body = newStreamInterceptor(
			resp.Body, prov, reqMeta, p.meter, p.recorder, p.tracker, p.metrics, p.logger,
			start, apiKeyHash, resp.Request.URL.Path,
			agentID, sessionID, userID,
		)
		return nil
	}

	// Non-streaming: read, parse, record, replace body.
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		p.logger.Error("reading response body", "error", err)
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Only meter successful responses.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	respMeta, err := prov.ParseResponse(body)
	if err != nil {
		p.logger.Debug("parsing response", "error", err)
		return nil
	}

	model := respMeta.Model
	if model == "" && reqMeta != nil {
		model = reqMeta.Model
	}

	cost := p.meter.Calculate(model, respMeta.InputTokens, respMeta.OutputTokens)
	estimated := !p.meter.KnownModel(model)

	record := &ledger.UsageRecord{
		ID:           ulid.Make().String(),
		Timestamp:    start,
		Provider:     prov.Name(),
		Model:        model,
		APIKeyHash:   apiKeyHash,
		InputTokens:  respMeta.InputTokens,
		OutputTokens: respMeta.OutputTokens,
		TotalTokens:  respMeta.TotalTokens,
		CostUSD:      cost,
		Estimated:    estimated,
		DurationMS:   time.Since(start).Milliseconds(),
		StatusCode:   resp.StatusCode,
		Path:         resp.Request.URL.Path,
		AgentID:      agentID,
		SessionID:    sessionID,
		UserID:       userID,
	}
	p.recorder.Record(record)

	// Update agent session cost.
	if p.tracker != nil && sessionID != "" {
		p.tracker.RecordCost(sessionID, cost, respMeta.TotalTokens)
	}

	// Update OTel metrics.
	if p.metrics != nil {
		p.metrics.RecordRequest(prov.Name(), model, resp.StatusCode,
			float64(record.DurationMS), respMeta.InputTokens, respMeta.OutputTokens,
			cost, false, apiKeyHash)
	}

	p.logger.Info("request",
		"provider", prov.Name(),
		"model", model,
		"input_tokens", respMeta.InputTokens,
		"output_tokens", respMeta.OutputTokens,
		"cost_usd", fmt.Sprintf("%.6f", cost),
		"duration_ms", record.DurationMS,
	)

	return nil
}

func (p *Proxy) errorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	p.logger.Error("proxy error", "error", err)
	writeJSONError(w, http.StatusBadGateway, "upstream request failed: "+err.Error())
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"type":    "proxy_error",
			"message": msg,
		},
	})
}

func writePreflightError(w http.ResponseWriter, br budget.Result, estimatedCost float64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"type":           "budget_exceeded",
			"message":        "estimated cost of request exceeds remaining budget",
			"estimated_cost": estimatedCost,
			"daily_spent":    br.DailySpent,
			"daily_limit":    br.DailyLimit,
			"monthly_spent":  br.MonthlySpent,
			"monthly_limit":  br.MonthlyLimit,
		},
	})
}

func writeBudgetError(w http.ResponseWriter, br budget.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"type":          "budget_exceeded",
			"message":       "spending limit exceeded",
			"daily_spent":   br.DailySpent,
			"daily_limit":   br.DailyLimit,
			"monthly_spent": br.MonthlySpent,
			"monthly_limit": br.MonthlyLimit,
		},
	})
}
