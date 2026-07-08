package watch

import (
	"fmt"
	"reflect"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

// Event represents a change in the environment detected between two snapshots.
type Event struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Payload   any    `json:"payload"`
}

// Diff compares two snapshots and returns a slice of events representing the changes.
func Diff(old, new model.Snapshot) []Event {
	var events []Event
	now := time.Now().UTC().Format(time.RFC3339)

	// Ports
	oldPorts := make(map[string]model.Port)
	for _, p := range old.Ports {
		oldPorts[portKey(p)] = p
	}
	newPorts := make(map[string]model.Port)
	for _, p := range new.Ports {
		newPorts[portKey(p)] = p
	}

	for key, np := range newPorts {
		if _, ok := oldPorts[key]; !ok {
			events = append(events, Event{Type: "port_bound", Timestamp: now, Payload: np})
		}
	}
	for key, op := range oldPorts {
		if _, ok := newPorts[key]; !ok {
			events = append(events, Event{Type: "port_unbound", Timestamp: now, Payload: op})
		}
	}

	// Conflicts
	oldConflicts := ports.Conflicts(old.Ports)
	oldConflicts = append(oldConflicts, ports.MCPServerConflicts(old.MCPServers, old.Ports)...)
	newConflicts := ports.Conflicts(new.Ports)
	newConflicts = append(newConflicts, ports.MCPServerConflicts(new.MCPServers, new.Ports)...)

	oldConfMap := make(map[string]model.Conflict)
	for _, c := range oldConflicts {
		oldConfMap[conflictKey(c)] = c
	}
	newConfMap := make(map[string]model.Conflict)
	for _, c := range newConflicts {
		newConfMap[conflictKey(c)] = c
	}

	for key, nc := range newConfMap {
		oc, ok := oldConfMap[key]
		if !ok {
			events = append(events, Event{Type: "conflict_detected", Timestamp: now, Payload: nc})
		} else if !reflect.DeepEqual(nc.Holders, oc.Holders) {
			events = append(events, Event{Type: "conflict_changed", Timestamp: now, Payload: nc})
		}
	}
	for key, oc := range oldConfMap {
		if _, ok := newConfMap[key]; !ok {
			events = append(events, Event{Type: "conflict_resolved", Timestamp: now, Payload: oc})
		}
	}

	// MCP Servers
	oldMCP := make(map[string]model.MCPServer)
	for _, s := range old.MCPServers {
		oldMCP[mcpKey(s)] = s
	}
	newMCP := make(map[string]model.MCPServer)
	for _, s := range new.MCPServers {
		newMCP[mcpKey(s)] = s
	}

	for key, ns := range newMCP {
		os, ok := oldMCP[key]
		if !ok {
			events = append(events, Event{Type: "mcp_server_added", Timestamp: now, Payload: ns})
		} else if !reflect.DeepEqual(ns, os) {
			events = append(events, Event{Type: "mcp_server_changed", Timestamp: now, Payload: ns})
		}
	}
	for key, os := range oldMCP {
		if _, ok := newMCP[key]; !ok {
			events = append(events, Event{Type: "mcp_server_removed", Timestamp: now, Payload: os})
		}
	}

	// Containers
	oldCont := make(map[string]model.Container)
	for _, c := range old.Containers {
		oldCont[c.ID] = c
	}
	newCont := make(map[string]model.Container)
	for _, c := range new.Containers {
		newCont[c.ID] = c
	}

	for key, nc := range newCont {
		if _, ok := oldCont[key]; !ok {
			events = append(events, Event{Type: "container_started", Timestamp: now, Payload: nc})
		}
	}
	for key, oc := range oldCont {
		if _, ok := newCont[key]; !ok {
			events = append(events, Event{Type: "container_stopped", Timestamp: now, Payload: oc})
		}
	}

	return events
}

func portKey(p model.Port) string {
	return fmt.Sprintf("%d/%s/%d/%s", p.Port, p.Protocol, p.PID, p.Address)
}

func conflictKey(c model.Conflict) string {
	return fmt.Sprintf("%d/%s", c.Port, c.Kind)
}

func mcpKey(s model.MCPServer) string {
	return fmt.Sprintf("%s/%s", s.Client, s.Name)
}
