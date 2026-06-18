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

	"github.com/danieljustus/symaira-scope/internal/cache"
	"github.com/danieljustus/symaira-scope/internal/containers"
	"github.com/danieljustus/symaira-scope/internal/explain"
	"github.com/danieljustus/symaira-scope/internal/mcphealth"
	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/mcptools"
	"github.com/danieljustus/symaira-scope/internal/model"
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
		newExplainCmd(),
		newCacheCmd(),
		newServeCmd(),
		newVersionCmd(),
	)
	return root
}

func newScanCmd() *cobra.Command {
	var noCache bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Full inventory snapshot (ports + MCP servers + containers)",
		RunE: func(_ *cobra.Command, _ []string) error {
			if !noCache {
				if snap, err := cache.Load(); err != nil {
					return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "cache load")
				} else if snap != nil {
					return printJSON(snap)
				}
			}

			snap, err := scan.Build()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "scan")
			}

			if !noCache {
				if err := cache.Save(&snap); err != nil {
					slog.Warn("cache save failed", "err", err)
				}
			}

			return printJSON(snap)
		},
	}
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Skip cache; always run a fresh scan")
	return cmd
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
			servers, notes := mcpcfg.Discover(mcpcfg.DefaultSources())
			if len(notes) > 0 {
				for _, n := range notes {
					slog.Warn(n)
				}
			}
			return printJSON(servers)
		},
	})

	var addName, addCommand, addClient, addURL string
	var addArgs []string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add an MCP server to a client config",
		RunE: func(_ *cobra.Command, _ []string) error {
			sources := mcpcfg.DefaultSources()
			var source *mcpcfg.Source
			for _, s := range sources {
				if s.Client == addClient {
					source = &s
					break
				}
			}
			if source == nil {
				return exitcodes.Wrap(fmt.Errorf("unknown client %q", addClient), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp add")
			}
			if err := mcpcfg.AddServer(*source, addName, mcpcfg.Entry{Command: addCommand, Args: addArgs, URL: addURL}); err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "mcp add")
			}
			fmt.Printf("Added %s to %s config.\n", addName, addClient)
			return nil
		},
	}
	addCmd.Flags().StringVar(&addName, "name", "", "Server name")
	addCmd.Flags().StringVar(&addCommand, "command", "", "Command to run")
	addCmd.Flags().StringArrayVar(&addArgs, "args", nil, "Command arguments")
	addCmd.Flags().StringVar(&addURL, "url", "", "HTTP URL (for HTTP transport)")
	addCmd.Flags().StringVar(&addClient, "client", "", "AI client (e.g. claude-desktop, cursor)")
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("client")
	cmd.AddCommand(addCmd)

	var rmName, rmClient string
	rmCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an MCP server from a client config",
		RunE: func(_ *cobra.Command, _ []string) error {
			sources := mcpcfg.DefaultSources()
			var source *mcpcfg.Source
			for _, s := range sources {
				if s.Client == rmClient {
					source = &s
					break
				}
			}
			if source == nil {
				return exitcodes.Wrap(fmt.Errorf("unknown client %q", rmClient), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp remove")
			}
			if err := mcpcfg.RemoveServer(*source, rmName); err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "mcp remove")
			}
			fmt.Printf("Removed %s from %s config.\n", rmName, rmClient)
			return nil
		},
	}
	rmCmd.Flags().StringVar(&rmName, "name", "", "Server name")
	rmCmd.Flags().StringVar(&rmClient, "client", "", "AI client")
	rmCmd.MarkFlagRequired("name")
	rmCmd.MarkFlagRequired("client")
	cmd.AddCommand(rmCmd)

	var probe bool
	health := &cobra.Command{
		Use:   "health",
		Short: "Health-check discovered MCP servers",
		RunE: func(_ *cobra.Command, _ []string) error {
			servers, notes := mcpcfg.Discover(mcpcfg.DefaultSources())
			if !probe {
				results := make([]model.MCPHealthResult, len(servers))
				for i, s := range servers {
					results[i] = model.MCPHealthResult{Name: s.Name, Client: s.Client, Status: "unknown"}
				}
				if len(notes) > 0 {
					for _, n := range notes {
						slog.Warn(n)
					}
				}
				return printJSON(results)
			}
			return printJSON(mcphealth.ProbeAll(servers))
		},
	}
	health.Flags().BoolVar(&probe, "probe", false, "actually probe each server (WARNING: executes commands from MCP config files)")
	cmd.AddCommand(health)

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
		Short: "Report ports bound by more than one process or occupied by configured services",
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := ports.ListListening()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "list ports")
			}
			all := ports.Conflicts(p)
			servers, _ := mcpcfg.Discover(mcpcfg.DefaultSources())
			all = append(all, ports.MCPServerConflicts(servers, p)...)
			return printJSON(all)
		},
	}
}

func newExplainCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "explain", Short: "Explain what uses a port or server"}

	var port int
	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Explain what's using a specific port",
		RunE: func(_ *cobra.Command, _ []string) error {
			exp, err := explain.ExplainPort(port)
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "explain port")
			}
			return printJSON(exp)
		},
	}
	portCmd.Flags().IntVar(&port, "number", 0, "Port number to explain")
	portCmd.MarkFlagRequired("number")
	cmd.AddCommand(portCmd)

	var serverName string
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Explain a specific MCP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			exp, err := explain.ExplainServer(serverName)
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "explain server")
			}
			return printJSON(exp)
		},
	}
	serverCmd.Flags().StringVar(&serverName, "name", "", "Server name to explain")
	serverCmd.MarkFlagRequired("name")
	cmd.AddCommand(serverCmd)

	return cmd
}

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Inspect or manage the snapshot cache"}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show cache status and metadata",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(cache.Stats())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Delete the snapshot cache file",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := cache.Clear(); err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "cache clear")
			}
			fmt.Println("Cache cleared.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stats",
		Short: "Print cache statistics as JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(cache.Stats())
		},
	})

	return cmd
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
