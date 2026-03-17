package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

const (
	groupByAgent   = "agent"
	groupBySession = "session"
	displayNone    = "(none)"
)

func orNone(s string) string {
	if s == "" {
		return displayNone
	}
	return s
}

func costsCmd() *cobra.Command {
	var (
		configPath string
		last       string
		groupBy    string
		tenant     string
	)

	cmd := &cobra.Command{
		Use:   "costs",
		Short: "Show cost report",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCosts(configPath, last, groupBy, tenant)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	cmd.Flags().StringVar(&last, "last", "24h", "time window (e.g., 1h, 24h, 7d)")
	cmd.Flags().StringVar(&groupBy, "by", "model", "group by: model, provider, key, agent, session")
	cmd.Flags().StringVar(&tenant, "tenant", "", "filter by tenant ID")

	return cmd
}

func runCosts(configPath, last, groupBy, tenant string) error {
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

	if len(entries) == 0 {
		fmt.Println("No usage recorded in the given time window.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	switch groupBy {
	case groupByAgent:
		_, _ = fmt.Fprintln(w, "AGENT\tREQUESTS\tINPUT TOKENS\tOUTPUT TOKENS\tCOST (USD)")
		_, _ = fmt.Fprintln(w, "-----\t--------\t------------\t-------------\t----------")
	case groupBySession:
		_, _ = fmt.Fprintln(w, "AGENT\tSESSION\tREQUESTS\tINPUT TOKENS\tOUTPUT TOKENS\tCOST (USD)")
		_, _ = fmt.Fprintln(w, "-----\t-------\t--------\t------------\t-------------\t----------")
	default:
		_, _ = fmt.Fprintln(w, "PROVIDER\tMODEL\tREQUESTS\tINPUT TOKENS\tOUTPUT TOKENS\tCOST (USD)")
		_, _ = fmt.Fprintln(w, "--------\t-----\t--------\t------------\t-------------\t----------")
	}

	var totalReqs int
	var totalIn, totalOut int64
	var totalCost float64

	for _, e := range entries {
		switch groupBy {
		case groupByAgent:
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t$%.4f\n",
				orNone(e.AgentID), e.Requests,
				e.InputTokens, e.OutputTokens, e.TotalCostUSD)
		case groupBySession:
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t$%.4f\n",
				orNone(e.AgentID), orNone(e.SessionID), e.Requests,
				e.InputTokens, e.OutputTokens, e.TotalCostUSD)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t$%.4f\n",
				e.Provider, e.Model, e.Requests,
				e.InputTokens, e.OutputTokens, e.TotalCostUSD)
		}
		totalReqs += e.Requests
		totalIn += e.InputTokens
		totalOut += e.OutputTokens
		totalCost += e.TotalCostUSD
	}

	switch groupBy {
	case groupByAgent:
		_, _ = fmt.Fprintln(w, "-----\t--------\t------------\t-------------\t----------")
		_, _ = fmt.Fprintf(w, "TOTAL\t%d\t%d\t%d\t$%.4f\n",
			totalReqs, totalIn, totalOut, totalCost)
	case groupBySession:
		_, _ = fmt.Fprintln(w, "-----\t-------\t--------\t------------\t-------------\t----------")
		_, _ = fmt.Fprintf(w, "TOTAL\t\t%d\t%d\t%d\t$%.4f\n",
			totalReqs, totalIn, totalOut, totalCost)
	default:
		_, _ = fmt.Fprintln(w, "--------\t-----\t--------\t------------\t-------------\t----------")
		_, _ = fmt.Fprintf(w, "TOTAL\t\t%d\t%d\t%d\t$%.4f\n",
			totalReqs, totalIn, totalOut, totalCost)
	}
	return w.Flush()
}

// parseDuration handles time windows like "1h", "24h", "7d", "30d".
func parseDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 24 * time.Hour, nil
	}

	// Handle "d" suffix for days.
	if s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}
