package watch_test

import (
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/watch"
)

func TestDiffPorts(t *testing.T) {
	oldSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 123, Process: "node", Address: "127.0.0.1"},
		},
	}
	newSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 123, Process: "node", Address: "127.0.0.1"},
			{Port: 9090, Protocol: "tcp", PID: 456, Process: "python", Address: "0.0.0.0"},
		},
	}

	events := watch.Diff(oldSnap, newSnap)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "port_bound" {
		t.Errorf("expected port_bound, got %s", events[0].Type)
	}

	events = watch.Diff(newSnap, oldSnap)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "port_unbound" {
		t.Errorf("expected port_unbound, got %s", events[0].Type)
	}
}

func TestDiffMCPServers(t *testing.T) {
	oldSnap := model.Snapshot{
		MCPServers: []model.MCPServer{
			{Name: "server-a", Client: "claude", Transport: "stdio", Command: "npm"},
		},
	}
	newSnap := model.Snapshot{
		MCPServers: []model.MCPServer{
			{Name: "server-a", Client: "claude", Transport: "stdio", Command: "npm", Args: []string{"run", "dev"}},
			{Name: "server-b", Client: "cursor", Transport: "stdio", Command: "python"},
		},
	}

	events := watch.Diff(oldSnap, newSnap)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	types := map[string]bool{}
	for _, e := range events {
		types[e.Type] = true
	}

	if !types["mcp_server_changed"] {
		t.Errorf("expected mcp_server_changed")
	}
	if !types["mcp_server_added"] {
		t.Errorf("expected mcp_server_added")
	}
}
