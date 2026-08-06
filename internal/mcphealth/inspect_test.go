package mcphealth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestInspectHTTP_ToolsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"demo-tool","description":"a demo tool","inputSchema":{"type":"object"}}]}}`))
	}))
	defer srv.Close()

	result, err := InspectHTTP(srv.URL, "tools/list", nil)
	if err != nil {
		t.Fatalf("InspectHTTP: %v", err)
	}
	if result.Method != "tools/list" {
		t.Errorf("method = %q, want tools/list", result.Method)
	}
	if len(result.Result) == 0 {
		t.Fatal("expected a result payload")
	}
	var payload struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result.Result, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Tools) != 1 || payload.Tools[0].Name != "demo-tool" {
		t.Errorf("unexpected tools payload: %+v", payload.Tools)
	}
	if len(result.Error) != 0 {
		t.Errorf("unexpected error payload: %s", result.Error)
	}
}

func TestInspectHTTP_JSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"unknown tool"}}`))
	}))
	defer srv.Close()

	result, err := InspectHTTP(srv.URL, "tools/call", map[string]any{"name": "nope", "arguments": map[string]any{}})
	if err != nil {
		t.Fatalf("a JSON-RPC error response must be surfaced in the result, not as a Go error: %v", err)
	}
	if len(result.Error) == 0 {
		t.Fatal("expected an error payload")
	}
	var rpcErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Error, &rpcErr); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if rpcErr.Message != "unknown tool" {
		t.Errorf("error message = %q, want %q", rpcErr.Message, "unknown tool")
	}
	if len(result.Result) != 0 {
		t.Errorf("unexpected result payload: %s", result.Result)
	}
}

func TestInspectHTTP_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	if _, err := InspectHTTP(srv.URL, "tools/list", nil); err == nil {
		t.Fatal("expected an error for an unparseable body")
	}
}

func TestInspectHTTP_EmptyURL(t *testing.T) {
	if _, err := InspectHTTP("", "tools/list", nil); err == nil {
		t.Fatal("expected an error for an empty URL")
	}
}

func TestInspectHTTP_Unreachable(t *testing.T) {
	if _, err := InspectHTTP("http://127.0.0.1:1", "tools/list", nil); err == nil {
		t.Fatal("expected an error for an unreachable server")
	}
}

func TestInspectStdio_ToolsList(t *testing.T) {
	cmd, args := helperInvocation(t, "inspect", "")
	result, err := InspectStdio(model.MCPServer{Command: cmd, Args: args}, "tools/list", nil)
	if err != nil {
		t.Fatalf("InspectStdio: %v", err)
	}
	if result.Method != "tools/list" {
		t.Errorf("method = %q, want tools/list", result.Method)
	}
	var payload struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(result.Result, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Method != "tools/list" {
		t.Errorf("echoed method = %q, want tools/list", payload.Method)
	}
	if len(payload.Params) != 0 {
		t.Errorf("tools/list must not carry params, got %s", payload.Params)
	}
}

func TestInspectStdio_ToolsCall(t *testing.T) {
	cmd, args := helperInvocation(t, "inspect", "")
	params := map[string]any{
		"name": "echo",
		"arguments": map[string]any{
			"text": "hello",
			"port": 3000,
		},
	}
	result, err := InspectStdio(model.MCPServer{Command: cmd, Args: args}, "tools/call", params)
	if err != nil {
		t.Fatalf("InspectStdio: %v", err)
	}
	var payload struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(result.Result, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Method != "tools/call" {
		t.Errorf("echoed method = %q, want tools/call", payload.Method)
	}
	var echoed struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(payload.Params, &echoed); err != nil {
		t.Fatalf("unmarshal echoed params: %v", err)
	}
	if echoed.Name != "echo" {
		t.Errorf("tool name = %q, want echo", echoed.Name)
	}
	if got := echoed.Arguments["text"]; got != "hello" {
		t.Errorf("argument text = %v, want hello", got)
	}
	if got := echoed.Arguments["port"]; got != float64(3000) {
		t.Errorf("argument port = %v (%T), want 3000", got, got)
	}
}

func TestInspectStdio_EnvPropagation(t *testing.T) {
	cmd, args := helperInvocation(t, "inspect", "")
	result, err := InspectStdio(model.MCPServer{
		Command: cmd,
		Args:    args,
		Env:     map[string]string{helperEnvEcho: "env-reached-child"},
	}, "tools/list", nil)
	if err != nil {
		t.Fatalf("InspectStdio: %v", err)
	}
	var payload struct {
		EchoEnv string `json:"echo_env"`
	}
	if err := json.Unmarshal(result.Result, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.EchoEnv != "env-reached-child" {
		t.Errorf("child env echo = %q, want %q", payload.EchoEnv, "env-reached-child")
	}
}

func TestInspectStdio_EmptyCommand(t *testing.T) {
	if _, err := InspectStdio(model.MCPServer{}, "tools/list", nil); err == nil {
		t.Fatal("expected an error for an empty command")
	}
}

func TestInspect_TransportDispatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
	}))
	defer srv.Close()

	for _, transport := range []string{"http", "sse"} {
		result, err := Inspect(model.MCPServer{Name: "demo", Transport: transport, URL: srv.URL}, "tools/list", nil)
		if err != nil {
			t.Fatalf("Inspect(%s): %v", transport, err)
		}
		if !strings.Contains(string(result.Result), `"ok"`) {
			t.Errorf("Inspect(%s) result = %s, want ok payload", transport, result.Result)
		}
	}

	if _, err := Inspect(model.MCPServer{Name: "demo", Transport: "stdio"}, "tools/list", nil); err == nil {
		t.Error("expected an error for a stdio server without a command")
	}
}
