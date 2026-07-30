package main

import (
	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-scope/internal/mcptools"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "serve",
		Aliases:      []string{"mcp-serve"},
		Short:        "Start the MCP stdio server for AI agents",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return mcptools.Serve(version)
		},
	}
}
