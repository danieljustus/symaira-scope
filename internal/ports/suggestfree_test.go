package ports

import (
	"fmt"
	"net"
	"testing"
)

// basePort reserves a free TCP port on 127.0.0.1 and returns it. The
// listener stays open until test cleanup, so the reserved port is stable
// for the whole test. Test ranges are derived as base-offset, so they stay
// within the valid port space and avoid OS-excluded ranges (e.g. Windows
// Hyper-V/WinNAT reservations, which make binding fail with WSAEACCES).
func basePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot reserve a TCP port on this platform: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l.Addr().(*net.TCPAddr).Port
}

// bindPorts binds TCP listeners on every port in [from, to] on 127.0.0.1,
// simulating ports already taken by other processes. Listeners are released
// via t.Cleanup when the test finishes. A port that cannot be bound (already
// taken by another process or excluded by the OS) is already not free, which
// is exactly the state this fixture simulates, so it is skipped rather than
// failing the test.
func bindPorts(t *testing.T, from, to int) {
	t.Helper()
	for p := from; p <= to; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			t.Logf("port %d already taken/excluded: %v", p, err)
			continue
		}
		t.Cleanup(func() { _ = l.Close() })
	}
}

// requireFree asserts that every port in [from, to] is currently bindable.
// When the OS excludes any of them the fixture cannot be built, so the test
// skips instead of failing on a platform limitation.
func requireFree(t *testing.T, from, to int) {
	t.Helper()
	for p := from; p <= to; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			t.Skipf("port %d not bindable on this platform (%v); skipping", p, err)
		}
		_ = l.Close()
	}
}

func boundSet(from, to int) map[int]bool {
	s := make(map[int]bool, to-from+1)
	for p := from; p <= to; p++ {
		s[p] = true
	}
	return s
}

// --- Sequential fallback (ranges of <= 64 ports) ---

func TestSuggestFreeSequentialReturnsRequestedCount(t *testing.T) {
	base := basePort(t)
	from, to := base-6, base-1 // 6 ports => sequential path
	requireFree(t, from, to)
	got := SuggestFree(3, from, to)
	if len(got) != 3 {
		t.Fatalf("want 3 free ports, got %v", got)
	}
	for _, p := range got {
		if p < from || p > to {
			t.Fatalf("port %d out of range [%d, %d]", p, from, to)
		}
	}
}

func TestSuggestFreeSequentialAllTakenReturnsEmpty(t *testing.T) {
	base := basePort(t)
	from, to := base-106, base-101 // 6 ports => sequential path
	bindPorts(t, from, to)
	if got := SuggestFree(3, from, to); len(got) != 0 {
		t.Fatalf("want no free ports, got %v", got)
	}
}

func TestSuggestFreeSequentialSkipsTakenPorts(t *testing.T) {
	base := basePort(t)
	from, to := base-205, base-201 // 5 ports => sequential path
	requireFree(t, from, to)
	bindPorts(t, base-204, base-204)
	bindPorts(t, base-202, base-202)
	got := SuggestFree(5, from, to)
	want := []int{base - 205, base - 203, base - 201}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

// --- Worker path (ranges of > 64 ports) ---

func TestSuggestFreeWorkerAllTakenReturnsEmpty(t *testing.T) {
	base := basePort(t)
	from, to := base-1100, base-1000 // 101 ports => worker path
	bindPorts(t, from, to)
	if got := SuggestFree(5, from, to); len(got) != 0 {
		t.Fatalf("want no free ports, got %v", got)
	}
}

func TestSuggestFreeWorkerSkipsTakenPorts(t *testing.T) {
	base := basePort(t)
	from, to := base-1300, base-1200 // 101 ports => worker path
	requireFree(t, from, to)
	bindPorts(t, base-1290, base-1250) // 41 taken, ~60 free
	got := SuggestFree(3, from, to)
	if len(got) != 3 {
		t.Fatalf("want 3 free ports, got %v", got)
	}
	taken := boundSet(base-1290, base-1250)
	for _, p := range got {
		if p < from || p > to {
			t.Fatalf("port %d out of range [%d, %d]", p, from, to)
		}
		if taken[p] {
			t.Fatalf("returned taken port %d", p)
		}
	}
}

// TestSuggestFreeWorkerEarlyExitSinglePort exercises the atomic early-exit
// (found.Load() >= count): with count=1 the first worker that binds a port
// satisfies the request, and every other worker must bail out on the next
// loop iteration instead of scanning the whole range. The len(free) < count
// guard under the mutex is also hit because several workers may bind a port
// before noticing the request is already satisfied.
func TestSuggestFreeWorkerEarlyExitSinglePort(t *testing.T) {
	base := basePort(t)
	from, to := base-1500, base-1400 // 101 ports => worker path
	requireFree(t, from, to)
	got := SuggestFree(1, from, to)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 port, got %v", got)
	}
	if p := got[0]; p < from || p > to {
		t.Fatalf("port %d out of range [%d, %d]", p, from, to)
	}
}

// --- Edge cases ---

func TestSuggestFreeCountZeroDefaultsToOne(t *testing.T) {
	base := basePort(t)
	p := base - 2000
	requireFree(t, p, p)
	got := SuggestFree(0, p, p)
	if len(got) != 1 || got[0] != p {
		t.Fatalf("count 0 should behave like count 1, got %v", got)
	}
}

func TestSuggestFreeNegativeCountDefaultsToOne(t *testing.T) {
	base := basePort(t)
	p := base - 2001
	requireFree(t, p, p)
	got := SuggestFree(-3, p, p)
	if len(got) != 1 || got[0] != p {
		t.Fatalf("negative count should behave like count 1, got %v", got)
	}
}

func TestSuggestFreeInvalidRangeReturnsNil(t *testing.T) {
	base := basePort(t)
	if got := SuggestFree(3, base-3000, base-3001); got != nil {
		t.Fatalf("from > to should return nil, got %v", got)
	}
}
