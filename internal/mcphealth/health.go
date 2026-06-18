// Package mcphealth probes MCP servers to check if they respond to an
// initialize request.
//
// Trust model: symscope reads MCP server configs from well-known local paths.
// When --probe is used, it executes the commands and URLs found in those
// configs. This is safe because symscope is a local-only tool that trusts its
// own config files — the same trust model as the AI clients themselves.
// Malicious configs could execute arbitrary binaries, but the user must have
// already installed those configs.
package mcphealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
)

const probeTimeout = 5 * time.Second

func ProbeAll(servers []model.MCPServer) []model.MCPHealthResult {
	results := make([]model.MCPHealthResult, len(servers))
	for i, s := range servers {
		if s.Transport == "http" || s.Transport == "sse" {
			results[i] = ProbeHTTP(s.URL)
		} else {
			results[i] = ProbeStdio(s.Command, s.Args)
		}
		results[i].Name = s.Name
		results[i].Client = s.Client
	}
	return results
}

func ProbeStdio(cmd string, args []string) model.MCPHealthResult {
	if cmd == "" {
		return model.MCPHealthResult{Status: "unknown", Error: "no command"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, args...)
	setProcAttr(c)

	stdin, err := c.StdinPipe()
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("stdin pipe: %v", err)}
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("stdout pipe: %v", err)}
	}

	if err := c.Start(); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("start: %v", err)}
	}

	req := initRequest()
	reqBytes, _ := json.Marshal(req)

	start := time.Now()
	if _, err := stdin.Write(reqBytes); err != nil {
		killProcess(c)
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("write: %v", err)}
	}
	stdin.Close()

	var buf bytes.Buffer
	io.Copy(&buf, stdout)
	elapsed := time.Since(start).Milliseconds()

	killProcess(c)

	if ctx.Err() == context.DeadlineExceeded {
		return model.MCPHealthResult{Status: "unhealthy", Error: "timeout", LatencyMs: elapsed}
	}

	var rpcResp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rpcResp); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("parse: %v", err), LatencyMs: elapsed}
	}

	if _, ok := rpcResp["result"]; ok {
		return model.MCPHealthResult{Status: "healthy", LatencyMs: elapsed}
	}

	return model.MCPHealthResult{Status: "unhealthy", Error: "no result field in response", LatencyMs: elapsed}
}

func ProbeHTTP(url string) model.MCPHealthResult {
	if url == "" {
		return model.MCPHealthResult{Status: "unknown", Error: "no URL"}
	}

	req := initRequest()
	reqBytes, _ := json.Marshal(req)

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("request: %v", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("do: %v", err), LatencyMs: elapsed}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rpcResp map[string]any
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("parse: %v", err), LatencyMs: elapsed}
	}

	if _, ok := rpcResp["result"]; ok {
		return model.MCPHealthResult{Status: "healthy", LatencyMs: elapsed}
	}

	return model.MCPHealthResult{Status: "unhealthy", Error: "no result field in response", LatencyMs: elapsed}
}

func initRequest() map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "symscope",
				"version": "0.2.0",
			},
		},
	}
}
