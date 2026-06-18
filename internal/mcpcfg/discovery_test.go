package mcpcfg

import (
	"os"
	"path/filepath"
	"testing"
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

	got := Discover([]Source{{Client: "test", Path: path, Key: "mcpServers"}})
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
