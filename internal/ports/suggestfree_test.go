package ports

import (
	"fmt"
	"net"
	"testing"
)

// Port ranges used here (46000-47500) sit below the macOS ephemeral pool
// (49152+) and are disjoint from the existing SuggestFree test (49200-49999)
// and from each other, so tests cannot interfere when run repeatedly.

// bindPorts binds TCP listeners on every port in [from, to] on 127.0.0.1,
// simulating ports already taken by other processes. Listeners are released
// via t.Cleanup when the test finishes.
func bindPorts(t *testing.T, from, to int) {
	t.Helper()
	for p := from; p <= to; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			t.Fatalf("bind port %d: %v", p, err)
		}
		t.Cleanup(func() { _ = l.Close() })
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
	const from, to = 46000, 46005 // 6 ports => sequential path
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
	const from, to = 46100, 46105 // 6 ports => sequential path
	bindPorts(t, from, to)
	if got := SuggestFree(3, from, to); len(got) != 0 {
		t.Fatalf("want no free ports, got %v", got)
	}
}

func TestSuggestFreeSequentialSkipsTakenPorts(t *testing.T) {
	const from, to = 46200, 46204 // 5 ports => sequential path
	bindPorts(t, 46201, 46201)
	bindPorts(t, 46203, 46203)
	got := SuggestFree(5, from, to)
	want := []int{46200, 46202, 46204}
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
	const from, to = 47000, 47100 // 101 ports => worker path
	bindPorts(t, from, to)
	if got := SuggestFree(5, from, to); len(got) != 0 {
		t.Fatalf("want no free ports, got %v", got)
	}
}

func TestSuggestFreeWorkerSkipsTakenPorts(t *testing.T) {
	const from, to = 47200, 47300 // 101 ports => worker path
	bindPorts(t, 47210, 47250)    // 41 taken, ~60 free
	got := SuggestFree(3, from, to)
	if len(got) != 3 {
		t.Fatalf("want 3 free ports, got %v", got)
	}
	taken := boundSet(47210, 47250)
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
	const from, to = 47400, 47500 // 101 ports => worker path
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
	got := SuggestFree(0, 46300, 46300)
	if len(got) != 1 || got[0] != 46300 {
		t.Fatalf("count 0 should behave like count 1, got %v", got)
	}
}

func TestSuggestFreeNegativeCountDefaultsToOne(t *testing.T) {
	got := SuggestFree(-3, 46301, 46301)
	if len(got) != 1 || got[0] != 46301 {
		t.Fatalf("negative count should behave like count 1, got %v", got)
	}
}

func TestSuggestFreeInvalidRangeReturnsNil(t *testing.T) {
	if got := SuggestFree(3, 48000, 47999); got != nil {
		t.Fatalf("from > to should return nil, got %v", got)
	}
}
