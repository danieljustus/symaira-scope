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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/version"
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

	body, elapsed, err := rpcStdio(cmd, args, nil, initRequest())
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: err.Error(), LatencyMs: elapsed}
	}

	var rpcResp map[string]any
	if err := json.Unmarshal(body, &rpcResp); err != nil {
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

	body, elapsed, err := rpcHTTP(url, initRequest())
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: err.Error(), LatencyMs: elapsed}
	}

	var rpcResp map[string]any
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("parse: %v", err), LatencyMs: elapsed}
	}

	if _, ok := rpcResp["result"]; ok {
		return model.MCPHealthResult{Status: "healthy", LatencyMs: elapsed}
	}

	return model.MCPHealthResult{Status: "unhealthy", Error: "no result field in response", LatencyMs: elapsed}
}

// rpcStdio runs one JSON-RPC request against a stdio MCP server subprocess and
// returns the raw response bytes and latency. env, when non-empty, is merged
// into the child's environment (config entries win over the parent env).
// Transport-level failures (spawn, pipe, write, timeout) are returned as
// errors; response parsing is the caller's job.
func rpcStdio(cmd string, args []string, env map[string]string, req map[string]any) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, args...)
	setProcAttr(c)
	if len(env) > 0 {
		childEnv := os.Environ()
		for k, v := range env {
			childEnv = append(childEnv, k+"="+v)
		}
		c.Env = childEnv
	}

	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := c.Start(); err != nil {
		return nil, 0, fmt.Errorf("start: %w", err)
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		killProcess(c)
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	start := time.Now()
	if _, err := stdin.Write(reqBytes); err != nil {
		killProcess(c)
		return nil, 0, fmt.Errorf("write: %w", err)
	}
	stdin.Close()

	var buf bytes.Buffer
	io.Copy(&buf, stdout)
	elapsed := time.Since(start).Milliseconds()

	killProcess(c)

	if ctx.Err() == context.DeadlineExceeded {
		return nil, elapsed, errors.New("timeout")
	}
	return buf.Bytes(), elapsed, nil
}

// rpcHTTP POSTs one JSON-RPC request to an HTTP MCP server endpoint and
// returns the raw response bytes and latency. Transport-level failures are
// returned as errors; response parsing is the caller's job.
func rpcHTTP(url string, req map[string]any) ([]byte, int64, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return nil, elapsed, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, elapsed, fmt.Errorf("read body: %w", err)
	}
	return body, elapsed, nil
}

func initRequest() map[string]any {
	return buildRequest("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "symscope",
			"version": version.Version,
		},
	})
}
