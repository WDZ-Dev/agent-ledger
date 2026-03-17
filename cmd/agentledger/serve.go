package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/WDZ-Dev/agent-ledger/internal/admin"
	"github.com/WDZ-Dev/agent-ledger/internal/agent"
	"github.com/WDZ-Dev/agent-ledger/internal/alert"
	"github.com/WDZ-Dev/agent-ledger/internal/budget"
	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/dashboard"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/mcp"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	appmetrics "github.com/WDZ-Dev/agent-ledger/internal/otel"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
	"github.com/WDZ-Dev/agent-ledger/internal/proxy"
	"github.com/WDZ-Dev/agent-ledger/internal/ratelimit"
	"github.com/WDZ-Dev/agent-ledger/internal/tenant"
)

func serveCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the AgentLedger proxy",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe(configPath)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")

	return cmd
}

func runServe(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	logger := newLogger(cfg.Log)

	// Storage — switch on driver type.
	// Both backends implement Ledger + agent.SessionStore + DB().
	var store interface {
		ledger.Ledger
		agent.SessionStore
		DB() *sql.DB
	}
	switch cfg.Storage.Driver {
	case "postgres":
		pgStore, pgErr := ledger.NewPostgres(cfg.Storage.DSN, cfg.Storage.MaxOpenConns, cfg.Storage.MaxIdleConns)
		if pgErr != nil {
			return pgErr
		}
		store = pgStore
		logger.Info("storage backend: postgres")
	default:
		sqliteStore, sqliteErr := ledger.NewSQLite(cfg.Storage.DSN)
		if sqliteErr != nil {
			return sqliteErr
		}
		store = sqliteStore
		logger.Info("storage backend: sqlite")
	}
	defer func() { _ = store.Close() }()

	// Async recorder
	rec := ledger.NewRecorder(store, cfg.Recording.BufferSize, cfg.Recording.Workers, logger)
	defer rec.Close()

	// Provider registry
	reg := provider.NewRegistry(cfg.Providers)

	// Cost meter
	m := meter.New()

	// Budget manager
	var budgetMgr *budget.Manager
	budgetCfg := budget.Config{
		Default: budget.Rule{
			APIKeyPattern:   cfg.Budgets.Default.APIKeyPattern,
			DailyLimitUSD:   cfg.Budgets.Default.DailyLimitUSD,
			MonthlyLimitUSD: cfg.Budgets.Default.MonthlyLimitUSD,
			SoftLimitPct:    cfg.Budgets.Default.SoftLimitPct,
			Action:          cfg.Budgets.Default.Action,
		},
	}
	for _, r := range cfg.Budgets.Rules {
		budgetCfg.Rules = append(budgetCfg.Rules, budget.Rule{
			APIKeyPattern:   r.APIKeyPattern,
			DailyLimitUSD:   r.DailyLimitUSD,
			MonthlyLimitUSD: r.MonthlyLimitUSD,
			SoftLimitPct:    r.SoftLimitPct,
			Action:          r.Action,
		})
	}
	budgetMgr = budget.NewManager(store, budgetCfg, logger)
	if budgetMgr.Enabled() {
		logger.Info("budget enforcement enabled")
	}

	// Circuit breaker transport (optional)
	var transport http.RoundTripper
	if cfg.CircuitBreaker.MaxFailures > 0 {
		base := &http.Transport{
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
		}
		timeout := time.Duration(cfg.CircuitBreaker.TimeoutSecs) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		transport = budget.NewBreakerTransport(base, int64(cfg.CircuitBreaker.MaxFailures), timeout)
		logger.Info("circuit breaker enabled",
			"max_failures", cfg.CircuitBreaker.MaxFailures,
			"timeout_secs", cfg.CircuitBreaker.TimeoutSecs,
		)
	}

	// OpenTelemetry + Prometheus metrics
	metrics, metricsHandler, metricsShutdown, err := appmetrics.SetupPrometheus()
	if err != nil {
		return fmt.Errorf("setting up metrics: %w", err)
	}
	defer metricsShutdown()
	logger.Info("prometheus metrics enabled at /metrics")

	// Agent session tracker
	agentCfg := agent.Config{
		SessionTimeoutMins: cfg.Agent.SessionTimeoutMins,
		LoopThreshold:      cfg.Agent.LoopThreshold,
		LoopWindowMins:     cfg.Agent.LoopWindowMins,
		LoopAction:         cfg.Agent.LoopAction,
		GhostMaxAgeMins:    cfg.Agent.GhostMaxAgeMins,
		GhostMinCalls:      cfg.Agent.GhostMinCalls,
		GhostMinCostUSD:    cfg.Agent.GhostMinCostUSD,
	}
	tracker := agent.NewTracker(store, agentCfg, metrics, logger)
	defer tracker.Close()
	if tracker.Enabled() {
		logger.Info("agent session tracking enabled",
			"loop_threshold", agentCfg.LoopThreshold,
			"ghost_max_age_mins", agentCfg.GhostMaxAgeMins,
		)
	}

	// Alerting (optional).
	var notifier alert.Notifier
	{
		var notifiers []alert.Notifier
		if cfg.Alerts.Slack.WebhookURL != "" {
			notifiers = append(notifiers, alert.NewSlackNotifier(cfg.Alerts.Slack.WebhookURL))
			logger.Info("slack alerting enabled")
		}
		for _, wh := range cfg.Alerts.Webhooks {
			notifiers = append(notifiers, alert.NewWebhookNotifier(wh.URL, wh.Headers))
			logger.Info("webhook alerting enabled", "url", wh.URL)
		}
		if len(notifiers) > 0 {
			cooldown := 5 * time.Minute
			if cfg.Alerts.CooldownMin > 0 {
				cooldown = time.Duration(cfg.Alerts.CooldownMin) * time.Minute
			}
			notifier = alert.NewRateLimitedNotifier(
				alert.NewMultiNotifier(notifiers...),
				cooldown,
			)
		}
	}

	// Wire alerting into budget manager.
	if notifier != nil && budgetMgr.Enabled() {
		budgetMgr.SetCallbacks(
			func(ctx context.Context, apiKeyHash string, result budget.Result) {
				_ = notifier.Notify(ctx, alert.Alert{
					Type:     "budget_warning",
					Severity: "warning",
					Message:  fmt.Sprintf("API key %s approaching budget limit", apiKeyHash),
					Details: map[string]string{
						"api_key_hash": apiKeyHash,
						"daily_spent":  fmt.Sprintf("%.2f", result.DailySpent),
						"daily_limit":  fmt.Sprintf("%.2f", result.DailyLimit),
					},
				})
			},
			func(ctx context.Context, apiKeyHash string, result budget.Result) {
				_ = notifier.Notify(ctx, alert.Alert{
					Type:     "budget_exceeded",
					Severity: "critical",
					Message:  fmt.Sprintf("API key %s exceeded budget limit", apiKeyHash),
					Details: map[string]string{
						"api_key_hash": apiKeyHash,
						"daily_spent":  fmt.Sprintf("%.2f", result.DailySpent),
						"daily_limit":  fmt.Sprintf("%.2f", result.DailyLimit),
					},
				})
			},
		)
	}

	// Wire alerting into agent tracker.
	if notifier != nil && tracker.Enabled() {
		tracker.SetAlertNotifier(func(ctx context.Context, agentAlert agent.Alert) {
			severity := "warning"
			if agentAlert.Type == "ghost_detected" {
				severity = "critical"
			}
			_ = notifier.Notify(ctx, alert.Alert{
				Type:     agentAlert.Type,
				Severity: severity,
				Message:  agentAlert.Message,
				Details: map[string]string{
					"session_id": agentAlert.SessionID,
					"agent_id":   agentAlert.AgentID,
				},
			})
		})
	}

	// Tenant resolver (optional).
	var tenantResolver tenant.Resolver
	if cfg.Tenants.Enabled {
		var resolvers []tenant.Resolver
		resolvers = append(resolvers, &tenant.HeaderResolver{})
		if len(cfg.Tenants.KeyMappings) > 0 {
			var mappings []tenant.KeyMapping
			for _, km := range cfg.Tenants.KeyMappings {
				mappings = append(mappings, tenant.KeyMapping{
					APIKeyPattern: km.APIKeyPattern,
					TenantID:      km.TenantID,
				})
			}
			resolvers = append(resolvers, tenant.NewConfigResolver(mappings))
		}
		tenantResolver = tenant.NewChainResolver(resolvers...)
		logger.Info("multi-tenancy enabled")
	}

	// Rate limiter (optional).
	var limiter *ratelimit.Limiter
	{
		rlCfg := ratelimit.Config{
			Default: ratelimit.Rule{
				RequestsPerMinute: cfg.RateLimits.Default.RequestsPerMinute,
				RequestsPerHour:   cfg.RateLimits.Default.RequestsPerHour,
			},
		}
		for _, r := range cfg.RateLimits.Rules {
			rlCfg.Rules = append(rlCfg.Rules, ratelimit.Rule{
				APIKeyPattern:     r.APIKeyPattern,
				RequestsPerMinute: r.RequestsPerMinute,
				RequestsPerHour:   r.RequestsPerHour,
			})
		}
		limiter = ratelimit.New(rlCfg)
		if limiter.Enabled() {
			logger.Info("rate limiting enabled",
				"default_rpm", rlCfg.Default.RequestsPerMinute,
				"default_rph", rlCfg.Default.RequestsPerHour,
			)
		}
	}

	// Proxy
	p := proxy.New(reg, m, rec, budgetMgr, tracker, metrics, limiter, tenantResolver, transport, logger)

	// HTTP routing:
	//   /v1/*         → LLM proxy
	//   /health       → Health check
	//   /metrics      → Prometheus
	//   /api/dashboard/* → Dashboard REST API (if enabled)
	//   /*            → Dashboard static UI (if enabled)
	mux := http.NewServeMux()
	mux.Handle("/v1/", p)
	mux.Handle("/health", p)
	mux.Handle("/metrics", metricsHandler)

	// Register dynamic provider path prefixes (e.g., /groq/, /gemini/).
	for _, prefix := range reg.PathPrefixes() {
		mux.Handle(prefix+"/", p)
		logger.Info("registered provider route", "prefix", prefix)
	}

	// MCP HTTP proxy (optional).
	if cfg.MCP.Enabled && cfg.MCP.Upstream != "" {
		var mcpRules []mcp.PricingRule
		for _, r := range cfg.MCP.Pricing {
			mcpRules = append(mcpRules, mcp.PricingRule{
				Server:      r.Server,
				Tool:        r.Tool,
				CostPerCall: r.CostPerCall,
			})
		}
		mcpPricer := mcp.NewPricer(mcpRules)
		mcpProxy := mcp.NewHTTPProxy(cfg.MCP.Upstream, mcpPricer, rec, logger)
		mux.Handle("/mcp/", mcpProxy)
		logger.Info("MCP HTTP proxy enabled", "upstream", cfg.MCP.Upstream)
	}

	// Admin API (optional).
	if cfg.Admin.Enabled && cfg.Admin.Token != "" {
		adminStore := admin.NewStore(store.DB())
		adminHandler := admin.NewHandler(adminStore, store, budgetMgr, cfg.Admin.Token)
		adminHandler.RegisterRoutes(mux)
		logger.Info("admin API enabled")
	}

	if cfg.Dashboard.Enabled {
		dashHandler := dashboard.NewHandler(store, tracker)
		dashHandler.RegisterRoutes(mux)
		mux.Handle("/", dashboard.StaticHandler())
		logger.Info("dashboard enabled")
	}

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Info("proxy listening", "addr", cfg.Listen)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		return err
	}

	logger.Info("proxy stopped")
	return nil
}

func newLogger(cfg config.LogConfig) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
