// Command symscope inventories local ports, containers, and MCP servers for AI
// development environments — as a CLI and as an MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"
	"github.com/danieljustus/symaira-corekit/logkit"
	"github.com/danieljustus/symaira-corekit/updatecheck"

	"github.com/danieljustus/symaira-scope/internal/containers"
	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/mcptools"
	"github.com/danieljustus/symaira-scope/internal/ports"
	"github.com/danieljustus/symaira-scope/internal/scan"
)

var version = "0.1.0-dev"

func main() {
	slog.SetDefault(logkit.NewFromEnv("symscope"))
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "symscope:", exitcodes.FormatCLIError(err))
		os.Exit(int(exitcodes.ExitCodeFromError(err)))
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
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
		newServeCmd(),
		newVersionCmd(),
	)
	return root
}

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Full inventory snapshot (ports + MCP servers + containers)",
		RunE: func(_ *cobra.Command, _ []string) error {
			snap, err := scan.Build()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "scan")
			}
			return printJSON(snap)
		},
	}
}

func newPortsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ports", Short: "List or suggest local ports"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List local listening ports",
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := ports.ListListening()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "list ports")
			}
			return printJSON(p)
		},
	})

	var count, from, to int
	suggest := &cobra.Command{
		Use:   "suggest",
		Short: "Suggest free TCP ports in a range",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(map[string]any{"free": ports.SuggestFree(count, from, to)})
		},
	}
	suggest.Flags().IntVar(&count, "count", 3, "How many free ports to suggest")
	suggest.Flags().IntVar(&from, "from", 3000, "Range start")
	suggest.Flags().IntVar(&to, "to", 9999, "Range end")
	cmd.AddCommand(suggest)

	return cmd
}

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "Inspect MCP servers configured in AI clients"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List discovered MCP servers",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(mcpcfg.Discover(mcpcfg.DefaultSources()))
		},
	})
	return cmd
}

func newClientsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "clients", Short: "AI client configuration status"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List known AI clients and whether their MCP config is present",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(mcpcfg.FoundClients(mcpcfg.DefaultSources()))
		},
	})
	return cmd
}

func newContainersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "containers",
		Short: "List running containers and published ports",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, notes := containers.List()
			return printJSON(map[string]any{"containers": c, "notes": notes})
		},
	}
}

func newConflictsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conflicts",
		Short: "Report ports bound by more than one process",
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := ports.ListListening()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "list ports")
			}
			return printJSON(ports.Conflicts(p))
		},
	}
}

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

func newVersionCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("symscope", version)
			if !check {
				return nil
			}
			release, err := updatecheck.NewChecker("danieljustus", "symaira-scope").Check(context.Background(), version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "update check failed: %v\n", err)
				return nil
			}
			if release != nil {
				fmt.Printf("Update available: %s\n%s\n", release.TagName, release.HTMLURL)
			} else {
				fmt.Println("Already up to date.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Check for updates on GitHub")
	return cmd
}
