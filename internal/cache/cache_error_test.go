//go:build !windows

package cache

// Error-path tests for dir, withLock and Save. They inject failures with
// OS-level seams (read-only directories, a FIFO at the lock path, unset
// HOME), which behave differently on Windows, hence the build tag. None of
// these tests may run with t.Parallel(): cacheDirOverride is a package-level
// global that each test redirects and restores.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// useReadOnlyParent redirects the cache under a parent directory with mode
// 0500, so MkdirAll inside dir() fails with EACCES. Skips on root, where
// permission bits are not enforced.
func useReadOnlyParent(t *testing.T) {
	t.Helper()
	parent := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("permission-based failure injection requires a non-root user")
	}
	cacheDirOverride = parent
	t.Cleanup(func() { cacheDirOverride = "" })
}

func TestDirMkdirError(t *testing.T) {
	useReadOnlyParent(t)

	d, err := dir()
	if err == nil {
		t.Fatalf("dir() = %q, want error", d)
	}
	if !strings.Contains(err.Error(), "cache dir mkdir") {
		t.Errorf("error = %q, want cache dir mkdir", err)
	}
}

func TestDirUserCacheError(t *testing.T) {
	// Force os.UserCacheDir to fail by unsetting HOME (and XDG_CACHE_HOME on
	// Linux). cacheDirOverride must stay empty for the default branch.
	cacheDirOverride = ""
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Cleanup(func() { cacheDirOverride = "" })

	d, err := dir()
	if err == nil {
		t.Fatalf("dir() = %q, want error", d)
	}
	if !strings.Contains(err.Error(), "cache dir:") {
		t.Errorf("error = %q, want cache dir:", err)
	}
}

func TestDirUserCacheMkdirError(t *testing.T) {
	// Point HOME at a temp dir whose cache parent exists as a regular file,
	// so MkdirAll in the default branch fails with ENOTDIR.
	home := t.TempDir()
	cacheParent := filepath.Join(home, ".cache")
	if runtime.GOOS == "darwin" {
		cacheParent = filepath.Join(home, "Library", "Caches")
	}
	if err := os.MkdirAll(filepath.Dir(cacheParent), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheParent, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}

	cacheDirOverride = ""
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")
	t.Cleanup(func() { cacheDirOverride = "" })

	d, err := dir()
	if err == nil {
		t.Fatalf("dir() = %q, want error", d)
	}
	if !strings.Contains(err.Error(), "cache dir mkdir") {
		t.Errorf("error = %q, want cache dir mkdir", err)
	}
}

func TestWithLockDirError(t *testing.T) {
	useReadOnlyParent(t)

	ran := false
	err := withLock(true, func() error {
		ran = true
		return nil
	})
	if err == nil {
		t.Fatal("withLock: expected error from dir()")
	}
	if ran {
		t.Error("fn must not run when the lock path cannot be resolved")
	}
	if !strings.Contains(err.Error(), "cache dir mkdir") {
		t.Errorf("error = %q, want cache dir mkdir", err)
	}
}

func TestWithLockLockOpenError(t *testing.T) {
	useTempCache(t)

	cp, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}
	lp := cp + ".lock"
	// A directory at the lock path makes os.OpenFile fail with EISDIR.
	if err := os.Mkdir(lp, 0o700); err != nil {
		t.Fatal(err)
	}

	ran := false
	err = withLock(true, func() error {
		ran = true
		return nil
	})
	if err == nil {
		t.Fatal("withLock: expected lock open error")
	}
	if ran {
		t.Error("fn must not run when the lock file cannot be opened")
	}
	if !strings.Contains(err.Error(), "lock open") {
		t.Errorf("error = %q, want lock open", err)
	}
}

func TestWithLockFlockError(t *testing.T) {
	useTempCache(t)

	cp, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}

	// Probe first: on this platform, does flock fail on a FIFO? It does on
	// macOS (EOPNOTSUPP); if it does not elsewhere, skip rather than assume.
	probe := filepath.Join(t.TempDir(), "probe")
	if err := syscall.Mkfifo(probe, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	pf, err := os.OpenFile(probe, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if perr := syscall.Flock(int(pf.Fd()), syscall.LOCK_EX); perr == nil {
		pf.Close()
		t.Skip("flock does not fail on FIFOs on this platform")
	}
	pf.Close()

	// A FIFO at the lock path opens fine (O_RDWR never blocks on FIFOs) but
	// flock rejects it, hitting the lock-acquire error branch.
	lp := cp + ".lock"
	if err := syscall.Mkfifo(lp, 0o600); err != nil {
		t.Fatal(err)
	}

	ran := false
	err = withLock(true, func() error {
		ran = true
		return nil
	})
	if err == nil {
		t.Fatal("withLock: expected lock acquire error")
	}
	if ran {
		t.Error("fn must not run when the lock cannot be acquired")
	}
	if !strings.Contains(err.Error(), "lock acquire") {
		t.Errorf("error = %q, want lock acquire", err)
	}
}

func TestSaveDirError(t *testing.T) {
	useReadOnlyParent(t)

	err := Save(testSnapshot())
	if err == nil {
		t.Fatal("Save: expected error from dir()")
	}
	if !strings.Contains(err.Error(), "cache dir mkdir") {
		t.Errorf("error = %q, want cache dir mkdir", err)
	}
}

func TestSaveWriteError(t *testing.T) {
	useTempCache(t)

	snap := testSnapshot()
	if err := Save(snap); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// Make the cache directory read-only after the lock file exists, so
	// withLock still succeeds but AtomicWriteFile cannot create its temp file.
	cp, err := cachePath()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(cp)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("permission-based failure injection requires a non-root user")
	}
	defer func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Logf("restore cache dir perms: %v", err)
		}
	}()

	if err := Save(snap); err == nil {
		t.Fatal("Save: expected cache write error")
	} else if !strings.Contains(err.Error(), "cache write") {
		t.Errorf("error = %q, want cache write", err)
	}

	// Cleanup behavior: the failed atomic write must not leave a partial
	// snapshot behind — the previous one still loads, and no temp files linger.
	got, err := Load()
	if err != nil {
		t.Fatalf("Load after failed Save: %v", err)
	}
	if got == nil {
		t.Fatal("expected previous snapshot to survive a failed Save")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("cache dir after failed Save = %v, want only snapshot.json and its lock", names)
	}
}
