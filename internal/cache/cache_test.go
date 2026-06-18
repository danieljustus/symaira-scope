package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// useTempCache redirects the cache to a temporary directory for test isolation.
func useTempCache(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	cacheDirOverride = tmp
	t.Cleanup(func() { cacheDirOverride = "" })
}

func testSnapshot() *model.Snapshot {
	return &model.Snapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Ports: []model.Port{
			{Port: 8080, Protocol: "tcp", Address: "0.0.0.0:8080", PID: 1234, Process: "test"},
		},
	}
}

func TestSaveAndLoad(t *testing.T) {
	useTempCache(t)

	snap := testSnapshot()
	if err := Save(snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil snapshot")
	}
	if got.GeneratedAt != snap.GeneratedAt {
		t.Errorf("GeneratedAt = %q, want %q", got.GeneratedAt, snap.GeneratedAt)
	}
	if len(got.Ports) != 1 || got.Ports[0].Port != 8080 {
		t.Errorf("unexpected ports: %+v", got.Ports)
	}
}

func TestLoadCacheMiss(t *testing.T) {
	useTempCache(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load on empty cache: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for cache miss, got %+v", got)
	}
}

func TestLoadStaleCache(t *testing.T) {
	useTempCache(t)

	// Manually write an envelope with CachedAt in the distant past.
	cp, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}
	env := cacheEnvelope{
		CachedAt: time.Now().UTC().Add(-1 * time.Hour), // well past TTL
		Snapshot: *testSnapshot(),
	}
	data, _ := json.Marshal(env)
	if err := os.WriteFile(cp, data, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for stale cache, got %+v", got)
	}
}

func TestClear(t *testing.T) {
	useTempCache(t)

	snap := testSnapshot()
	if err := Save(snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load after Clear: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after Clear, got %+v", got)
	}
}

func TestClearNoFile(t *testing.T) {
	useTempCache(t)

	// Clear on a non-existent cache should not error.
	if err := Clear(); err != nil {
		t.Fatalf("Clear on missing cache: %v", err)
	}
}

func TestStatsEmpty(t *testing.T) {
	useTempCache(t)

	st := Stats()
	if st.Exists {
		t.Error("expected Exists=false for empty cache")
	}
	if st.TTL != TTL.String() {
		t.Errorf("TTL = %q, want %q", st.TTL, TTL.String())
	}
}

func TestStatsAfterSave(t *testing.T) {
	useTempCache(t)

	snap := testSnapshot()
	if err := Save(snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	st := Stats()
	if !st.Exists {
		t.Error("expected Exists=true after Save")
	}
	if st.Size <= 0 {
		t.Errorf("expected positive size, got %d", st.Size)
	}
	if !st.Valid {
		t.Error("expected Valid=true for fresh cache")
	}
	if st.Age == "" {
		t.Error("expected non-empty Age")
	}
}

func TestStatsCorruptedFile(t *testing.T) {
	useTempCache(t)

	cp, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(cp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := Stats()
	if !st.Exists {
		t.Error("expected Exists=true for corrupted file")
	}
	// Corrupted = not valid.
	if st.Valid {
		t.Error("expected Valid=false for corrupted cache")
	}
}

func TestOverwriteCache(t *testing.T) {
	useTempCache(t)

	snap1 := testSnapshot()
	snap1.GeneratedAt = "2025-01-01T00:00:00Z"
	if err := Save(snap1); err != nil {
		t.Fatalf("Save 1: %v", err)
	}

	snap2 := testSnapshot()
	snap2.GeneratedAt = "2026-06-18T00:00:00Z"
	if err := Save(snap2); err != nil {
		t.Fatalf("Save 2: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil")
	}
	if got.GeneratedAt != "2026-06-18T00:00:00Z" {
		t.Errorf("GeneratedAt = %q, want second save's value", got.GeneratedAt)
	}
}

func TestCachePath(t *testing.T) {
	useTempCache(t)

	cp, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	if filepath.Ext(cp) != ".json" {
		t.Errorf("expected .json extension, got %q", cp)
	}
}
