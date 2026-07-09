package mcpcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestDiscoverParsesProjectConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	data := `{"mcpServers":{
		"vault":{"command":"symvault","args":["mcp"]},
		"remote":{"url":"http://localhost:9000"}
	}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "test", Path: path, Key: "mcpServers"}})
	if len(got) != 2 {
		t.Fatalf("want 2 servers, got %d", len(got))
	}

	byName := map[string]string{}
	for _, s := range got {
		byName[s.Name] = s.Transport
	}
	if byName["vault"] != "stdio" {
		t.Errorf("vault should be stdio, got %q", byName["vault"])
	}
	if byName["remote"] != "http" {
		t.Errorf("remote should be http, got %q", byName["remote"])
	}
}

func TestFoundClientsMarksMissing(t *testing.T) {
	got := FoundClients([]Source{{Client: "nope", Path: "/does/not/exist.json", Key: "mcpServers"}})
	if len(got) != 1 || got[0].Present {
		t.Fatalf("missing config should be Present=false, got %+v", got)
	}
}

func TestDiscoverYAMLGoose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := `mcp_servers:
  my-server:
    command: python
    args: ["-m", "server"]
  remote:
    url: http://localhost:8080
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "goose", Path: path, Key: "mcp_servers"}})
	if len(got) != 2 {
		t.Fatalf("want 2 servers, got %d", len(got))
	}

	byName := map[string]model.MCPServer{}
	for _, s := range got {
		byName[s.Name] = s
	}
	if byName["my-server"].Command != "python" {
		t.Errorf("my-server command: want %q, got %q", "python", byName["my-server"].Command)
	}
	if byName["remote"].Transport != "http" {
		t.Errorf("remote transport: want %q, got %q", "http", byName["remote"].Transport)
	}
}

func TestDiscoverYAMLStructuredGoose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Goose-style with nested env and type fields.
	data := `mcp_servers:
  my-server:
    command: node
    args: ["server.js"]
    env:
      API_KEY: abc123
      DEBUG: "true"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "goose", Path: path, Key: "mcp_servers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Command != "node" {
		t.Errorf("command: want %q, got %q", "node", got[0].Command)
	}
}

func TestDiscoverYAMLKeyNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := `other_key:
  foo: bar
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "test", Path: path, Key: "mcp_servers"}})
	if len(got) != 0 {
		t.Fatalf("want 0 servers when key missing, got %d", len(got))
	}
}

