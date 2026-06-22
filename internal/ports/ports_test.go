package ports

import (
	"fmt"
	"strings"
	"syscall"
	"testing"

	psnet "github.com/shirou/gopsutil/v4/net"

	"github.com/danieljustus/symaira-scope/internal/model"
)

func TestSuggestFreeReturnsRequestedCount(t *testing.T) {
	free := SuggestFree(3, 49200, 49999)
	if len(free) != 3 {
		t.Fatalf("want 3 free ports, got %d", len(free))
	}
	for _, p := range free {
		if p < 49200 || p > 49999 {
			t.Fatalf("port %d out of requested range", p)
		}
	}
}

func TestConflictsIgnoresSamePID(t *testing.T) {
	in := []model.Port{
		{Port: 8080, Protocol: "tcp", PID: 1, Process: "a"},
		{Port: 8080, Protocol: "tcp", PID: 1, Process: "a"}, // IPv4 + IPv6 of one process
	}
	if got := Conflicts(in); len(got) != 0 {
		t.Fatalf("same PID on a port must not conflict, got %v", got)
	}
}

func TestConflictsDetectsTwoPIDs(t *testing.T) {
	in := []model.Port{
		{Port: 8080, Protocol: "tcp", PID: 1, Process: "a"},
		{Port: 8080, Protocol: "tcp", PID: 2, Process: "b"},
	}
	got := Conflicts(in)
	if len(got) != 1 || got[0].Port != 8080 || len(got[0].Holders) != 2 {
		t.Fatalf("expected one conflict on 8080 with two holders, got %v", got)
	}
}

func TestConflictsIgnoresPIDZero(t *testing.T) {
	in := []model.Port{
		{Port: 53, Protocol: "tcp", PID: 0, Process: ""},
		{Port: 53, Protocol: "tcp", PID: 7, Process: "named"},
	}
	if got := Conflicts(in); len(got) != 0 {
		t.Fatalf("PID 0 must be ignored, got %v", got)
	}
}

type fakeConnLister struct {
	conns []psnet.ConnectionStat
	err   error
}

func (f *fakeConnLister) Connections(kind string) ([]psnet.ConnectionStat, error) {
	return f.conns, f.err
}

func TestListListeningFiltersNonListenTCP(t *testing.T) {
	lister := &fakeConnLister{
		conns: []psnet.ConnectionStat{
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 3000}, Type: uint32(syscall.SOCK_STREAM), Status: "ESTABLISHED", Pid: 100},
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 8080}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 200},
		},
	}
	ports, err := listListeningWith(lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 || ports[0].Port != 8080 {
		t.Errorf("expected only LISTEN port 8080, got %v", ports)
	}
}

func TestListListeningIncludesUDP(t *testing.T) {
	lister := &fakeConnLister{
		conns: []psnet.ConnectionStat{
			{Laddr: psnet.Addr{IP: "0.0.0.0", Port: 5353}, Type: uint32(syscall.SOCK_DGRAM), Pid: 100},
		},
	}
	ports, err := listListeningWith(lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 || ports[0].Port != 5353 || ports[0].Protocol != "udp" {
		t.Errorf("expected UDP port 5353, got %v", ports)
	}
}

func TestListListeningDeduplicatesIdenticalEntries(t *testing.T) {
	lister := &fakeConnLister{
		conns: []psnet.ConnectionStat{
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 9090}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 1},
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 9090}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 1},
		},
	}
	ports, err := listListeningWith(lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 {
		t.Errorf("expected deduplication to 1 port, got %d", len(ports))
	}
}

func TestListListeningResolvesProcessName(t *testing.T) {
	// PID 0 → no process name resolved
	lister := &fakeConnLister{
		conns: []psnet.ConnectionStat{
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 80}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 0},
		},
	}
	ports, err := listListeningWith(lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 1 || ports[0].PID != 0 || ports[0].Process != "" {
		t.Errorf("PID 0 should have empty process, got %+v", ports)
	}
}

