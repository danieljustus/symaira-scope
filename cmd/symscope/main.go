// Command symscope inventories local ports, containers, and MCP servers for AI
// development environments — as a CLI and as an MCP server.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"
	"github.com/danieljustus/symaira-corekit/logkit"
)

var version = "0.2.0"

func main() {
	slog.SetDefault(logkit.NewFromEnv("symscope"))
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "symscope:", exitcodes.FormatCLIError(err))
		os.Exit(int(exitcodes.ExitCodeFromError(err)))
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "symscope",
		Short:   "Inventory ports, containers, and MCP servers for AI dev environments",
		Version: version,
		Long: `symscope inventories local listening ports, Docker-published ports, and the
MCP servers configured across your AI clients — from one place, as a CLI and as
an MCP server for agents.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newScanCmd(),
		newPortsCmd(),
		newMCPCmd(),
		newClientsCmd(),
		newContainersCmd(),
		newConflictsCmd(),
		newExplainCmd(),
		newCacheCmd(),
		newServeCmd(),
		newVersionCmd(),
		newWatchCmd(),
	)
	return root
}
