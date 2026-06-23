package explain

import (
	"testing"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestExplainPortFound(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) {
		return []model.Port{
			{Port: 8080, Protocol: "tcp", Address: "127.0.0.1", PID: 1, Process: "node"},
		}, nil
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return []model.MCPServer{
			{Name: "api", Client: "cursor", Transport: "http", URL: "http://localhost:8080"},
		}, nil
	}

	exp, err := ExplainPort(8080)
	if err != nil {
		t.Fatalf("ExplainPort: %v", err)
	}
	if exp.Port != 8080 {
		t.Errorf("port: want 8080, got %d", exp.Port)
	}
	if exp.Listening == nil {
		t.Fatal("expected Listening to be set")
	}
	if exp.Listening.Process != "node" {
		t.Errorf("process: want %q, got %q", "node", exp.Listening.Process)
	}
	if len(exp.MCPServers) != 1 || exp.MCPServers[0].Name != "api" {
		t.Errorf("mcpServers: want [api], got %v", exp.MCPServers)
	}
}

func TestExplainPortNotFound(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) {
		return nil, nil
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return nil, nil
	}

	exp, err := ExplainPort(9999)
	if err != nil {
		t.Fatalf("ExplainPort: %v", err)
	}
	if exp.Listening != nil {
		t.Error("expected Listening to be nil")
	}
	if len(exp.MCPServers) != 0 {
		t.Errorf("expected no MCPServers, got %v", exp.MCPServers)
	}
	if !exp.Suggested {
		t.Error("expected Suggested=true when nothing found")
	}
}

func TestExplainPortSkipsStdioServers(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) { return nil, nil }
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return []model.MCPServer{
			{Name: "stdio", Client: "c", Transport: "stdio", Command: "node"},
		}, nil
	}

	exp, err := ExplainPort(3000)
	if err != nil {
		t.Fatalf("ExplainPort: %v", err)
	}
	if len(exp.MCPServers) != 0 {
		t.Errorf("stdio servers should be skipped, got %v", exp.MCPServers)
	}
	if !exp.Suggested {
		t.Error("expected Suggested=true")
	}
}

func TestExplainPortListeningError(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) {
		return nil, &mockError{"port enum failed"}
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) { return nil, nil }

	_, err := ExplainPort(80)
	if err == nil {
		t.Fatal("expected error from ListListening")
	}
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

func TestExplainServerFound(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return []model.MCPServer{
			{Name: "myserver", Client: "cursor", Transport: "stdio", Command: "node"},
		}, nil
	}
	listListeningFn = func() ([]model.Port, error) { return nil, nil }

	exp, err := ExplainServer("myserver")
	if err != nil {
		t.Fatalf("ExplainServer: %v", err)
	}
	if exp.Name != "myserver" {
		t.Errorf("name: want %q, got %q", "myserver", exp.Name)
	}
	if exp.Client != "cursor" {
		t.Errorf("client: want %q, got %q", "cursor", exp.Client)
	}
	if exp.Server.Transport != "stdio" {
		t.Errorf("transport: want %q, got %q", "stdio", exp.Server.Transport)
	}
}

func TestExplainServerNotFound(t *testing.T) {
	origDiscover := discoverFn
	defer func() { discoverFn = origDiscover }()

	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return nil, nil
	}

	_, err := ExplainServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestExplainServerWithOccupiedPort(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) {
		return []model.MCPServer{
			{Name: "http-server", Client: "c", Transport: "http", URL: "http://localhost:3000"},
		}, nil
	}
	listListeningFn = func() ([]model.Port, error) {
		return []model.Port{
			{Port: 3000, Protocol: "tcp", PID: 42, Process: "nginx"},
		}, nil
	}

	exp, err := ExplainServer("http-server")
	if err != nil {
		t.Fatalf("ExplainServer: %v", err)
	}
	if exp.Occupied == nil {
		t.Fatal("expected Occupied to be set")
	}
	if exp.Occupied.Process != "nginx" {
		t.Errorf("occupied process: want %q, got %q", "nginx", exp.Occupied.Process)
	}
}

func TestExplainPortWithConflict(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) {
		return []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 1, Process: "a"},
			{Port: 8080, Protocol: "tcp", PID: 2, Process: "b"},
		}, nil
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) { return nil, nil }

	exp, err := ExplainPort(8080)
	if err != nil {
		t.Fatalf("ExplainPort: %v", err)
	}
	if len(exp.Conflicts) == 0 {
		t.Error("expected conflicts for port with two PIDs")
	}
}

func TestExplainPortMultipleListeners(t *testing.T) {
	origListening := listListeningFn
	origDiscover := discoverFn
	defer func() {
		listListeningFn = origListening
		discoverFn = origDiscover
	}()

	listListeningFn = func() ([]model.Port, error) {
		return []model.Port{
			{Port: 8080, Protocol: "tcp", PID: 1, Process: "a"},
			{Port: 8080, Protocol: "udp", PID: 2, Process: "b"},
		}, nil
	}
	discoverFn = func(sources []mcpcfg.Source) ([]model.MCPServer, []string) { return nil, nil }

	exp, err := ExplainPort(8080)
	if err != nil {
		t.Fatalf("ExplainPort: %v", err)
	}
	if len(exp.Processes) != 2 {
		t.Errorf("expected 2 processes, got %d", len(exp.Processes))
	}
}
