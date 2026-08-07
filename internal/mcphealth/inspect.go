package mcphealth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// Inspect sends one JSON-RPC request to a discovered server over its
// transport (stdio for commands, HTTP POST for http/sse URLs — the same
// dispatch as ProbeAll) and returns the structured response. A JSON-RPC
// error response from the server is surfaced in the result, not as a Go
// error; only transport-level failures (spawn, timeout, unparseable body)
// return an error.
func Inspect(s model.MCPServer, method string, params map[string]any) (model.MCPInspectResult, error) {
	if s.Transport == "http" || s.Transport == "sse" {
		return InspectHTTP(s.URL, method, params)
	}
	return InspectStdio(s, method, params)
}

// InspectStdio runs one JSON-RPC method against a stdio server. The server's
// Env map is merged into the child environment (config entries win).
func InspectStdio(s model.MCPServer, method string, params map[string]any) (model.MCPInspectResult, error) {
	if s.Command == "" {
		return model.MCPInspectResult{}, errors.New("no command")
	}
	body, elapsed, err := rpcStdio(s.Command, s.Args, s.Env, buildRequest(method, params))
	if err != nil {
		return model.MCPInspectResult{}, err
	}
	return parseInspectResponse(method, body, elapsed)
}

// InspectHTTP POSTs one JSON-RPC method to an HTTP MCP server endpoint.
func InspectHTTP(url, method string, params map[string]any) (model.MCPInspectResult, error) {
	if url == "" {
		return model.MCPInspectResult{}, errors.New("no URL")
	}
	body, elapsed, err := rpcHTTP(url, buildRequest(method, params))
	if err != nil {
		return model.MCPInspectResult{}, err
	}
	return parseInspectResponse(method, body, elapsed)
}

// buildRequest assembles a JSON-RPC 2.0 request. The params field is omitted
// when nil so method-only requests (e.g. tools/list) stay minimal.
func buildRequest(method string, params map[string]any) map[string]any {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	return req
}

// parseInspectResponse decodes a raw JSON-RPC response into a structured
// inspect result, preserving the server's result/error payloads verbatim.
func parseInspectResponse(method string, body []byte, elapsed int64) (model.MCPInspectResult, error) {
	var rpcResp map[string]json.RawMessage
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return model.MCPInspectResult{}, fmt.Errorf("parse: %w", err)
	}
	result := model.MCPInspectResult{Method: method, LatencyMs: elapsed}
	if raw, ok := rpcResp["result"]; ok {
		result.Result = raw
	}
	if raw, ok := rpcResp["error"]; ok {
		result.Error = raw
	}
	return result, nil
}
