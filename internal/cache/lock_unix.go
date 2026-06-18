//go:build !windows

package cache

import (
	"os"
	"syscall"
)

// flock acquires an advisory lock on f. When exclusive is true it takes an
// exclusive (write) lock; otherwise a shared (read) lock.
func flock(f *os.File, exclusive bool) error {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	return syscall.Flock(int(f.Fd()), how)
}

// funlock releases the advisory lock on f.
func funlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
