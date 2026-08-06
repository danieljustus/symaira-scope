package mcphealth

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Environment variables that turn the test binary into an MCP stdio helper
// process. See TestHelperProcess and helperInvocation.
const (
	helperEnv        = "SYMSCOPE_MCPHEALTH_HELPER"
	helperEnvMode    = "SYMSCOPE_MCPHEALTH_HELPER_MODE"
	helperEnvPidfile = "SYMSCOPE_MCPHEALTH_HELPER_PIDFILE"
	helperEnvEcho    = "SYMSCOPE_MCPHEALTH_HELPER_ECHO"
)

// TestHelperProcess is not a real test: when the test binary is re-executed
// with SYMSCOPE_MCPHEALTH_HELPER=1 it acts as an MCP stdio server (or a
// deliberately broken one) instead. Modes:
//
//   - hang:     reads stdin but never answers (a wedged server)
//   - exit:     exits immediately (a server that dies before the handshake)
//   - respond:  answers with a valid JSON-RPC initialize result
//   - noresult: answers with a JSON-RPC error (no "result" field)
//   - inspect:  echoes the received request method/params back inside the
//     result (for Inspect* tests)
func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	switch os.Getenv(helperEnvMode) {
	case "hang":
		// Record our pid before hanging so the parent test can prove we were
		// actually killed by killProcess after the probe timed out.
		if pidfile := os.Getenv(helperEnvPidfile); pidfile != "" {
			_ = os.WriteFile(pidfile, []byte(strconv.Itoa(os.Getpid())), 0o600)
		}
		_, _ = io.Copy(io.Discard, os.Stdin) // consume the initialize request
		// Never answer. A bare select{} would trip the Go runtime's deadlock
		// detector ("all goroutines are asleep") and exit the process, so
		// sleep instead; the probe's killProcess terminates us at 5s.
		time.Sleep(24 * time.Hour)
	case "exit":
		os.Exit(0)
	case "respond":
		_, _ = io.Copy(io.Discard, os.Stdin)
		fmt.Fprint(os.Stdout, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"helper","version":"1.0"}}}`)
		os.Exit(0)
	case "noresult":
		_, _ = io.Copy(io.Discard, os.Stdin)
		fmt.Fprint(os.Stdout, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"bad request"}}`)
		os.Exit(0)
	case "inspect":
		reqBytes, _ := io.ReadAll(os.Stdin)
		var req map[string]any
		_ = json.Unmarshal(reqBytes, &req)
		result := map[string]any{
			"method":   req["method"],
			"echo_env": os.Getenv(helperEnvEcho),
		}
		if params, ok := req["params"]; ok {
			result["params"] = params
		}
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
		fmt.Fprint(os.Stdout, string(resp))
		os.Exit(0)
	}
	os.Exit(0)
}

// helperInvocation returns the command line that re-executes the current test
// binary as a stdio helper process in the given mode. The mode is passed
// through the test process environment, which ProbeStdio's child inherits.
func helperInvocation(t *testing.T, mode, pidfile string) (string, []string) {
	t.Helper()
	t.Setenv(helperEnv, "1")
	t.Setenv(helperEnvMode, mode)
	if pidfile != "" {
		t.Setenv(helperEnvPidfile, pidfile)
	}
	return os.Args[0], []string{"-test.run=^TestHelperProcess$"}
}

// readHelperPid polls pidfile (written by the hang helper before it hangs)
// until it contains a pid, failing the test if it never appears.
func readHelperPid(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil {
			if pid, perr := strconv.Atoi(strings.TrimSpace(string(data))); perr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper pid file %s never appeared", path)
	return 0
}

func TestProbeStdio_Timeout(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "helper.pid")
	cmd, args := helperInvocation(t, "hang", pidfile)

	start := time.Now()
	result := ProbeStdio(cmd, args)
	elapsed := time.Since(start)

	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy for hanging server, got %q (error: %s)", result.Status, result.Error)
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("expected error to mention timeout, got %q", result.Error)
	}
	if result.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", result.LatencyMs)
	}
	// The probe must spend its ~5s probeTimeout budget waiting for the
	// hanging server instead of returning early. Generous window to avoid
	// flakes on slow machines.
	if elapsed < 4*time.Second {
		t.Errorf("expected timeout probe to take roughly %s, took %s", probeTimeout, elapsed)
	}

	// The helper recorded its pid before hanging; by the time ProbeStdio
	// returned, killProcess must have terminated and reaped it.
	pid := readHelperPid(t, pidfile, 3*time.Second)
	assertProcessDead(t, pid)
}

func TestProbeStdio_WriteFailure(t *testing.T) {
	// Deterministic part: a server that exits immediately can never complete
	// the handshake, so the probe must report unhealthy either way. The error
	// is "write: ..." when the child wins the race against the parent's stdin
	// write (EPIPE), or "parse: ..." when the write lands first and the child
	// dies before responding.
	cmd, args := helperInvocation(t, "exit", "")
	result := ProbeStdio(cmd, args)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy for exiting server, got %q (error: %s)", result.Status, result.Error)
	}
	if result.Error == "" {
		t.Error("expected a non-empty error for exiting server")
	}

	// The stdin-write failure branch (health.go stdin.Write error) is only
	// reachable when the child process exits before the parent's write syscall
	// lands. The test binary starts too slowly for that race to fire, so retry
	// with a fast native binary (/usr/bin/true) that exits in microseconds.
	// Bounded retries keep the test fast and flake-free.
	if runtime.GOOS == "windows" {
		t.Log("write-failure branch not exercised on windows: no fast-exit race partner available")
		return
	}
	trueBin, err := exec.LookPath("true")
	if err != nil {
		t.Logf("write-failure branch not exercised: no 'true' binary on PATH: %v", err)
		return
	}
	for i := 0; i < 50; i++ {
		res := ProbeStdio(trueBin, nil)
		if res.Status != "unhealthy" {
			t.Errorf("expected unhealthy for exiting server, got %q (error: %s)", res.Status, res.Error)
		}
		if strings.Contains(res.Error, "write:") {
			t.Logf("write-failure branch covered after %d extra attempt(s)", i+1)
			return
		}
	}
	t.Log("write-failure branch not observed in 50 attempts: child-exit race did not fire; " +
		"branch is not deterministically reachable (documented in TestProbeStdio_WriteFailure)")
}

func TestProbeStdio_Healthy(t *testing.T) {
	cmd, args := helperInvocation(t, "respond", "")
	result := ProbeStdio(cmd, args)
	if result.Status != "healthy" {
		t.Errorf("expected healthy, got %q (error: %s)", result.Status, result.Error)
	}
	if result.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", result.LatencyMs)
	}
}

func TestProbeStdio_NoResultField(t *testing.T) {
	cmd, args := helperInvocation(t, "noresult", "")
	result := ProbeStdio(cmd, args)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %q (error: %s)", result.Status, result.Error)
	}
	if !strings.Contains(result.Error, "no result field") {
		t.Errorf("expected 'no result field' error, got %q", result.Error)
	}
}
