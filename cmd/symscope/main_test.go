package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	ver "github.com/danieljustus/symaira-scope/internal/version"
)

// captureStdout redirects os.Stdout for the duration of the test and returns
// a function that reads back everything written to it since the redirect.
func captureStdout(t *testing.T) func() string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = orig
		_ = w.Close()
		_ = r.Close()
	})
	return func() string {
		_ = w.Close()
		data, _ := io.ReadAll(r)
		return string(data)
	}
}

// runCLI executes the root command in-process with the given args and
// returns everything written to os.Stdout plus the error (nil on success;
// main() maps nil to exit code 0).
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	read := captureStdout(t)
	err := cmd.Execute()
	return read(), err
}

func isolateCLIHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return home
}

func expectCLIError(t *testing.T, args []string, wantCode exitcodes.ExitCode, wantMessage string) {
	t.Helper()
	_, err := runCLI(t, args...)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if code := exitcodes.ExitCodeFromError(err); code != wantCode {
		t.Errorf("exit code = %d, want %d", code, wantCode)
	}
	if !strings.Contains(err.Error(), wantMessage) {
		t.Errorf("error %q does not contain %q", err, wantMessage)
	}
}

func runIsolatedCLIProcess(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	return runCLIProcess(t, []string{
		"HOME=" + t.TempDir(),
		"XDG_CACHE_HOME=" + t.TempDir(),
	}, args...)
}

func TestVersion(t *testing.T) {
	out, err := runCLI(t, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(out, ver.Version) {
		t.Errorf("output %q does not contain version %q", out, ver.Version)
	}
}

func TestVersionJSON(t *testing.T) {
	out, err := runCLI(t, "version", "--json")
	if err != nil {
		t.Fatalf("version --json failed: %v", err)
	}
	var got struct {
		Tool          string `json:"tool"`
		Version       string `json:"version"`
		SchemaVersion int    `json:"schema_version"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("version --json output %q is not valid JSON: %v", out, err)
	}
	if got.Tool != "symscope" {
		t.Errorf("tool = %q, want %q", got.Tool, "symscope")
	}
	if got.Version != ver.Version {
		t.Errorf("version = %q, want %q", got.Version, ver.Version)
	}
	if got.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", got.SchemaVersion)
	}
}

func TestWatchValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unsupported format", []string{"watch", "--format", "json"}, `unsupported format "json"`},
		{"non-positive interval", []string{"watch", "--interval", "0s"}, "interval must be greater than 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectCLIError(t, tt.args, exitcodes.ExitConfig, tt.wantErr)
		})
	}
}

func TestMCPAddRemoveValidation(t *testing.T) {
	// Isolated HOME: DefaultSources() resolves every known client path under
	// this temp dir, and any file a buggy validation path might write would
	home := isolateCLIHome(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"add unknown client", []string{"mcp", "add", "--name", "srv", "--client", "not-a-client", "--command", "echo"}, `unknown client "not-a-client"`},
		{"add missing command and url", []string{"mcp", "add", "--name", "srv", "--client", "claude-code"}, "at least one of --command or --url is required"},
		{"remove unknown client", []string{"mcp", "remove", "--name", "srv", "--client", "not-a-client"}, `unknown client "not-a-client"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectCLIError(t, tt.args, exitcodes.ExitConfig, tt.wantErr)
			// Validation must fail before any config file is written.
			entries, err := os.ReadDir(home)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("validation failure wrote files under HOME: %v", entries)
			}
		})
	}
}

func TestPortsSuggestConfigFallback(t *testing.T) {
	// A broken SYMSCOPE_* env value makes config.Load() fail; the suggest
	// command must fall back to defaults (ports.go lines 37-41) and succeed.
	isolateCLIHome(t)
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_FROM", "not-an-int")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_TO", "")
	_, err := runCLI(t, "ports", "suggest", "--count", "1")
	if err != nil {
		t.Fatalf("ports suggest with broken config should fall back to defaults: %v", err)
	}
}

func TestCommandOutputCommands(t *testing.T) {
	for _, args := range [][]string{
		{"cache", "show"},
		{"cache", "stats"},
		{"clients", "list"},
		{"containers"},
		{"ports", "list"},
		{"explain", "port", "--number", "1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, stderr, code := runIsolatedCLIProcess(t, args...)
			if code != 0 {
				t.Fatalf("%v failed with exit code %d (stderr: %s)", args, code, stderr)
			}
			var value any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &value); err != nil {
				t.Fatalf("%v output %q is not valid JSON: %v", args, out, err)
			}
		})
	}
}

