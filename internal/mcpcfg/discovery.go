// Package mcpcfg discovers MCP servers configured across local AI clients by
// parsing their well-known config files (no network, read-only).
package mcpcfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// Source is one config file to inspect and the JSON key holding the servers.
type Source struct {
	Client string
	Path   string
	Key    string
}

// DefaultSources lists the AI-client config locations symscope knows about.
// More clients are added over time (see roadmap).
func DefaultSources() []Source {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return []Source{
		{"claude-desktop", filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json"), "mcpServers"},
		{"claude-code", filepath.Join(home, ".claude.json"), "mcpServers"},
		{"cursor", filepath.Join(home, ".cursor/mcp.json"), "mcpServers"},
		{"windsurf", filepath.Join(home, ".codeium/windsurf/mcp_config.json"), "mcpServers"},
		{"vscode", filepath.Join(home, "Library/Application Support/Code/User/mcp.json"), "servers"},
		{"project", filepath.Join(cwd, ".mcp.json"), "mcpServers"},
	}
}

type entry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	URL     string   `json:"url"`
	Type    string   `json:"type"`
}

// Discover parses each source that exists and returns the servers found.
func Discover(sources []Source) []model.MCPServer {
	var out []model.MCPServer
	for _, s := range sources {
		data, err := os.ReadFile(s.Path)
		if err != nil {
			continue
		}
		var doc map[string]json.RawMessage
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		raw, ok := doc[s.Key]
		if !ok {
			continue
		}
		var servers map[string]entry
		if json.Unmarshal(raw, &servers) != nil {
			continue
		}
		for name, e := range servers {
			transport := "stdio"
			if e.URL != "" {
				transport = "http"
			}
			if e.Type != "" {
				transport = e.Type
			}
			out = append(out, model.MCPServer{
				Name:       name,
				Client:     s.Client,
				Transport:  transport,
				Command:    e.Command,
				Args:       e.Args,
				URL:        e.URL,
				ConfigPath: s.Path,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Client != out[j].Client {
			return out[i].Client < out[j].Client
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// FoundClients reports which known client configs are present on disk.
func FoundClients(sources []Source) []model.ClientConfig {
	out := make([]model.ClientConfig, 0, len(sources))
	for _, s := range sources {
		_, err := os.Stat(s.Path)
		out = append(out, model.ClientConfig{Client: s.Client, Path: s.Path, Present: err == nil})
	}
	return out
}
