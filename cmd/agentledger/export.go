package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

func exportCmd() *cobra.Command {
	var (
		configPath string
		last       string
		groupBy    string
		tenant     string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export cost data as CSV or JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runExport(configPath, last, groupBy, tenant, format)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	cmd.Flags().StringVar(&last, "last", "30d", "time window (e.g., 1h, 24h, 7d, 30d)")
	cmd.Flags().StringVar(&groupBy, "by", "model", "group by: model, provider, key, agent, session")
	cmd.Flags().StringVar(&tenant, "tenant", "", "filter by tenant ID")
	cmd.Flags().StringVarP(&format, "format", "f", "csv", "output format: csv or json")

	return cmd
}

func runExport(configPath, last, groupBy, tenant, format string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	var store ledger.Ledger
	switch cfg.Storage.Driver {
	case "postgres":
		store, err = ledger.NewPostgres(cfg.Storage.DSN, cfg.Storage.MaxOpenConns, cfg.Storage.MaxIdleConns)
	default:
		store, err = ledger.NewSQLite(cfg.Storage.DSN)
	}
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	window, err := parseDuration(last)
	if err != nil {
		return fmt.Errorf("invalid --last value %q: %w", last, err)
	}

	now := time.Now()
	filter := ledger.CostFilter{
		Since:    now.Add(-window),
		Until:    now,
		GroupBy:  groupBy,
		TenantID: tenant,
	}

	entries, err := store.QueryCosts(context.Background(), filter)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(entries)
	case "csv":
		return writeCSV(os.Stdout, entries)
	default:
		return fmt.Errorf("unsupported format %q: use csv or json", format)
	}
}

func writeCSV(out *os.File, entries []ledger.CostEntry) error {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{
		"provider", "model", "api_key_hash", "agent_id", "session_id",
		"requests", "input_tokens", "output_tokens", "cost_usd",
	}
	if err := w.Write(header); err != nil {
		return err
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
			strconv.FormatFloat(e.TotalCostUSD, 'f', 6, 64),
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}

	return nil
}
