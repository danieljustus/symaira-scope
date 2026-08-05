package mcpcfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailscale/hujson"
)

// readJSONCDoc parses a JSONC document into a generic map, tolerating
// comments and trailing commas, for whole-document assertions.
func readJSONCDoc(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	ast, err := hujson.Parse(data)
	if err != nil {
		t.Fatalf("parse jsonc: %v", err)
	}
	standard, err := hujson.Standardize(ast.Pack())
	if err != nil {
		t.Fatalf("standardize jsonc: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(standard, &doc); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	return doc
}

// TestAddServerCreatesTopLevelKey verifies that AddServer on a config that
// exists but lacks the mcpServers key writes a brand-new top-level key
// (via jsoncAddTopLevelKey) while preserving the existing keys, their
// order, and their position.
func TestAddServerCreatesTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	existing := `{"other": true}`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "my-server", Entry{Command: "node", Args: []string{"server.js"}}); err != nil {
		t.Fatalf("AddServer on config without mcpServers: %v", err)
	}

	// The new server must be readable back.
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

	// The pre-existing key must survive with its value intact.
	doc := readJSONCDoc(t, path)
	if v, ok := doc["other"].(bool); !ok || !v {
		t.Errorf("existing key %q: want true, got %v", "other", doc["other"])
	}

	// Key order must be preserved: "other" stays before the new key.
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	idxOther := strings.Index(string(out), `"other"`)
	idxMcp := strings.Index(string(out), `"mcpServers"`)
	if idxOther < 0 || idxMcp < 0 {
		t.Fatalf("expected both keys in output:\n%s", out)
	}
	if idxOther > idxMcp {
		t.Errorf("key order: %q should precede %q:\n%s", "other", "mcpServers", out)
	}

	// Default indentation of 2 spaces must be applied to the new key.
	if !strings.Contains(string(out), "\n  \"mcpServers\"") {
		t.Errorf("expected 2-space indented new key:\n%s", out)
	}
}

// TestAddServerEmptyObject verifies that AddServer on an empty JSON object
// creates the mcpServers key.
func TestAddServerEmptyObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "solo", Entry{Command: "python"}); err != nil {
		t.Fatalf("AddServer on empty object: %v", err)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["solo"].Command != "python" {
		t.Errorf("command: want %q, got %q", "python", servers["solo"].Command)
	}
}

// TestAddServerPreservesComments verifies that comments in a JSONC config
// without mcpServers are preserved when the top-level key is created.
func TestAddServerPreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	existing := "// hello\n{}"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "commented", Entry{Command: "go"}); err != nil {
		t.Fatalf("AddServer on comments-only config: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "// hello\n") {
		t.Errorf("leading comment not preserved:\n%s", out)
	}
	if !strings.Contains(string(out), `"mcpServers"`) {
		t.Errorf("mcpServers key missing:\n%s", out)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["commented"].Command != "go" {
		t.Errorf("command: want %q, got %q", "go", servers["commented"].Command)
	}
}

// TestAddServerDetectsExistingIndent verifies that the new top-level key
// matches the indentation of existing members (detectTopLevelIndent) and
// that comments inside the object are preserved.
func TestAddServerDetectsExistingIndent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	existing := "{\n    \"other\": true\n}"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "indented", Entry{Command: "node"}); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\n    \"mcpServers\"") {
		t.Errorf("expected 4-space indented new key:\n%s", out)
	}
	if strings.Contains(string(out), "\n  \"mcpServers\"") {
		t.Errorf("new key should not use default 2-space indent:\n%s", out)
	}
	if idxOther := strings.Index(string(out), `"other"`); idxOther > strings.Index(string(out), `"mcpServers"`) {
		t.Errorf("key order: %q should precede %q:\n%s", "other", "mcpServers", out)
	}

	servers, err := parseConfig(path, "mcpServers")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if servers["indented"].Command != "node" {
		t.Errorf("command: want %q, got %q", "node", servers["indented"].Command)
	}
}

// TestAddServerPreservesInnerComments verifies comments inside the root
// object survive the top-level key insertion.
func TestAddServerPreservesInnerComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	existing := "{\n  // a comment\n  \"other\": true\n}"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := AddServer(src, "c", Entry{Command: "x"}); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "// a comment") {
		t.Errorf("inner comment not preserved:\n%s", out)
	}
	if !strings.Contains(string(out), "\n  \"mcpServers\"") {
		t.Errorf("expected 2-space indented new key:\n%s", out)
	}
}

// TestAddServerNonObjectRoot verifies that AddServer propagates an error
// when the config's root is not an object, exercising the root-object
// guard inside jsoncAddTopLevelKey.
func TestAddServerNonObjectRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	err := AddServer(src, "s", Entry{Command: "x"})
	if err == nil {
		t.Fatal("expected error when root is not an object")
	}
	if !strings.Contains(err.Error(), "not an object") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRemoveServerMissingTopLevelKey verifies the error path of
// RemoveServer when the config has no mcpServers key at all (the
// jsoncRemoveMember missing-key branch), and that the file is untouched.
func TestRemoveServerMissingTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	existing := `{"other": true}`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	src := Source{Client: "test", Path: path, Key: "mcpServers"}
	if err := RemoveServer(src, "my-server"); err == nil {
		t.Fatal("expected error when config has no mcpServers key")
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != existing {
		t.Errorf("file should be unchanged, got:\n%s", out)
	}
}

// TestJSONCAddTopLevelKeyDirect exercises jsoncAddTopLevelKey directly,
// covering its error branches (invalid input JSONC, non-object root,
// invalid value JSON) and the happy path output shape.
func TestJSONCAddTopLevelKeyDirect(t *testing.T) {
	// Invalid input document.
	if _, err := jsoncAddTopLevelKey([]byte(`{`), "mcpServers", []byte(`{}`)); err == nil {
		t.Error("expected error for invalid input jsonc")
	}

	// Root is not an object.
	if _, err := jsoncAddTopLevelKey([]byte(`[1,2]`), "mcpServers", []byte(`{}`)); err == nil {
		t.Error("expected error for non-object root")
	}

	// Invalid value JSON.
	if _, err := jsoncAddTopLevelKey([]byte(`{}`), "mcpServers", []byte(`{`)); err == nil {
		t.Error("expected error for invalid value json")
	}

	// Happy path: key appended with default 2-space indent, existing
	// member preserved.
	out, err := jsoncAddTopLevelKey([]byte(`{"other": true}`), "mcpServers", []byte(`{"s":{"command":"x"}}`))
	if err != nil {
		t.Fatalf("jsoncAddTopLevelKey: %v", err)
	}
	if !strings.Contains(string(out), `"other": true`) || !strings.Contains(string(out), `"mcpServers"`) {
		t.Errorf("unexpected output:\n%s", out)
	}
	if !strings.Contains(string(out), "\n  \"mcpServers\"") {
		t.Errorf("expected 2-space indented new key:\n%s", out)
	}
	if strings.Index(string(out), `"other"`) > strings.Index(string(out), `"mcpServers"`) {
		t.Errorf("key order not preserved:\n%s", out)
	}

	// Empty root object.
	out, err = jsoncAddTopLevelKey([]byte(`{}`), "mcpServers", []byte(`{}`))
	if err != nil {
		t.Fatalf("jsoncAddTopLevelKey empty root: %v", err)
	}
	if !strings.Contains(string(out), `"mcpServers"`) {
		t.Errorf("mcpServers missing from output:\n%s", out)
	}
}
