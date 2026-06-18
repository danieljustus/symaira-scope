//go:build windows

package cache

import "os"

// flock is a no-op on Windows. Advisory file locks are not supported;
// concurrent safety relies on atomic writes (temp+rename) alone.
func flock(_ *os.File, _ bool) error { return nil }

// funlock is a no-op on Windows.
func funlock(_ *os.File) error { return nil }
