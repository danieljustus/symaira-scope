package scan

import (
	"errors"
	"testing"
	"time"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestBuildReturnsSnapshot(t *testing.T) {
	// Stub all collectors so the test is hermetic (no OS/Docker dependencies).
	origPorts := listListeningFn
	origDiscover := discoverFn
	origContainers := containersFn
	defer func() {
		listListeningFn = origPorts
		discoverFn = origDiscover
		containersFn = origContainers
	}()

	listListeningFn = func() ([]model.Port, error) {
		return []model.Port{
			{Port: 8080, Protocol: "tcp", Address: "127.0.0.1", PID: 1, Process: "node"},
		}, nil
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return []model.MCPServer{
			{Name: "vault", Client: "test", Transport: "stdio"},
		}, []string{"note from mcp"}
	}
	containersFn = func() ([]model.Container, []string) {
		return []model.Container{
			{ID: "abc123", Name: "web", Image: "nginx", Ports: []int{80}},
		}, []string{"note from containers"}
	}

	snap, err := Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	// GeneratedAt should be a valid RFC3339 timestamp.
	if _, err := time.Parse(time.RFC3339, snap.GeneratedAt); err != nil {
		t.Errorf("GeneratedAt is not valid RFC3339: %q", snap.GeneratedAt)
	}

	if len(snap.Ports) != 1 || snap.Ports[0].Port != 8080 {
		t.Errorf("ports: want [8080], got %v", snap.Ports)
	}
	if len(snap.MCPServers) != 1 || snap.MCPServers[0].Name != "vault" {
		t.Errorf("mcpServers: want [vault], got %v", snap.MCPServers)
	}
	if len(snap.Containers) != 1 || snap.Containers[0].Name != "web" {
		t.Errorf("containers: want [web], got %v", snap.Containers)
	}
}

func TestBuildMergesNotes(t *testing.T) {
	origPorts := listListeningFn
	origDiscover := discoverFn
	origContainers := containersFn
	defer func() {
		listListeningFn = origPorts
		discoverFn = origDiscover
		containersFn = origContainers
	}()

	listListeningFn = func() ([]model.Port, error) { return nil, nil }
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return nil, []string{"mcp-note-1", "mcp-note-2"}
	}
	containersFn = func() ([]model.Container, []string) {
		return nil, []string{"cont-note-1"}
	}

	snap, err := Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	if len(snap.Notes) != 3 {
		t.Fatalf("expected 3 merged notes, got %d: %v", len(snap.Notes), snap.Notes)
	}
}

func TestBuildPropagatesPortsError(t *testing.T) {
	origPorts := listListeningFn
	origDiscover := discoverFn
	origContainers := containersFn
	defer func() {
		listListeningFn = origPorts
		discoverFn = origDiscover
		containersFn = origContainers
	}()

	portsErr := errors.New("port enumeration failed")
	listListeningFn = func() ([]model.Port, error) { return nil, portsErr }
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return nil, nil
	}
	containersFn = func() ([]model.Container, []string) { return nil, nil }

	_, err := Build()
	if err == nil {
		t.Fatal("expected error when ports collector fails, got nil")
	}
	if !errors.Is(err, portsErr) {
		t.Errorf("expected ports error to propagate, got: %v", err)
	}
}

func TestBuildEmptyCollectors(t *testing.T) {
	origPorts := listListeningFn
	origDiscover := discoverFn
	origContainers := containersFn
	defer func() {
		listListeningFn = origPorts
		discoverFn = origDiscover
		containersFn = origContainers
	}()

	listListeningFn = func() ([]model.Port, error) { return nil, nil }
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) { return nil, nil }
	containersFn = func() ([]model.Container, []string) { return nil, nil }

	snap, err := Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(snap.Ports) != 0 || len(snap.MCPServers) != 0 || len(snap.Containers) != 0 {
		t.Errorf("expected empty snapshot, got ports=%d servers=%d containers=%d",
			len(snap.Ports), len(snap.MCPServers), len(snap.Containers))
	}
}
