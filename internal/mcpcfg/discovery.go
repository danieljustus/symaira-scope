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

	"github.com/danieljustus/symaira-corekit/fsutil"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/tailscale/hujson"
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
		{"kiro", filepath.Join(home, ".kiro/settings/mcp.json"), "mcpServers"},
		{"qoder", filepath.Join(home, ".qoder/settings.json"), "mcpServers"},
		{"copilot-cli", filepath.Join(home, ".copilot/mcp-config.json"), "mcpServers"},
		{"lmstudio", filepath.Join(home, ".lmstudio/mcp.json"), "mcpServers"},
		{"antigravity", filepath.Join(home, ".gemini/config/mcp_config.json"), "mcpServers"},
		{"gemini-cli", filepath.Join(home, ".gemini/settings.json"), "mcpServers"},
	}
}

type Entry struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	URL       string            `json:"url"`
	ServerURL string            `json:"serverUrl"`
	Type      string            `json:"type"`
	Env       map[string]string `json:"env"`
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
	// Use hujson for JSONC-aware parsing (handles comments, trailing commas).
	ast, err := hujson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	// Standardize produces valid JSON, stripping comments and trailing commas.
	packed := ast.Pack()
	standard, err := hujson.Standardize(packed)
	if err != nil {
		return nil, fmt.Errorf("standardize json: %w", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(standard, &doc); err != nil {
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

var candidateKeys = []string{"mcpServers", "servers", "mcp_servers", "context_servers", "mcp.servers", "mcp"}

func parseConfigAutoDetect(path string) (map[string]Entry, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		return parseYAMLConfigAutoDetect(data, candidateKeys)
	}
	return parseJSONConfigAutoDetect(data, candidateKeys)
}

func parseJSONConfigAutoDetect(data []byte, candidates []string) (map[string]Entry, string, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("parse json: %w", err)
	}
	for _, key := range candidates {
		raw, err := lookupJSONKey(doc, key)
		if err != nil {
			continue
		}
		var servers map[string]Entry
		if err := json.Unmarshal(raw, &servers); err == nil {
			return servers, key, nil
		}
	}
	return nil, "", fmt.Errorf("no known server key found")
}

func lookupJSONKey(doc map[string]json.RawMessage, key string) (json.RawMessage, error) {
	parts := strings.Split(key, ".")
	current := doc
	for i, part := range parts {
		raw, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", key)
		}
		if i == len(parts)-1 {
			return raw, nil
		}
		var next map[string]json.RawMessage
		if err := json.Unmarshal(raw, &next); err != nil {
			return nil, err
		}
		current = next
	}
	return nil, fmt.Errorf("key %q not found", key)
}

func parseYAMLConfigAutoDetect(data []byte, candidates []string) (map[string]Entry, string, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("parse yaml: %w", err)
	}
	for _, key := range candidates {
		raw, err := lookupYAMLKey(doc, key)
		if err != nil {
			continue
		}
		jsonBytes, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var servers map[string]Entry
		if err := json.Unmarshal(jsonBytes, &servers); err == nil {
			return servers, key, nil
		}
	}
	return nil, "", fmt.Errorf("no known server key found")
}

func lookupYAMLKey(doc map[string]any, key string) (any, error) {
	parts := strings.Split(key, ".")
	current := doc
	for i, part := range parts {
		v, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", key)
		}
		if i == len(parts)-1 {
			return v, nil
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("key %q is not a map", key)
		}
		current = next
	}
	return nil, fmt.Errorf("key %q not found", key)
}