func TestDiscoverAiderYML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".aider.conf.yml")
	data := `mcp:
  my-tools:
    command: aider-mcp
    args: ["--stdio"]
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "aider", Path: path, Key: "mcp"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "my-tools" {
		t.Errorf("name: want %q, got %q", "my-tools", got[0].Name)
	}
	if got[0].Command != "aider-mcp" {
		t.Errorf("command: want %q, got %q", "aider-mcp", got[0].Command)
	}
}

func TestDiscoverGlobExpansion(t *testing.T) {
	dir := t.TempDir()
	extDir := filepath.Join(dir, "extensions")
	// Create two matching extension dirs.
	for _, name := range []string{"roo-cline-1.0.0", "roo-code-2.0.0"} {
		settingsDir := filepath.Join(extDir, name, "settings")
		if err := os.MkdirAll(settingsDir, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(settingsDir, "roo_mcp_settings.json")
		data := `{"mcpServers":{"server-` + name + `":{"command":"echo"}}}`
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	pattern := filepath.Join(extDir, "*roo*", "settings", "roo_mcp_settings.json")
	got, _ := Discover([]Source{{Client: "roo-code", Path: pattern, Key: "mcpServers"}})
	if len(got) != 2 {
		t.Fatalf("want 2 servers from glob, got %d", len(got))
	}

	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["server-roo-cline-1.0.0"] || !names["server-roo-code-2.0.0"] {
		t.Errorf("expected both roo servers, got %v", names)
	}
}

func TestDiscoverGlobNoMatch(t *testing.T) {
	dir := t.TempDir()
	pattern := filepath.Join(dir, "*nonexistent*", "settings.json")
	got, _ := Discover([]Source{{Client: "test", Path: pattern, Key: "mcpServers"}})
	if len(got) != 0 {
		t.Fatalf("want 0 servers from non-matching glob, got %d", len(got))
	}
}

func TestDiscoverDeduplicatesByClientName(t *testing.T) {
	dir := t.TempDir()

	// Two sources with the same client and overlapping server names.
	path1 := filepath.Join(dir, "global.json")
	data1 := `{"mcpServers":{"shared":{"command":"cmd1"},"only-global":{"command":"cmd2"}}}`
	if err := os.WriteFile(path1, []byte(data1), 0o600); err != nil {
		t.Fatal(err)
	}

	path2 := filepath.Join(dir, "workspace.json")
	data2 := `{"mcpServers":{"shared":{"command":"cmd3"},"only-workspace":{"command":"cmd4"}}}`
	if err := os.WriteFile(path2, []byte(data2), 0o600); err != nil {
		t.Fatal(err)
	}

	sources := []Source{
		{Client: "vscode", Path: path1, Key: "mcpServers"},
		{Client: "vscode", Path: path2, Key: "mcpServers"},
	}
	got, _ := Discover(sources)
	if len(got) != 3 {
		t.Fatalf("want 3 unique servers (dedup shared), got %d", len(got))
	}

	// The first-seen "shared" entry should win (from global).
	for _, s := range got {
		if s.Name == "shared" && s.Command != "cmd1" {
			t.Errorf("dedup: want first-seen command %q, got %q", "cmd1", s.Command)
		}
	}
}

func TestDiscoverJSONWithEnvField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"mcpServers":{"env-server":{"command":"node","args":["s.js"],"env":{"TOKEN":"xyz"}}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "test", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Command != "node" {
		t.Errorf("command: want %q, got %q", "node", got[0].Command)
	}
}

func TestDiscoverCline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	data := `{"mcpServers":{"cline-server":{"command":"cline-mcp"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "cline", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Client != "cline" {
		t.Errorf("client: want %q, got %q", "cline", got[0].Client)
	}
}

func TestDiscoverContinue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"mcpServers":{"continue-server":{"command":"continue-mcp","args":["--stdio"]}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "continue", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "continue-server" {
		t.Errorf("name: want %q, got %q", "continue-server", got[0].Name)
	}
}

func TestDiscoverZed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	data := `{"mcpServers":{"zed-server":{"command":"zed-mcp","url":"http://localhost:3000"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "zed", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Transport != "http" {
		t.Errorf("transport: want %q, got %q", "http", got[0].Transport)
	}
}

func TestDiscoverKiro(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	data := `{"mcpServers":{"kiro-server":{"command":"kiro-mcp"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "kiro", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "kiro-server" {
		t.Errorf("name: want %q, got %q", "kiro-server", got[0].Name)
	}
	if got[0].Client != "kiro" {
		t.Errorf("client: want %q, got %q", "kiro", got[0].Client)
	}
}

func TestDiscoverQoder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"mcpServers":{"qoder-server":{"command":"qoder-mcp"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "qoder", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "qoder-server" {
		t.Errorf("name: want %q, got %q", "qoder-server", got[0].Name)
	}
}

func TestDiscoverCopilotCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-config.json")
	data := `{"mcpServers":{"copilot-server":{"command":"copilot-mcp","type":"local"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "copilot-cli", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Transport != "local" {
		t.Errorf("transport: want %q, got %q", "local", got[0].Transport)
	}
}

func TestDiscoverLMStudio(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	data := `{"mcpServers":{"lmstudio-server":{"command":"lmstudio-mcp"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "lmstudio", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "lmstudio-server" {
		t.Errorf("name: want %q, got %q", "lmstudio-server", got[0].Name)
	}
}

func TestDiscoverAntigravity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	data := `{"mcpServers":{"antigravity-server":{"command":"antigravity-mcp","serverUrl":"http://localhost:4000"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "antigravity", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].URL != "http://localhost:4000" {
		t.Errorf("url: want %q, got %q", "http://localhost:4000", got[0].URL)
	}
}

func TestDiscoverGeminiCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	data := `{"mcpServers":{"gemini-server":{"command":"gemini-mcp"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, _ := Discover([]Source{{Client: "gemini-cli", Path: path, Key: "mcpServers"}})
	if len(got) != 1 {
		t.Fatalf("want 1 server, got %d", len(got))
	}
	if got[0].Name != "gemini-server" {
		t.Errorf("name: want %q, got %q", "gemini-server", got[0].Name)
	}
}

func TestExpandGlobNoWildcard(t *testing.T) {
	s := Source{Client: "test", Path: "/some/fixed/path.json", Key: "k"}
	got := expandGlob(s)
	if len(got) != 1 || got[0].Path != s.Path {
		t.Errorf("non-glob should return single source, got %+v", got)
	}
}

func TestExpandGlobWildcardNoMatch(t *testing.T) {
	dir := t.TempDir()
	s := Source{Client: "test", Path: filepath.Join(dir, "*nope*", "file.json"), Key: "k"}
	got := expandGlob(s)
	if len(got) != 0 {
		t.Errorf("non-matching glob should return empty, got %d", len(got))
	}
}
