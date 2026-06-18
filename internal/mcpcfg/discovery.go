// Package mcpcfg discovers MCP servers configured across local AI clients by
// parsing their well-known config files (no network, read-only).
package mcpcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danieljustus/symaira-scope/internal/model"
	"gopkg.in/yaml.v3"
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
		// Original clients.
		{"claude-desktop", filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json"), "mcpServers"},
		{"claude-code", filepath.Join(home, ".claude.json"), "mcpServers"},
		{"cursor", filepath.Join(home, ".cursor/mcp.json"), "mcpServers"},
		{"windsurf", filepath.Join(home, ".codeium/windsurf/mcp_config.json"), "mcpServers"},
		{"vscode", filepath.Join(home, "Library/Application Support/Code/User/mcp.json"), "servers"},
		{"project", filepath.Join(cwd, ".mcp.json"), "mcpServers"},
		// New clients (issue #5).
		{"cline", filepath.Join(home, ".config/cline/mcp_config.json"), "mcpServers"},
		{"continue", filepath.Join(home, ".continue/config.json"), "mcpServers"},
		{"goose", filepath.Join(home, ".config/goose/config.yaml"), "mcp_servers"},
		{"aider", filepath.Join(home, ".aider.conf.yml"), "mcp"},
		{"aider", filepath.Join(home, ".aider/conf.yaml"), "mcp"},
		{"roo-code", filepath.Join(home, ".vscode/extensions/*roo*/settings/roo_mcp_settings.json"), "mcpServers"},
		{"zed", filepath.Join(home, ".config/zed/mcp.json"), "mcpServers"},
		{"vscode-workspace", filepath.Join(cwd, ".vscode/mcp.json"), "servers"},
	}
}

type Entry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url"`
	Type    string            `json:"type"`
	Env     map[string]string `json:"env"`
}

// parseConfig reads a config file and returns the server map under the given key.
// It detects JSON vs YAML by file extension (.yaml/.yml → YAML, else JSON).
func parseConfig(path, key string) (map[string]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		return parseYAMLConfig(data, key)
	}
	return parseJSONConfig(data, key)
}

func parseJSONConfig(data []byte, key string) (map[string]Entry, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	raw, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	var servers map[string]Entry
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("parse servers: %w", err)
	}
	return servers, nil
}

func parseYAMLConfig(data []byte, key string) (map[string]Entry, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	raw, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml subtree: %w", err)
	}
	var servers map[string]Entry
	if err := json.Unmarshal(jsonBytes, &servers); err != nil {
		return nil, fmt.Errorf("parse servers: %w", err)
	}
	return servers, nil
}

// expandGlob returns one Source per glob match, or the original source when
// the path contains no wildcard characters.
func expandGlob(s Source) []Source {
	if !strings.Contains(s.Path, "*") {
		return []Source{s}
	}
	matches, err := filepath.Glob(s.Path)
	if err != nil || len(matches) == 0 {
		return nil
	}
	out := make([]Source, 0, len(matches))
	for _, m := range matches {
		out = append(out, Source{Client: s.Client, Path: m, Key: s.Key})
	}
	return out
}

// Discover parses each source that exists and returns the servers found.
func Discover(sources []Source) []model.MCPServer {
	// Expand any glob patterns in source paths.
	var expanded []Source
	for _, s := range sources {
		expanded = append(expanded, expandGlob(s)...)
	}

	var out []model.MCPServer
	seen := map[string]bool{} // "client:name" → already emitted
	for _, s := range expanded {
		servers, err := parseConfig(s.Path, s.Key)
		if err != nil {
			continue
		}
		for name, e := range servers {
			dedupKey := s.Client + ":" + name
			if seen[dedupKey] {
				continue
			}
			seen[dedupKey] = true
			transport := "stdio"
			if e.URL != "" {
				transport = "http"
			}
			if e.Type != "" {
				transport = e.Type
			}
			secretBacked := false
			for _, v := range e.Env {
				if strings.HasPrefix(v, "vault://") {
					secretBacked = true
					break
				}
			}
			out = append(out, model.MCPServer{
				Name:        name,
				Client:      s.Client,
				Transport:   transport,
				Command:     e.Command,
				Args:        e.Args,
				URL:         e.URL,
				ConfigPath:  s.Path,
				Env:         e.Env,
				SecretBacked: secretBacked,
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

// AddServer writes a new MCP server entry to a client's config file.
// If the file doesn't exist, it creates it with the proper structure.
func AddServer(source Source, name string, server Entry) error {
	data, err := os.ReadFile(source.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config: %w", err)
	}

	var doc map[string]any
	if err == nil {
		ext := strings.ToLower(filepath.Ext(source.Path))
		if ext == ".yaml" || ext == ".yml" {
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parse yaml: %w", err)
			}
		} else {
			if err := json.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parse json: %w", err)
			}
		}
	} else {
		doc = make(map[string]any)
	}

	servers, ok := doc[source.Key].(map[string]any)
	if !ok {
		servers = make(map[string]any)
	}

	serverMap := map[string]any{
		"command": server.Command,
	}
	if len(server.Args) > 0 {
		serverMap["args"] = server.Args
	}
	if server.URL != "" {
		serverMap["url"] = server.URL
	}
	if len(server.Env) > 0 {
		serverMap["env"] = server.Env
	}

	servers[name] = serverMap
	doc[source.Key] = servers

	ext := strings.ToLower(filepath.Ext(source.Path))
	if ext == ".yaml" || ext == ".yml" {
		out, err := yaml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}
		return os.WriteFile(source.Path, out, 0644)
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return os.WriteFile(source.Path, append(out, '\n'), 0644)
}

// RemoveServer removes an MCP server entry from a client's config file.
func RemoveServer(source Source, name string) error {
	data, err := os.ReadFile(source.Path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var doc map[string]any
	ext := strings.ToLower(filepath.Ext(source.Path))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse yaml: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	}

	servers, ok := doc[source.Key].(map[string]any)
	if !ok {
		return fmt.Errorf("no servers found under key %q", source.Key)
	}

	if _, exists := servers[name]; !exists {
		return fmt.Errorf("server %q not found in %s config", name, source.Client)
	}

	delete(servers, name)
	doc[source.Key] = servers

	if ext == ".yaml" || ext == ".yml" {
		out, err := yaml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}
		return os.WriteFile(source.Path, out, 0644)
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return os.WriteFile(source.Path, append(out, '\n'), 0644)
}