func TestListListeningSortedByPort(t *testing.T) {
	lister := &fakeConnLister{
		conns: []psnet.ConnectionStat{
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 9000}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 1},
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 1000}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 1},
			{Laddr: psnet.Addr{IP: "127.0.0.1", Port: 5000}, Type: uint32(syscall.SOCK_STREAM), Status: "LISTEN", Pid: 1},
		},
	}
	ports, err := listListeningWith(lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
	if ports[0].Port != 1000 || ports[1].Port != 5000 || ports[2].Port != 9000 {
		t.Errorf("ports not sorted: %v", ports)
	}
}

func TestListListeningPropagatesError(t *testing.T) {
	lister := &fakeConnLister{
		err: fmt.Errorf("permission denied"),
	}
	_, err := listListeningWith(lister)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestMCPServerConflictsNoConflict(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "s1", Client: "c", URL: "http://localhost:3000"},
	}
	listening := []model.Port{
		{Port: 8080, Protocol: "tcp", Process: "nginx"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %v", got)
	}
}

func TestMCPServerConflictsSingleConflict(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "s1", Client: "c1", URL: "http://localhost:8080"},
		{Name: "s2", Client: "c2", URL: "http://localhost:8080"},
	}
	listening := []model.Port{
		{Port: 8080, Protocol: "tcp", Process: "nginx"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 1 || got[0].Port != 8080 {
		t.Fatalf("expected conflict on 8080, got %v", got)
	}
	if got[0].Kind != "mcp-occupied" {
		t.Errorf("kind: want %q, got %q", "mcp-occupied", got[0].Kind)
	}
}

func TestMCPServerConflictsMultipleServersSamePort(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "s1", Client: "c1", URL: "http://localhost:3000"},
		{Name: "s2", Client: "c2", URL: "http://localhost:3000"},
	}
	listening := []model.Port{
		{Port: 3000, Protocol: "tcp", Process: "node"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 1 || got[0].Port != 3000 {
		t.Fatalf("expected conflict on 3000, got %v", got)
	}
	if len(got[0].Holders) != 2 {
		t.Errorf("expected 2 holders, got %d: %v", len(got[0].Holders), got[0].Holders)
	}
}

func TestMCPServerConflictsSkipsEmptyURL(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "stdio-server", Client: "c", Command: "node"},
	}
	listening := []model.Port{
		{Port: 8080, Protocol: "tcp", Process: "node"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 0 {
		t.Fatalf("empty URL should not conflict, got %v", got)
	}
}

func TestMCPServerConflictsSkipsInvalidURL(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "bad", Client: "c", URL: "not-a-url"},
	}
	listening := []model.Port{
		{Port: 80, Protocol: "tcp", Process: "http"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 0 {
		t.Fatalf("invalid URL should skip gracefully, got %v", got)
	}
}

func TestMCPServerConflictsSkipsPortZero(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "s", Client: "c", URL: "http://localhost"},
	}
	listening := []model.Port{
		{Port: 80, Protocol: "tcp", Process: "http"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 0 {
		t.Fatalf("port 0 (or missing) should skip, got %v", got)
	}
}

func TestMCPServerConflictsEmptyInputs(t *testing.T) {
	got := MCPServerConflicts(nil, nil)
	if len(got) != 0 {
		t.Fatalf("empty inputs should return no conflicts, got %v", got)
	}
}

func TestMCPServerConflictsAnnotatesOccupied(t *testing.T) {
	servers := []model.MCPServer{
		{Name: "s1", Client: "cursor", URL: "http://localhost:9000"},
		{Name: "s2", Client: "vscode", URL: "http://localhost:9000"},
	}
	listening := []model.Port{
		{Port: 9000, Protocol: "tcp", Process: "myapp"},
	}
	got := MCPServerConflicts(servers, listening)
	if len(got) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(got))
	}
	for _, h := range got[0].Holders {
		if !strings.Contains(h, "occupied by myapp") {
			t.Errorf("holder %q should annotate occupied process", h)
		}
	}
}
