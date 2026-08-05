//go:build !windows

package mcphealth

import (
	"syscall"
	"testing"
)

// assertProcessDead verifies that the process with the given pid no longer
// exists. Signal 0 performs a liveness check without delivering a signal: it
// returns an error (ESRCH) once the process has been reaped. Used to prove
// that killProcess actually terminates the probed child.
func assertProcessDead(t *testing.T, pid int) {
	t.Helper()
	if err := syscall.Kill(pid, 0); err == nil {
		t.Errorf("expected process %d to be dead, but signal 0 succeeded", pid)
	} else {
		t.Logf("process %d confirmed dead: %v", pid, err)
	}
}
