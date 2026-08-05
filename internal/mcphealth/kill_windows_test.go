//go:build windows

package mcphealth

import "testing"

// assertProcessDead is a no-op on windows: there is no signal-0 liveness
// probe, and the probed child is reaped by killProcess's c.Wait().
func assertProcessDead(t *testing.T, pid int) {
	t.Helper()
	t.Logf("skipping process-death assertion on windows for pid %d", pid)
}
