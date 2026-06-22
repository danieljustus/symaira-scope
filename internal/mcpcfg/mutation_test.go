package mcpcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddServerNewJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	src := Source{Client: "test", Path: path, Key: "mcpServers"}

	err := AddServer(src, "my-server", Entry{Command: "node", Args: []string{"server.js"}})
	if err != nil {
		t.Fatalf("AddServer to new file: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["my-server"].Command != "node" {
		t.Errorf("command: want %q, got %q", "node", servers["my-server"].Command)
	}
	if len(servers["my-server"].Args) != 1 || servers["my-server"].Args[0] != "server.js" {
		t.Errorf("args: want [server.js], got %v", servers["my-server"].Args)
	}
}

func TestAddServerExistingJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	existing := `{"mcpServers":{"existing":{"command":"old-cmd"}}}`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	err := AddServer(src, "new-server", Entry{Command: "python"})
	if err != nil {
		t.Fatalf("AddServer to existing file: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["existing"].Command != "old-cmd" {
		t.Errorf("existing server should be preserved, got %q", servers["existing"].Command)
	}
	if servers["new-server"].Command != "python" {
		t.Errorf("new server: want %q, got %q", "python", servers["new-server"].Command)
	}
}

func TestAddServerOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	existing := `{"mcpServers":{"my-server":{"command":"old"}}}`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	err := AddServer(src, "my-server", Entry{Command: "new"})
	if err != nil {
		t.Fatalf("AddServer overwrite: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["my-server"].Command != "new" {
		t.Errorf("overwrite: want %q, got %q", "new", servers["my-server"].Command)
	}
}

func TestAddServerYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	src := Source{Client: "goose", Path: path, Key: "mcp_servers"}

	err := AddServer(src, "yaml-server", Entry{Command: "go", URL: "http://localhost:9090"})
	if err != nil {
		t.Fatalf("AddServer YAML: %v", err)
	}

	servers, err := parseConfig(path, "mcp_servers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["yaml-server"].Command != "go" {
		t.Errorf("command: want %q, got %q", "go", servers["yaml-server"].Command)
	}
	if servers["yaml-server"].URL != "http://localhost:9090" {
		t.Errorf("url: want %q, got %q", "http://localhost:9090", servers["yaml-server"].URL)
	}
}

func TestAddServerPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	// Create with restrictive permissions.
	if err := os.WriteFile(path, []byte(`{"mcpServers":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "s", Entry{Command: "x"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("permissions: want 0o644, got %o", info.Mode().Perm())
	}
}

func TestRemoveServerJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	data := `{"mcpServers":{"keep":{"command":"a"},"remove":{"command":"b"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := RemoveServer(src, "remove"); err != nil {
		t.Fatalf("RemoveServer: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if _, ok := servers["remove"]; ok {
		t.Error("expected 'remove' to be deleted")
	}
	if servers["keep"].Command != "a" {
		t.Errorf("'keep' should be preserved, got %q", servers["keep"].Command)
	}
}

func TestRemoveServerMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	data := `{"mcpServers":{"existing":{"command":"x"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	err := RemoveServer(src, "nonexistent")
	if err == nil {
		t.Fatal("expected error when removing nonexistent server")
	}
}

func TestRemoveServerYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := `mcp_servers:
  keep:
    command: a
  remove:
    command: b
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "goose", Path: path, Key: "mcp_servers"}
	if err := RemoveServer(src, "remove"); err != nil {
		t.Fatalf("RemoveServer YAML: %v", err)
	}

	servers, err := parseConfig(path, "mcp_servers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if _, ok := servers["remove"]; ok {
		t.Error("expected 'remove' to be deleted")
	}
	if servers["keep"].Command != "a" {
		t.Errorf("'keep' should be preserved, got %q", servers["keep"].Command)
	}
}

func TestRoundTripPreservesExistingEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	data := `{"mcpServers":{"alpha":{"command":"a","args":["--x"]},"beta":{"command":"b","url":"http://localhost:3000"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "gamma", Entry{Command: "c"}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveServer(src, "beta"); err != nil {
		t.Fatal(err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["alpha"].Command != "a" {
		t.Errorf("alpha: want %q, got %q", "a", servers["alpha"].Command)
	}
	if servers["gamma"].Command != "c" {
		t.Errorf("gamma: want %q, got %q", "c", servers["gamma"].Command)
	}
	if _, ok := servers["beta"]; ok {
		t.Error("beta should have been removed")
	}
}

func TestAddServerWithEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	src := Source{Client: "test", Path: path, Key: "mcpServers"}

	env := map[string]string{"TOKEN": "secret123"}
	err := AddServer(src, "env-server", Entry{Command: "node", Env: env})
	if err != nil {
		t.Fatalf("AddServer with env: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["env-server"].Command != "node" {
		t.Errorf("command: want %q, got %q", "node", servers["env-server"].Command)
	}
}