// DiscoverFiles parses explicit config files and returns the servers found under
// any known key. Errors for individual files are returned as notes.
func DiscoverFiles(paths []string) ([]model.MCPServer, []string) {
	var out []model.MCPServer
	var notes []string
	seen := map[string]bool{}
	for _, p := range paths {
		servers, _, err := parseConfigAutoDetect(p)
		if err != nil {
			notes = append(notes, fmt.Sprintf("config parse error for %s: %v", p, err))
			continue
		}
		for name, e := range servers {
			dedupKey := p + ":" + name
			if seen[dedupKey] {
				continue
			}
			seen[dedupKey] = true
			url := e.URL
			if url == "" {
				url = e.ServerURL
			}
			transport := "stdio"
			if url != "" {
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
				Name:         name,
				Client:       "file",
				Transport:    transport,
				Command:      e.Command,
				Args:         e.Args,
				URL:          url,
				ConfigPath:   p,
				Env:          e.Env,
				SecretBacked: secretBacked,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ConfigPath < out[j].ConfigPath
	})
	return out, notes
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
// Any parse errors are returned as notes in the second return value.
func Discover(sources []Source) ([]model.MCPServer, []string) {
	// Expand any glob patterns in source paths.
	var expanded []Source
	for _, s := range sources {
		expanded = append(expanded, expandGlob(s)...)
	}

	var out []model.MCPServer
	var notes []string
	seen := map[string]bool{} // "client:name" → already emitted
	for _, s := range expanded {
		servers, err := parseConfig(s.Path, s.Key)
		if err != nil {
			notes = append(notes, fmt.Sprintf("config parse error for %s (%s): %v", s.Client, s.Path, err))
			continue
		}
		for name, e := range servers {
			dedupKey := s.Client + ":" + name
			if seen[dedupKey] {
				continue
			}
			seen[dedupKey] = true
			url := e.URL
			if url == "" {
				url = e.ServerURL
			}
			transport := "stdio"
			if url != "" {
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
				Name:         name,
				Client:       s.Client,
				Transport:    transport,
				Command:      e.Command,
				Args:         e.Args,
				URL:          url,
				ConfigPath:   s.Path,
				Env:          e.Env,
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
	return out, notes
}

// FoundClients reports which known client configs are present on disk.
// Glob patterns in source paths are expanded so that clients like roo-code
// (which use wildcard extension paths) are correctly detected.
func FoundClients(sources []Source) []model.ClientConfig {
	seen := map[string]bool{}
	var out []model.ClientConfig
	for _, s := range sources {
		if seen[s.Client] {
			continue
		}
		expanded := expandGlob(s)
		present := false
		for _, e := range expanded {
			if _, err := os.Stat(e.Path); err == nil {
				present = true
				break
			}
		}
		seen[s.Client] = true
		out = append(out, model.ClientConfig{Client: s.Client, Path: s.Path, Present: present})
	}
	return out
}

// AddServer writes a new MCP server entry to a client's config file.
// If the file doesn't exist, it creates it with the proper structure.
func AddServer(source Source, name string, server Entry) error {
	ext := strings.ToLower(filepath.Ext(source.Path))
	isYAML := ext == ".yaml" || ext == ".yml"

	// YAML path — keep existing unmarshal/marshal behaviour (comments
	// and key-order are out of scope for YAML in this issue).
	if isYAML {
		return addServerYAML(source, name, server)
	}

	// JSONC-aware path.
	data, err := os.ReadFile(source.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read config: %w", err)
		}
		// File doesn't exist — create a new one.
		return addServerNewFile(source, name, server)
	}

	// File exists — use the surgical JSONC splicer.
	entryJSON, err := marshalEntry(server)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	out, err := jsoncAddMember(data, source.Key, name, entryJSON)
	if err != nil {
		// Key doesn't exist yet — add the whole key+value as a new
		// top-level member. The value must be a complete JSON object
		// mapping the server name to its entry; previously the name was
		// omitted, producing invalid JSONC ({{...}}) that always failed
		// to parse.
		serverObj, err := json.Marshal(map[string]Entry{name: server})
		if err != nil {
			return fmt.Errorf("marshal server object: %w", err)
		}
		out, err = jsoncAddTopLevelKey(data, source.Key, serverObj)
		if err != nil {
			return fmt.Errorf("add top-level key: %w", err)
		}
	}

	return writeJSON(source.Path, out)
}

// RemoveServer removes an MCP server entry from a client's config file.
func RemoveServer(source Source, name string) error {
	data, err := os.ReadFile(source.Path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(source.Path))
	if ext == ".yaml" || ext == ".yml" {
		return removeServerYAML(data, source, name)
	}

	// JSONC-aware path.
	out, err := jsoncRemoveMember(data, source.Key, name)
	if err != nil {
		return fmt.Errorf("remove server %q from %s config: %w", name, source.Client, err)
	}

	return writeJSON(source.Path, out)
}

// writeConfig marshals the document to the appropriate format and writes it
// atomically. The original file is preserved until the write succeeds,
// preventing corruption on crash or interrupt.
//
// When overwriting an existing config file the original mode is preserved so
// that restrictive permissions (e.g. 0o600 for files containing secrets) are
// not silently widened. New files default to 0o600.
func writeConfig(path string, doc map[string]any) error {
	// Preserve the existing file's permissions; default to 0o600 for new files
	// so that AI client configs containing env secrets are not world-readable.
	perm := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		out, err := yaml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}
		return fsutil.AtomicWriteFile(path, out, perm)
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return fsutil.AtomicWriteFile(path, append(out, '\n'), perm)
}

// addServerYAML preserves the existing unmarshal/marshal behaviour for
// YAML config files (comments and key-order are out of scope here).
func addServerYAML(source Source, name string, server Entry) error {
	data, err := os.ReadFile(source.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config: %w", err)
	}

	var doc map[string]any
	if err == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse yaml: %w", err)
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

	return writeConfig(source.Path, doc)
}

// addServerNewFile creates a new JSON config file with a single server entry.
func addServerNewFile(source Source, name string, server Entry) error {
	doc := make(map[string]any)
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
	doc[source.Key] = map[string]any{name: serverMap}
	return writeConfig(source.Path, doc)
}

// removeServerYAML preserves the existing behaviour for YAML config files.
func removeServerYAML(data []byte, source Source, name string) error {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
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

	return writeConfig(source.Path, doc)
}

// marshalEntry serialises an Entry to its compact JSON representation.
func marshalEntry(server Entry) ([]byte, error) {
	m := map[string]any{
		"command": server.Command,
	}
	if len(server.Args) > 0 {
		m["args"] = server.Args
	}
	if server.URL != "" {
		m["url"] = server.URL
	}
	if len(server.Env) > 0 {
		m["env"] = server.Env
	}
	return json.Marshal(m)
}

// writeJSON writes pre-formatted JSONC bytes atomically, preserving the
// existing file's mode (default 0o600 for new files).
func writeJSON(path string, data []byte) error {
	perm := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	return fsutil.AtomicWriteFile(path, append(data, '\n'), perm)
}