func TestCacheClearCommand(t *testing.T) {
	out, stderr, code := runIsolatedCLIProcess(t, "cache", "clear")
	if code != 0 {
		t.Fatalf("cache clear failed with exit code %d (stderr: %s)", code, stderr)
	}
	if strings.TrimSpace(out) != "Cache cleared." {
		t.Errorf("cache clear output = %q, want %q", out, "Cache cleared.")
	}
}

func TestExplainServerMissing(t *testing.T) {
	isolateCLIHome(t)
	expectCLIError(t, []string{"explain", "server", "--name", "missing"}, exitcodes.ExitSoftware, "not found")
}

// TestHelperProcess is the re-exec entry point for subprocess-based tests:
// when GO_WANT_HELPER_PROCESS=1 it runs the real CLI (main()) with args from
// SYMSCOPE_TEST_ARGS, so tests can exercise the binary end-to-end in a fresh
// process with an isolated environment. main() never returns (it os.Exits),
// which is exactly the exit-code behaviour under test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	var args []string
	if raw := os.Getenv("SYMSCOPE_TEST_ARGS"); raw != "" {
		args = strings.Split(raw, "\x1f")
	}
	os.Args = append([]string{"symscope"}, args...)
	main()
	// main() only returns on success (errors os.Exit inside main). Exit
	// explicitly so the testing framework cannot append its own output
	// (e.g. "PASS") to the CLI's stdout.
	os.Exit(0)
}

// runCLIProcess re-executes the test binary as a helper process to run the
// real CLI and returns its stdout, stderr, and exit code.
func runCLIProcess(t *testing.T, extraEnv []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "SYMSCOPE_") || strings.ContainsRune(kv, '\x00') {
			continue // never let a developer's real env (or stray NULs) leak into the child
		}
		env = append(env, kv)
	}
	env = append(env, "GO_WANT_HELPER_PROCESS=1", "SYMSCOPE_TEST_ARGS="+strings.Join(args, "\x1f"))
	env = append(env, extraEnv...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run helper process: %v", err)
		}
		code = ee.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}

// writeSymscopeConfig writes $HOME/.config/symscope/config.toml (the global
// config path used by configkit).
func writeSymscopeConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "symscope")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// parseFreePort decodes `ports suggest --count 1` output into the suggested port.
func parseFreePort(t *testing.T, stdout string) int {
	t.Helper()
	var got struct {
		Free []int `json:"free"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &got); err != nil {
		t.Fatalf("ports suggest output %q is not valid JSON: %v", stdout, err)
	}
	if len(got.Free) != 1 {
		t.Fatalf("expected 1 suggested port, got %v", got.Free)
	}
	return got.Free[0]
}

func TestPortsSuggestUsesDefaultRange(t *testing.T) {
	// Isolated HOME with no config file: suggest must use the default
	// 3000-9999 range from config.Defaults().
	home := t.TempDir()
	stdout, stderr, code := runCLIProcess(t, []string{
		"HOME=" + home,
		"XDG_CACHE_HOME=" + t.TempDir(),
	}, "ports", "suggest", "--count", "1")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if port := parseFreePort(t, stdout); port < 3000 || port > 9999 {
		t.Errorf("suggested port %d outside default range [3000, 9999]", port)
	}
}

func TestPortsSuggestEnvWinsOverFile(t *testing.T) {
	// config.toml sets suggest_from=4000/suggest_to=7000; SYMSCOPE_PORTS_
	// SUGGEST_FROM=6000 must win over the file (defaults < file < env).
	home := t.TempDir()
	writeSymscopeConfig(t, home, "[ports]\nsuggest_from = 4000\nsuggest_to = 7000\n")
	stdout, stderr, code := runCLIProcess(t, []string{
		"HOME=" + home,
		"XDG_CACHE_HOME=" + t.TempDir(),
		"SYMSCOPE_PORTS_SUGGEST_FROM=6000",
	}, "ports", "suggest", "--count", "1")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if port := parseFreePort(t, stdout); port < 6000 || port > 7000 {
		t.Errorf("suggested port %d outside [6000, 7000]; env override should win over config.toml", port)
	}
}
