package mcphealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
)

var probeTimeout = 5 * time.Second

type jsonrpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  requestParams `json:"params"`
}

type requestParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      clientInfo     `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func initRequest() jsonrpcRequest {
	return jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: requestParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    map[string]any{},
			ClientInfo:      clientInfo{Name: "symscope", Version: "0.2.0"},
		},
	}
}

func ProbeAll(servers []model.MCPServer) []model.MCPHealthResult {
	results := make([]model.MCPHealthResult, len(servers))
	for i, s := range servers {
		switch s.Transport {
		case "http", "sse":
			results[i] = ProbeHTTP(s.URL)
		default:
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
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("write: %v", err), LatencyMs: time.Since(start).Milliseconds()}
	}
	stdin.Close()

	respBytes, err := io.ReadAll(stdout)
	elapsed := time.Since(start).Milliseconds()

	killProcess(c)

	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("read: %v", err), LatencyMs: elapsed}
	}

	var resp map[string]any
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("invalid json response: %v", err), LatencyMs: elapsed}
	}

	if _, ok := resp["result"]; ok {
		return model.MCPHealthResult{Status: "healthy", LatencyMs: elapsed}
	}

	return model.MCPHealthResult{Status: "unhealthy", Error: "no result field in response", LatencyMs: elapsed}
}

func ProbeHTTP(url string) model.MCPHealthResult {
	if url == "" {
		return model.MCPHealthResult{Status: "unknown", Error: "no url"}
	}

	req := initRequest()
	reqBytes, _ := json.Marshal(req)

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBytes))
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("request: %v", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("post: %v", err), LatencyMs: elapsed}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("read body: %v", err), LatencyMs: elapsed}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("status %d", resp.StatusCode), LatencyMs: elapsed}
	}

	var rpcResp map[string]any
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return model.MCPHealthResult{Status: "unhealthy", Error: fmt.Sprintf("invalid json: %v", err), LatencyMs: elapsed}
	}

	if _, ok := rpcResp["result"]; ok {
		return model.MCPHealthResult{Status: "healthy", LatencyMs: elapsed}
	}

	return model.MCPHealthResult{Status: "unhealthy", Error: "no result field in response", LatencyMs: elapsed}
}

func killProcess(c *exec.Cmd) {
	if runtime.GOOS == "windows" {
		c.Process.Kill()
	} else {
		syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
	}
	c.Wait()
}
