// Package mcptools exposes symscope's inventory over the MCP stdio transport
// using corekit/mcpserver.
package mcptools

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/danieljustus/symaira-corekit/mcpserver"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/mcphealth"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/ports"
	"github.com/danieljustus/symaira-scope/internal/scan"
)

const emptyObject = `{"type":"object","properties":{}}`

// Register adds all symscope tools to the server.
func Register(srv *mcpserver.Server) {
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "scan",
		Description: "Inventory listening ports, discovered MCP servers, and containers in one snapshot.",
		InputSchema: json.RawMessage(emptyObject),
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			return scan.Build()
		},
	})
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "ports_list",
		Description: "List local listening TCP/UDP ports with the owning process.",
		InputSchema: json.RawMessage(emptyObject),
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			return ports.ListListening()
		},
	})
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "ports_suggest",
		Description: "Suggest free TCP ports in a range (defaults: 3 ports, 3000-9999).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"count":{"type":"integer"},"from":{"type":"integer"},"to":{"type":"integer"}}}`),
		Handler: func(_ context.Context, in json.RawMessage) (any, error) {
			args := struct {
				Count int `json:"count"`
				From  int `json:"from"`
				To    int `json:"to"`
			}{Count: 3, From: 3000, To: 9999}
			_ = json.Unmarshal(in, &args)
			return map[string]any{"free": ports.SuggestFree(args.Count, args.From, args.To)}, nil
		},
	})
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "mcp_list",
		Description: "Discover MCP servers configured across local AI clients (Claude, Cursor, VS Code, Windsurf, project).",
		InputSchema: json.RawMessage(emptyObject),
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			servers, _ := mcpcfg.Discover(mcpcfg.DefaultSources())
			return servers, nil
		},
	})
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "conflicts",
		Description: "Report TCP ports bound by more than one process.",
		InputSchema: json.RawMessage(emptyObject),
		Handler: func(_ context.Context, _ json.RawMessage) (any, error) {
			p, err := ports.ListListening()
			if err != nil {
				return nil, err
			}
			return ports.Conflicts(p), nil
		},
	})
	srv.RegisterTool(&mcpserver.Tool{
		Name:        "mcp_health",
		Description: "Report health status of discovered MCP servers. By default returns 'unknown' for each server without executing anything. Pass probe=true to actually probe each server (WARNING: executes commands and makes HTTP requests from MCP config files).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"probe":{"type":"boolean","description":"Actually probe each server by executing its command or making an HTTP request. Default: false."}}}`),
		Handler: func(_ context.Context, in json.RawMessage) (any, error) {
			args := struct {
				Probe bool `json:"probe"`
			}{}
			_ = json.Unmarshal(in, &args)
			servers, _ := mcpcfg.Discover(mcpcfg.DefaultSources())
			if !args.Probe {
				results := make([]model.MCPHealthResult, len(servers))
				for i, s := range servers {
					results[i] = model.MCPHealthResult{Name: s.Name, Client: s.Client, Status: "unknown"}
				}
				return results, nil
			}
			return mcphealth.ProbeAll(servers), nil
		},
	})
}

// Serve starts the MCP server on stdio with graceful shutdown.
func Serve(version string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := mcpserver.New("symscope", version)
	Register(srv)
	return srv.ServeStdio(ctx)
}
