package main

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/mcphealth"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/output"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "Inspect MCP servers configured in AI clients"}

	var checkCredentials bool
	var files []string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered MCP servers",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			servers, notes := mcpcfg.Discover(mcpcfg.DefaultSources())
			if len(files) > 0 {
				fileServers, fileNotes := mcpcfg.DiscoverFiles(files)
				servers = append(servers, fileServers...)
				notes = append(notes, fileNotes...)
			}
			if checkCredentials {
				for i := range servers {
					servers[i].CredentialWarnings = mcpcfg.CheckCredentials(servers[i])
				}
			}
			if len(notes) > 0 {
				for _, n := range notes {
					slog.Warn(n)
				}
			}
			return out.Print(servers)
		},
	}
	listCmd.Flags().BoolVar(&checkCredentials, "check-credentials", false, "Flag env values that look like exposed credentials")
	listCmd.Flags().StringSliceVar(&files, "files", nil, "Additional config file(s) to parse; output is additive to default discovery")
	cmd.AddCommand(listCmd)

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
			if addCommand == "" && addURL == "" {
				return exitcodes.Wrap(fmt.Errorf("at least one of --command or --url is required"), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp add")
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
			out := output.New("json")
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
				return out.Print(results)
			}
			return out.Print(mcphealth.ProbeAll(servers))
		},
	}
	health.Flags().BoolVar(&probe, "probe", false, "actually probe each server (WARNING: executes commands from MCP config files)")
	cmd.AddCommand(health)

	return cmd
}
