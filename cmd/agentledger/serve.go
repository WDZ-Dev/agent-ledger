package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/WDZ-Dev/agent-ledger/internal/agent"
	"github.com/WDZ-Dev/agent-ledger/internal/budget"
	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
	"github.com/WDZ-Dev/agent-ledger/internal/proxy"
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

	// Storage
	store, err := ledger.NewSQLite(cfg.Storage.DSN)
	if err != nil {
		return err
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
	tracker := agent.NewTracker(store, agentCfg, logger)
	defer tracker.Close()
	if tracker.Enabled() {
		logger.Info("agent session tracking enabled",
			"loop_threshold", agentCfg.LoopThreshold,
			"ghost_max_age_mins", agentCfg.GhostMaxAgeMins,
		)
	}

	// Proxy
	p := proxy.New(reg, m, rec, budgetMgr, tracker, transport, logger)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           p,
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
