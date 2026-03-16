package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/mcp"
)

func mcpWrapCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "mcp-wrap [flags] -- command [args...]",
		Short: "Wrap an MCP server process for tool call metering",
		Long: `mcp-wrap starts an MCP server as a child process and transparently
intercepts JSON-RPC messages on stdin/stdout to record tool call usage.

Agent context is read from environment variables:
  AGENTLEDGER_AGENT_ID      — agent identifier
  AGENTLEDGER_SESSION_ID    — session/execution identifier
  AGENTLEDGER_USER_ID       — user who triggered the agent
  AGENTLEDGER_TASK          — human-readable task description`,
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: false,
		RunE: func(_ *cobra.Command, args []string) error {
			code, err := runMCPWrap(configPath, args[0], args[1:])
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")

	return cmd
}

func runMCPWrap(configPath string, command string, args []string) (int, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return 1, fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg.Log)

	// Open storage directly — no proxy server needed.
	store, err := ledger.NewSQLite(cfg.Storage.DSN)
	if err != nil {
		return 1, fmt.Errorf("opening storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Small buffer and single worker for stdio mode.
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	// Read agent context from environment variables.
	agentID := os.Getenv("AGENTLEDGER_AGENT_ID")
	sessionID := os.Getenv("AGENTLEDGER_SESSION_ID")
	userID := os.Getenv("AGENTLEDGER_USER_ID")

	// Build pricing rules from config.
	var rules []mcp.PricingRule
	for _, r := range cfg.MCP.Pricing {
		rules = append(rules, mcp.PricingRule{
			Server:      r.Server,
			Tool:        r.Tool,
			CostPerCall: r.CostPerCall,
		})
	}
	pricer := mcp.NewPricer(rules)

	interceptor := mcp.NewInterceptor("unknown", pricer, rec, agentID, sessionID, userID, logger)
	wrapper := mcp.NewStdioWrapper(interceptor, logger)

	logger.Info("mcp-wrap starting",
		"command", command,
		"args", args,
		"agent_id", agentID,
		"session_id", sessionID,
	)

	code, err := wrapper.Run(context.Background(), command, args)
	if err != nil {
		logger.Error("mcp-wrap error", "error", err)
		return 1, err
	}

	if dropped := rec.Dropped(); dropped > 0 {
		logger.Warn("mcp records dropped", "count", dropped)
	}

	return code, nil
}
