// Package cache provides a snapshot cache with atomic writes, TTL expiry,
// and advisory file locking for concurrent-process safety.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danieljustus/symaira-corekit/fsutil"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// TTL is the cache time-to-live. After this duration the cached snapshot
// is considered stale and will not be returned by Load.
const TTL = 5 * time.Minute

// cacheEnvelope wraps a snapshot with a timestamp so TTL can be checked.
type cacheEnvelope struct {
	CachedAt time.Time      `json:"cached_at"`
	Snapshot model.Snapshot `json:"snapshot"`
}

// CacheStats describes the current state of the cache file on disk.
type CacheStats struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size_bytes,omitempty"`
	Age    string `json:"age,omitempty"`
	Valid  bool   `json:"valid"`
	TTL    string `json:"ttl"`
}

// cacheDirOverride is set by tests to redirect the cache directory.
var cacheDirOverride string

// dir returns the cache directory (~/.cache/symscope on Linux,
// ~/Library/Caches/symscope on macOS) and ensures it exists.
func dir() (string, error) {
	if cacheDirOverride != "" {
		d := filepath.Join(cacheDirOverride, "symscope")
		if err := os.MkdirAll(d, 0o700); err != nil {
			return "", fmt.Errorf("cache dir mkdir: %w", err)
		}
		return d, nil
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache dir: %w", err)
	}
	d := filepath.Join(base, "symscope")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", fmt.Errorf("cache dir mkdir: %w", err)
	}
	return d, nil
}

// cachePath returns the full path to the snapshot cache file.
func cachePath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "snapshot.json"), nil
}

// lockPath returns the path of the advisory lock file alongside the cache.
func lockPath() (string, error) {
	cp, err := cachePath()
	if err != nil {
		return "", err
	}
	return cp + ".lock", nil
}

// withLock executes fn while holding an advisory file lock on the lock file.
// The lock is released when fn returns.
func withLock(exclusive bool, fn func() error) error {
	lp, err := lockPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(lp, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("lock open: %w", err)
	}
	defer f.Close()

	if err := flock(f, exclusive); err != nil {
		return fmt.Errorf("lock acquire: %w", err)
	}
	defer funlock(f)

	return fn()
}

// Save writes a snapshot to the cache atomically. An advisory exclusive lock
// is held for the duration of the write.
func Save(snap *model.Snapshot) error {
	return withLock(true, func() error {
		cp, err := cachePath()
		if err != nil {
			return err
		}

		env := cacheEnvelope{
			CachedAt: time.Now().UTC(),
			Snapshot: *snap,
		}

		data, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("cache marshal: %w", err)
		}

		if err := fsutil.AtomicWriteFile(cp, data, 0o600); err != nil {
			return fmt.Errorf("cache write: %w", err)
		}
		return nil
	})
}

// Load reads a cached snapshot if it exists and is within TTL. Returns nil
// (with no error) when the cache is missing or stale — callers should treat
// this as a cache miss and run scan.Build() directly.
func Load() (*model.Snapshot, error) {
	var snap *model.Snapshot

	err := withLock(false, func() error {
		cp, err := cachePath()
		if err != nil {
			return err
		}

		data, err := os.ReadFile(cp)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // cache miss — not an error
			}
			return fmt.Errorf("cache read: %w", err)
		}

		var env cacheEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			// Corrupted cache — treat as miss, don't error.
			return nil
		}

		if time.Since(env.CachedAt) > TTL {
			// Stale — treat as miss.
			return nil
		}

		snap = &env.Snapshot
		return nil
	})

	if err != nil {
		return nil, err
	}
	return snap, nil
}

// Clear deletes the cache file and its lock file if they exist.
func Clear() error {
	return withLock(true, func() error {
		cp, err := cachePath()
		if err != nil {
			return err
		}
		// Best-effort removal; ignore "not found".
		_ = os.Remove(cp)
		_ = os.Remove(cp + ".lock")
		return nil
	})
}

// Stats returns metadata about the cache file on disk.
func Stats() CacheStats {
	cp, err := cachePath()
	if err != nil {
		return CacheStats{TTL: TTL.String()}
	}

	st := CacheStats{
		Path: cp,
		TTL:  TTL.String(),
	}

	fi, err := os.Stat(cp)
	if err != nil {
		return st
	}

	st.Exists = true
	st.Size = fi.Size()
	st.Age = time.Since(fi.ModTime()).Round(time.Second).String()

	// Check validity by reading the envelope timestamp.
	data, err := os.ReadFile(cp)
	if err != nil {
		return st
	}
	var env cacheEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return st
	}
	st.Valid = time.Since(env.CachedAt) <= TTL

	return st
}
