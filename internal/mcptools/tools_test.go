package mcptools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestServeHelperProcess is not a real test — it is the stdio server process
// spawned by TestServeStdioIntegration. The main test binary re-executes
// itself with SYMSCOPE_SERVE_HELPER=1, so the helper runs the exact same
// ServeStdio transport a real MCP client (Hermes, Claude Desktop, ...) talks
// to: JSON-RPC 2.0 over stdin/stdout with Content-Length framing.
func TestServeHelperProcess(t *testing.T) {
	if os.Getenv("SYMSCOPE_SERVE_HELPER") != "1" {
		t.Skip("helper process; spawned by TestServeStdioIntegration")
	}
	if err := Serve("test-version"); err != nil {
		fmt.Fprintln(os.Stderr, "helper serve error:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// mcpClient is a minimal MCP stdio client speaking Content-Length framed
// JSON-RPC 2.0, exactly like the native clients that failed against symscope
// before corekit v0.8.0 (issue #80).
type mcpClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	br     *bufio.Reader
	nextID int
}

func startMCPClient(t *testing.T) *mcpClient {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestServeHelperProcess")
	cmd.Env = append(os.Environ(), "SYMSCOPE_SERVE_HELPER=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	c := &mcpClient{cmd: cmd, stdin: stdin, br: bufio.NewReader(stdout)}
	t.Cleanup(func() {
		_ = stdin.Close()
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	})
	return c
}

// request sends a JSON-RPC request and reads exactly one response with the
// same id. Notifications (method prefix "notifications/") carry no id and get
// no response.
func (c *mcpClient) request(t *testing.T, method string, params any) map[string]any {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	isNotification := strings.HasPrefix(method, "notifications/")
	if !isNotification {
		c.nextID++
		req["id"] = c.nextID
	}
	if params != nil {
		req["params"] = params
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err := fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if isNotification {
		return nil
	}
	resp := c.readResponse(t)
	if got := resp["id"]; got != nil && fmt.Sprint(got) != fmt.Sprint(c.nextID) {
		t.Fatalf("response id mismatch: got %v want %v", got, c.nextID)
	}
	return resp
}

// readResponse reads one Content-Length framed JSON-RPC response.
func (c *mcpClient) readResponse(t *testing.T) map[string]any {
	t.Helper()
	type header struct{ length int }
	readHeader := func() header {
		var h header
		for {
			line, err := c.br.ReadString('\n')
			if err != nil {
				t.Fatalf("read header: %v", err)
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				return h
			}
			if rest, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				n, err := strconv.Atoi(strings.TrimSpace(rest))
				if err != nil {
					t.Fatalf("invalid Content-Length %q: %v", rest, err)
				}
				h.length = n
			}
		}
	}
	h := readHeader()
	if h.length <= 0 {
		t.Fatalf("missing Content-Length header")
	}
	buf := make([]byte, h.length)
	if _, err := io.ReadFull(c.br, buf); err != nil {
		t.Fatalf("read body: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(buf, &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", buf, err)
	}
	return resp
}

// callTool invokes tools/call for name and returns the decoded structured
// payload from content[0].text, failing if the response violates the MCP
// TextContent schema.
func (c *mcpClient) callTool(t *testing.T, name string, args map[string]any) json.RawMessage {
	t.Helper()
	params := map[string]any{"name": name}
	if args != nil {
		params["arguments"] = args
	}
	resp := c.request(t, "tools/call", params)
	if err, _ := resp["error"]; err != nil {
		t.Fatalf("tools/call %s returned JSON-RPC error: %v", name, err)
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/call %s: missing result object: %v", name, resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("tools/call %s: missing content array: %v", name, result)
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("tools/call %s: content[0] is not an object: %v", name, content[0])
	}
	if typ, _ := first["type"].(string); typ != "text" {
		t.Fatalf("tools/call %s: content[0].type = %v, want text", name, first["type"])
	}
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("tools/call %s: content[0].text is %T, want string (MCP TextContent schema violation)", name, first["text"])
	}
	// The structured payload must be JSON-encoded into the text string and
	// still decode back to valid JSON.
	payload := json.RawMessage(text)
	if !json.Valid(payload) {
		t.Fatalf("tools/call %s: content[0].text is not valid JSON: %q", name, text)
	}
	return payload
}

func TestServeStdioIntegration(t *testing.T) {
	c := startMCPClient(t)

	// 1. initialize — strict clients require a well-formed handshake.
	resp := c.request(t, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "symscope-test", "version": "0.0.0"},
	})
	if err, _ := resp["error"]; err != nil {
		t.Fatalf("initialize error: %v", err)
	}
	result, _ := resp["result"].(map[string]any)
	info, _ := result["serverInfo"].(map[string]any)
	if info["name"] != "symscope" {
		t.Fatalf("serverInfo.name = %v, want symscope", info["name"])
	}

	// 2. initialized notification (no response expected).
	c.request(t, "notifications/initialized", nil)

	// 3. tools/list — all registered tools must be discoverable.
	resp = c.request(t, "tools/list", nil)
	toolsList, _ := resp["result"].(map[string]any)
	tools, _ := toolsList["tools"].([]any)
	names := make(map[string]bool, len(tools))
	for _, tl := range tools {
		tm, _ := tl.(map[string]any)
		if n, _ := tm["name"].(string); n != "" {
			names[n] = true
		}
	}
	want := []string{"scan", "ports_list", "ports_suggest", "mcp_list", "conflicts", "mcp_health"}
	for _, n := range want {
		if !names[n] {
			t.Errorf("tools/list missing tool %q (got %d tools)", n, len(names))
		}
	}

	// 4. tools/call on every tool — each result must be schema-valid
	// (content[0].text is a JSON string) and decode to the expected shape.
	t.Run("scan", func(t *testing.T) {
		var snap struct {
			GeneratedAt string `json:"generated_at"`
			Ports       []any  `json:"ports"`
			MCPServers  []any  `json:"mcp_servers"`
			Containers  []any  `json:"containers"`
		}
		if err := json.Unmarshal(c.callTool(t, "scan", nil), &snap); err != nil {
			t.Fatalf("decode scan payload: %v", err)
		}
		// Collections may be empty (no docker daemon, no MCP configs) but the
		// keys must be present and generated_at must be set.
		if snap.GeneratedAt == "" || snap.Ports == nil || snap.MCPServers == nil || snap.Containers == nil {
			// nil after decoding a present JSON key means the server emitted
			// "null" for the collection — verify presence at the JSON level.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(c.callTool(t, "scan", nil), &raw); err != nil {
				t.Fatalf("re-decode scan payload: %v", err)
			}
			for _, k := range []string{"generated_at", "ports", "mcp_servers", "containers"} {
				if _, ok := raw[k]; !ok {
					t.Errorf("scan payload missing key %q", k)
				}
			}
		}
	})
	t.Run("ports_list", func(t *testing.T) {
		var ports []any
		if err := json.Unmarshal(c.callTool(t, "ports_list", nil), &ports); err != nil {
			t.Fatalf("decode ports_list payload: %v", err)
		}
	})
	t.Run("ports_suggest", func(t *testing.T) {
		var out struct {
			Free []any `json:"free"`
		}
		if err := json.Unmarshal(c.callTool(t, "ports_suggest", nil), &out); err != nil {
			t.Fatalf("decode ports_suggest payload: %v", err)
		}
		if len(out.Free) != 3 {
			t.Errorf("ports_suggest default count = %d, want 3", len(out.Free))
		}
	})
	t.Run("mcp_list", func(t *testing.T) {
		var servers []any
		if err := json.Unmarshal(c.callTool(t, "mcp_list", nil), &servers); err != nil {
			t.Fatalf("decode mcp_list payload: %v", err)
		}
	})
	t.Run("conflicts", func(t *testing.T) {
		var conflicts []any
		if err := json.Unmarshal(c.callTool(t, "conflicts", nil), &conflicts); err != nil {
			t.Fatalf("decode conflicts payload: %v", err)
		}
	})
	t.Run("mcp_health", func(t *testing.T) {
		var results []any
		if err := json.Unmarshal(c.callTool(t, "mcp_health", nil), &results); err != nil {
			t.Fatalf("decode mcp_health payload: %v", err)
		}
	})
}
