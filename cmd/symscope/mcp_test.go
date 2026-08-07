package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-corekit/exitcodes"
)

// writeMCPServerConfig writes a JSON MCP config file (mcpServers key) for
// --files-based tests and returns its path.
func writeMCPServerConfig(t *testing.T, servers map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcp.json")
	doc := map[string]any{"mcpServers": servers}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestMCPInspectorHelperProcess is a re-exec helper that acts as a minimal MCP
// stdio server for `mcp inspect` CLI tests: it reads the JSON-RPC request,
// echoes the SYMSCOPE_MCP_INSPECT_HELPER_ECHO env var (proving that the
// server's env reaches the child) and answers with a canned tools/list
// result. Not a real test — it returns immediately unless re-executed with
// SYMSCOPE_MCP_INSPECT_HELPER=1.
func TestMCPInspectorHelperProcess(t *testing.T) {
	if os.Getenv("SYMSCOPE_MCP_INSPECT_HELPER") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	resp, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]any{
			"tools": []map[string]any{{
				"name":        "demo-tool",
				"description": "a demo tool",
				"inputSchema": map[string]any{"type": "object"},
			}},
			"echo_env": os.Getenv("SYMSCOPE_MCP_INSPECT_HELPER_ECHO"),
		},
	})
	fmt.Fprint(os.Stdout, string(resp))
	os.Exit(0)
}

func TestMCPInspectToolsListHTTP(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotMethod, _ = req["method"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"demo-tool","description":"a demo tool","inputSchema":{"type":"object"}}]}}`))
	}))
	defer srv.Close()

	cfg := writeMCPServerConfig(t, map[string]any{"demo": map[string]any{"url": srv.URL}})
	stdout, stderr, code := runCLIProcess(t, []string{"HOME=" + t.TempDir()},
		"mcp", "inspect", "--name", "demo", "--method", "tools/list", "--files", cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if gotMethod != "tools/list" {
		t.Errorf("server received method %q, want tools/list", gotMethod)
	}

	var got struct {
		Name   string `json:"name"`
		Method string `json:"method"`
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	// --format json output must be pure, script-parseable JSON (no banner text).
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("stdout %q is not clean JSON: %v", stdout, err)
	}
	if got.Name != "demo" || got.Method != "tools/list" {
		t.Errorf("unexpected envelope: name=%q method=%q", got.Name, got.Method)
	}
	if len(got.Result.Tools) != 1 || got.Result.Tools[0].Name != "demo-tool" {
		t.Errorf("unexpected tools: %+v", got.Result.Tools)
	}
}

func TestMCPInspectToolsCallHTTP(t *testing.T) {
	var gotParams map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotParams, _ = req["params"].(map[string]any)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	}))
	defer srv.Close()

	cfg := writeMCPServerConfig(t, map[string]any{"demo": map[string]any{"url": srv.URL}})
	stdout, stderr, code := runCLIProcess(t, []string{"HOME=" + t.TempDir()},
		"mcp", "inspect", "--name", "demo", "--method", "tools/call",
		"--tool-name", "echo", "--tool-arg", "text=hello", "--tool-arg", "port=3000",
		"--tool-args-json", `{"extra":true}`, "--files", cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}

	name, _ := gotParams["name"].(string)
	if name != "echo" {
		t.Errorf("tool name = %q, want echo", name)
	}
	arguments, _ := gotParams["arguments"].(map[string]any)
	if arguments["text"] != "hello" {
		t.Errorf("argument text = %v, want hello", arguments["text"])
	}
	if arguments["port"] != float64(3000) {
		t.Errorf("argument port = %v (%T), want 3000", arguments["port"], arguments["port"])
	}
	if arguments["extra"] != true {
		t.Errorf("argument extra = %v, want true", arguments["extra"])
	}

	var got struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("stdout %q is not clean JSON: %v", stdout, err)
	}
	if len(got.Result.Content) != 1 || got.Result.Content[0].Text != "ok" {
		t.Errorf("unexpected content: %+v", got.Result.Content)
	}
}

func TestMCPInspectStdio(t *testing.T) {
	cfg := writeMCPServerConfig(t, map[string]any{
		"demo-stdio": map[string]any{
			"command": os.Args[0],
			"args":    []string{"-test.run=^TestMCPInspectorHelperProcess$"},
			"env":     map[string]string{"SYMSCOPE_MCP_INSPECT_HELPER_ECHO": "env-reached-child"},
		},
	})
	stdout, stderr, code := runCLIProcess(t, []string{
		"HOME=" + t.TempDir(),
		"SYMSCOPE_MCP_INSPECT_HELPER=1",
	}, "mcp", "inspect", "--name", "demo-stdio", "--method", "tools/list", "--files", cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	var got struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
			EchoEnv string `json:"echo_env"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("stdout %q is not clean JSON: %v", stdout, err)
	}
	if len(got.Result.Tools) != 1 || got.Result.Tools[0].Name != "demo-tool" {
		t.Errorf("unexpected tools: %+v", got.Result.Tools)
	}
	if got.Result.EchoEnv != "env-reached-child" {
		t.Errorf("child env echo = %q, want env-reached-child (server env must reach the child)", got.Result.EchoEnv)
	}
}

func TestMCPInspectValidation(t *testing.T) {
	isolateCLIHome(t)
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"missing name", []string{"mcp", "inspect"}, "--name is required"},
		{"unsupported method", []string{"mcp", "inspect", "--name", "demo", "--method", "resources/list"}, `unsupported method "resources/list"`},
		{"tools/call without tool name", []string{"mcp", "inspect", "--name", "demo", "--method", "tools/call"}, "--tool-name is required"},
		{"malformed tool arg", []string{"mcp", "inspect", "--name", "demo", "--method", "tools/call", "--tool-name", "echo", "--tool-arg", "noequals"}, `--tool-arg "noequals" must be key=value`},
		{"malformed tool args json", []string{"mcp", "inspect", "--name", "demo", "--method", "tools/call", "--tool-name", "echo", "--tool-args-json", "{bad"}, "--tool-args-json"},
		{"unknown server", []string{"mcp", "inspect", "--name", "missing", "--method", "tools/list"}, `no MCP server named "missing"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectCLIError(t, tt.args, exitcodes.ExitConfig, tt.wantErr)
		})
	}
}
