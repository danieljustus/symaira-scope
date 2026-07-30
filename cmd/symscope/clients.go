package main

import (
	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/output"
)

func newClientsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "clients", Short: "AI client configuration status"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List known AI clients and whether their MCP config is present",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			return out.Print(mcpcfg.FoundClients(mcpcfg.DefaultSources()))
		},
	})
	return cmd
}
