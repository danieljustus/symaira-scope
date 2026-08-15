package watch

import (
	"reflect"
	"testing"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestConflictKey(t *testing.T) {
	tests := []struct {
		name string
		c    model.Conflict
		want string
	}{
		{
			name: "process conflict",
			c:    model.Conflict{Port: 8080, Holders: []string{"alpha(pid 111)", "beta(pid 222)"}},
			want: "8080/",
		},
		{
			name: "mcp occupied",
			c:    model.Conflict{Port: 9090, Holders: []string{"claude/s1"}, Kind: "mcp-occupied"},
			want: "9090/mcp-occupied",
		},
		{
			name: "port zero",
			c:    model.Conflict{Port: 0, Kind: "mcp-occupied"},
			want: "0/mcp-occupied",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conflictKey(tt.c); got != tt.want {
				t.Errorf("conflictKey(%+v) = %q, want %q", tt.c, got, tt.want)
			}
		})
	}
}

func TestDiffConflictDetected(t *testing.T) {
	oldSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
		},
	}
	newSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
			{Port: 8080, Protocol: "tcp", PID: 222, Process: "beta", Address: "127.0.0.1"},
		},
	}

	events := eventsByType(Diff(oldSnap, newSnap), "conflict_detected")
	if len(events) != 1 {
		t.Fatalf("expected 1 conflict_detected event, got %d", len(events))
	}
	assertConflictEvent(t, events[0], model.Conflict{
		Port:    8080,
		Holders: []string{"alpha(pid 111)", "beta(pid 222)"},
	})
}

func TestDiffConflictChanged(t *testing.T) {
	oldSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
			{Port: 8080, Protocol: "tcp", PID: 222, Process: "beta", Address: "127.0.0.1"},
		},
	}
	newSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
			{Port: 8080, Protocol: "tcp", PID: 333, Process: "gamma", Address: "127.0.0.1"},
		},
	}

	events := eventsByType(Diff(oldSnap, newSnap), "conflict_changed")
	if len(events) != 1 {
		t.Fatalf("expected 1 conflict_changed event, got %d", len(events))
	}
	assertConflictEvent(t, events[0], model.Conflict{
		Port:    8080,
		Holders: []string{"alpha(pid 111)", "gamma(pid 333)"},
	})
}

func TestDiffConflictResolved(t *testing.T) {
	oldSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
			{Port: 8080, Protocol: "tcp", PID: 222, Process: "beta", Address: "127.0.0.1"},
		},
	}
	newSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
		},
	}

	events := eventsByType(Diff(oldSnap, newSnap), "conflict_resolved")
	if len(events) != 1 {
		t.Fatalf("expected 1 conflict_resolved event, got %d", len(events))
	}
	// conflict_resolved carries the OLD conflict as payload.
	assertConflictEvent(t, events[0], model.Conflict{
		Port:    8080,
		Holders: []string{"alpha(pid 111)", "beta(pid 222)"},
	})
}

func TestDiffConflictDetectedMCPServerOccupied(t *testing.T) {
	oldSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 9000, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
		},
		MCPServers: []model.MCPServer{
			{Name: "s1", Client: "claude", URL: "http://127.0.0.1:9000"},
		},
	}
	newSnap := model.Snapshot{
		Ports: []model.Port{
			{Port: 9000, Protocol: "tcp", PID: 111, Process: "alpha", Address: "127.0.0.1"},
		},
		MCPServers: []model.MCPServer{
			{Name: "s1", Client: "claude", URL: "http://127.0.0.1:9000"},
			{Name: "s2", Client: "cursor", URL: "http://127.0.0.1:9000"},
		},
	}

	events := eventsByType(Diff(oldSnap, newSnap), "conflict_detected")
	if len(events) != 1 {
		t.Fatalf("expected 1 conflict_detected event, got %d", len(events))
	}
	assertConflictEvent(t, events[0], model.Conflict{
		Port:    9000,
		Holders: []string{"claude/s1 (occupied by alpha)", "cursor/s2 (occupied by alpha)"},
		Kind:    "mcp-occupied",
	})
}

func eventsByType(events []Event, typ string) []Event {
	var out []Event
	for _, e := range events {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func assertConflictEvent(t *testing.T, e Event, want model.Conflict) {
	t.Helper()
	got, ok := e.Payload.(model.Conflict)
	if !ok {
		t.Fatalf("payload is %T, want model.Conflict", e.Payload)
	}
	if got.Port != want.Port || got.Kind != want.Kind || !reflect.DeepEqual(got.Holders, want.Holders) {
		t.Errorf("conflict payload = %+v, want %+v", got, want)
	}
}
