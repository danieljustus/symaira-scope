package mcphealth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestProbeHTTP_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"test","version":"1.0"}}}`))
	}))
	defer srv.Close()

	result := ProbeHTTP(srv.URL)
	if result.Status != "healthy" {
		t.Errorf("expected healthy, got %s (error: %s)", result.Status, result.Error)
	}
	if result.LatencyMs < 0 {
		t.Errorf("latency should be non-negative, got %d", result.LatencyMs)
	}
}

func TestProbeHTTP_NoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"bad"}}`))
	}))
	defer srv.Close()

	result := ProbeHTTP(srv.URL)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestProbeHTTP_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	result := ProbeHTTP(srv.URL)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestProbeHTTP_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	result := ProbeHTTP(srv.URL)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestProbeHTTP_EmptyURL(t *testing.T) {
	result := ProbeHTTP("")
	if result.Status != "unknown" {
		t.Errorf("expected unknown for empty url, got %s", result.Status)
	}
}

func TestProbeHTTP_Unreachable(t *testing.T) {
	result := ProbeHTTP("http://127.0.0.1:1")
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy for unreachable, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for unreachable server")
	}
}

func TestProbeStdio_EmptyCommand(t *testing.T) {
	result := ProbeStdio("", nil)
	if result.Status != "unknown" {
		t.Errorf("expected unknown for empty command, got %s", result.Status)
	}
}

func TestProbeStdio_BadCommand(t *testing.T) {
	result := ProbeStdio("nonexistent-binary-xyz", nil)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy for bad command, got %s", result.Status)
	}
}

func TestProbeAll_MixedServers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"test","version":"1.0"}}}`))
	}))
	defer srv.Close()

	servers := []model.MCPServer{
		{Name: "http-server", Client: "test", Transport: "http", URL: srv.URL},
		{Name: "empty-stdio", Client: "test", Transport: "stdio", Command: ""},
		{Name: "bad-stdio", Client: "test", Transport: "stdio", Command: "nonexistent-binary-xyz"},
	}

	results := ProbeAll(servers)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Status != "healthy" {
		t.Errorf("http-server: expected healthy, got %s", results[0].Status)
	}
	if results[1].Status != "unknown" {
		t.Errorf("empty-stdio: expected unknown, got %s", results[1].Status)
	}
	if results[2].Status != "unhealthy" {
		t.Errorf("bad-stdio: expected unhealthy, got %s", results[2].Status)
	}

	for _, r := range results {
		if r.Name == "" {
			t.Error("expected name to be set")
		}
		if r.Client == "" {
			t.Error("expected client to be set")
		}
	}
}
