package ports

import (
	"testing"

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
