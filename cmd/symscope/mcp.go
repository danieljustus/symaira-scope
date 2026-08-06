package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

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
	var listFormat string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered MCP servers",
		RunE: func(_ *cobra.Command, _ []string) error {
			switch listFormat {
			case "json", "ndjson", "manifest":
			default:
				return exitcodes.Wrap(fmt.Errorf("unsupported format %q (supported: json, ndjson, manifest)", listFormat), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp list")
			}
			out := output.New(listFormat)
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
			if listFormat == "manifest" {
				return output.New("json").Print(toManifestEntries(servers))
			}
			return out.Print(servers)
		},
	}
	listCmd.Flags().BoolVar(&checkCredentials, "check-credentials", false, "Flag env values that look like exposed credentials")
	listCmd.Flags().StringSliceVar(&files, "files", nil, "Additional config file(s) to parse; output is additive to default discovery")
	listCmd.Flags().StringVar(&listFormat, "format", "json", "Output format: json, ndjson, or manifest (registry server.json-shaped)")
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

	var inspectName, inspectMethod, inspectToolName, inspectToolArgsJSON, inspectFormat string
	var inspectToolArgs []string
	inspect := &cobra.Command{
		Use:   "inspect",
		Short: "Call a JSON-RPC method (tools/list, tools/call) on a discovered MCP server",
		Long: "Call a JSON-RPC method on a discovered MCP server and print the structured response.\n\n" +
			"Supported methods: tools/list, tools/call.\n\n" +
			"WARNING: inspect executes the discovered server's command or URL from its config,\n" +
			"exactly like `mcp health --probe` — only run it against servers you trust.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if inspectName == "" {
				return exitcodes.Wrap(fmt.Errorf("--name is required"), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp inspect")
			}
			if inspectMethod != "tools/list" && inspectMethod != "tools/call" {
				return exitcodes.Wrap(fmt.Errorf("unsupported method %q (supported: tools/list, tools/call)", inspectMethod), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp inspect")
			}
			var params map[string]any
			if inspectMethod == "tools/call" {
				if inspectToolName == "" {
					return exitcodes.Wrap(fmt.Errorf("--tool-name is required for tools/call"), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp inspect")
				}
				arguments, err := parseToolArguments(inspectToolArgs, inspectToolArgsJSON)
				if err != nil {
					return exitcodes.Wrap(err, exitcodes.ExitConfig, exitcodes.KindValidation, "mcp inspect")
				}
				params = map[string]any{"name": inspectToolName, "arguments": arguments}
			}

			servers, notes := mcpcfg.Discover(mcpcfg.DefaultSources())
			if len(files) > 0 {
				fileServers, fileNotes := mcpcfg.DiscoverFiles(files)
				servers = append(servers, fileServers...)
				notes = append(notes, fileNotes...)
			}
			if len(notes) > 0 {
				for _, n := range notes {
					slog.Warn(n)
				}
			}

			var server *model.MCPServer
			for i := range servers {
				if servers[i].Name == inspectName {
					server = &servers[i]
					break
				}
			}
			if server == nil {
				return exitcodes.Wrap(fmt.Errorf("no MCP server named %q found in discovered configs", inspectName), exitcodes.ExitConfig, exitcodes.KindValidation, "mcp inspect")
			}

			slog.Warn(fmt.Sprintf("mcp inspect: calling %s on server %q (this runs the command/URL from its config)", inspectMethod, inspectName))
			result, err := mcphealth.Inspect(*server, inspectMethod, params)
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "mcp inspect")
			}
			result.Name = server.Name
			result.Client = server.Client
			return output.New(inspectFormat).Print(result)
		},
	}
	inspect.Flags().StringVar(&inspectName, "name", "", "Server name (as shown by `mcp list`)")
	inspect.Flags().StringVar(&inspectMethod, "method", "tools/list", "JSON-RPC method to call: tools/list or tools/call")
	inspect.Flags().StringVar(&inspectToolName, "tool-name", "", "Tool to call (required for tools/call)")
	inspect.Flags().StringArrayVar(&inspectToolArgs, "tool-arg", nil, "Tool argument as key=value (repeatable; overrides --tool-args-json)")
	inspect.Flags().StringVar(&inspectToolArgsJSON, "tool-args-json", "", "Tool arguments as a single JSON object")
	inspect.Flags().StringVar(&inspectFormat, "format", "json", "Output format (json)")
	inspect.Flags().StringSliceVar(&files, "files", nil, "Additional config file(s) to parse; additive to default discovery")
	cmd.AddCommand(inspect)

	return cmd
}

// parseToolArguments assembles a tools/call arguments map from repeated
// --tool-arg key=value pairs and/or a --tool-args-json blob. Values that
// parse as JSON literals (numbers, booleans, null) are passed through typed;
// everything else is sent as a string. --tool-arg pairs win over the blob.
func parseToolArguments(kv []string, blob string) (map[string]any, error) {
	arguments := map[string]any{}
	if blob != "" {
		if err := json.Unmarshal([]byte(blob), &arguments); err != nil {
			return nil, fmt.Errorf("--tool-args-json: %w", err)
		}
	}
	for _, pair := range kv {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--tool-arg %q must be key=value", pair)
		}
		var literal any
		if err := json.Unmarshal([]byte(value), &literal); err == nil {
			arguments[key] = literal
		} else {
			arguments[key] = value
		}
	}
	return arguments, nil
}
