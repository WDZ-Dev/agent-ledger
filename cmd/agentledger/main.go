package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "agentledger",
		Short: "Know what your agents cost",
		Long:  "AgentLedger is a reverse proxy that provides real-time cost attribution, budget enforcement, and financial observability for AI agents.",
	}

	root.AddCommand(serveCmd())
	root.AddCommand(costsCmd())
	root.AddCommand(exportCmd())
	root.AddCommand(mcpWrapCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newHealthcheckCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("agentledger %s\n", version)
		},
	}
}

func newHealthcheckCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:    "healthcheck",
		Short:  "Check if the server is healthy",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := http.Get("http://" + addr + "/health")
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
			}
			fmt.Println("ok")
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:8787", "server address to check")

	return cmd
}
