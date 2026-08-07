package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestMCPListManifestFormat(t *testing.T) {
	cfg := writeMCPServerConfig(t, map[string]any{
		"demo-stdio": map[string]any{"command": "echo", "args": []string{"hi"}},
		"demo-http":  map[string]any{"url": "http://127.0.0.1:1"},
		"demo-sse":   map[string]any{"url": "http://127.0.0.1:2", "type": "sse"},
		"demo-env":   map[string]any{"command": "echo", "env": map[string]string{"FOO": "bar"}},
	})
	stdout, stderr, code := runCLIProcess(t, []string{"HOME=" + t.TempDir()},
		"mcp", "list", "--format", "manifest", "--files", cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}

	var entries []model.MCPServerManifest
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries); err != nil {
		t.Fatalf("stdout %q is not valid JSON: %v", stdout, err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 manifest entries, got %d", len(entries))
	}

	byName := map[string]model.MCPServerManifest{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	if byName["demo-stdio"].Transport.Type != "stdio" {
		t.Errorf("demo-stdio transport = %q, want stdio", byName["demo-stdio"].Transport.Type)
	}
	if byName["demo-http"].Transport.Type != "http" {
		t.Errorf("demo-http transport = %q, want http", byName["demo-http"].Transport.Type)
	}
	if byName["demo-sse"].Transport.Type != "http" {
		t.Errorf("demo-sse transport = %q, want http (SSE is reached over HTTP)", byName["demo-sse"].Transport.Type)
	}

	for _, name := range []string{"demo-stdio", "demo-http", "demo-sse", "demo-env"} {
		e := byName[name]
		if e.Client != "file" {
			t.Errorf("%s: client = %q, want file", name, e.Client)
		}
		if len(e.Packages) != 1 {
			t.Errorf("%s: expected 1 package, got %d", name, len(e.Packages))
			continue
		}
		if e.Packages[0].RegistryType != "local-binary" {
			t.Errorf("%s: registryType = %q, want local-binary", name, e.Packages[0].RegistryType)
		}
		if e.Packages[0].Identifier != name {
			t.Errorf("%s: identifier = %q, want %q", name, e.Packages[0].Identifier, name)
		}
		if e.Packages[0].Version != "" {
			t.Errorf("%s: version must be omitted for local discovery, got %q", name, e.Packages[0].Version)
		}
	}

	if env := byName["demo-env"].EnvironmentVariables; env["FOO"] != "bar" {
		t.Errorf("demo-env environmentVariables = %v, want FOO=bar", env)
	}
	if env := byName["demo-stdio"].EnvironmentVariables; len(env) != 0 {
		t.Errorf("demo-stdio must omit environmentVariables, got %v", env)
	}
}

func TestMCPListDefaultFormatUnchanged(t *testing.T) {
	cfg := writeMCPServerConfig(t, map[string]any{
		"demo-stdio": map[string]any{"command": "echo", "args": []string{"hi"}},
		"demo-http":  map[string]any{"url": "http://127.0.0.1:1"},
	})
	env := []string{"HOME=" + t.TempDir()}

	// Default `mcp list` (no --format) must stay byte-identical to the
	// explicit json format, and keep the raw MCPServer shape (no
	// packages/environmentVariables split).
	defOut, defErr, defCode := runCLIProcess(t, env, "mcp", "list", "--files", cfg)
	if defCode != 0 {
		t.Fatalf("default mcp list exit code = %d (stderr: %s)", defCode, defErr)
	}
	jsonOut, jsonErr, jsonCode := runCLIProcess(t, env, "mcp", "list", "--format", "json", "--files", cfg)
	if jsonCode != 0 {
		t.Fatalf("mcp list --format json exit code = %d (stderr: %s)", jsonCode, jsonErr)
	}
	if defOut != jsonOut {
		t.Errorf("default output differs from --format json:\n--- default ---\n%s\n--- json ---\n%s", defOut, jsonOut)
	}

	var servers []model.MCPServer
	if err := json.Unmarshal([]byte(strings.TrimSpace(defOut)), &servers); err != nil {
		t.Fatalf("default output %q is not valid JSON: %v", defOut, err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	byName := map[string]model.MCPServer{}
	for _, s := range servers {
		byName[s.Name] = s
	}
	if byName["demo-stdio"].Transport != "stdio" || byName["demo-stdio"].Command != "echo" {
		t.Errorf("expected raw MCPServer fields (transport/command), got %+v", byName["demo-stdio"])
	}
	if byName["demo-http"].URL != "http://127.0.0.1:1" {
		t.Errorf("demo-http url = %q, want http://127.0.0.1:1", byName["demo-http"].URL)
	}

	var raw []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(defOut)), &raw); err != nil {
		t.Fatal(err)
	}
	for _, entry := range raw {
		if _, ok := entry["packages"]; ok {
			t.Errorf("default output must not contain packages key: %v", entry)
		}
		if _, ok := entry["environmentVariables"]; ok {
			t.Errorf("default output must not contain environmentVariables key: %v", entry)
		}
	}
}

func TestMCPListFormatValidation(t *testing.T) {
	isolateCLIHome(t)
	expectCLIError(t, []string{"mcp", "list", "--format", "xml"}, exitcodes.ExitConfig, `unsupported format "xml"`)
}
