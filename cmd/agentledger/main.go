package main

import (
	"fmt"
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
	root.AddCommand(newVersionCmd())

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
