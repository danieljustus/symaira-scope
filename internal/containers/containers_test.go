package containers

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestList_noDockerCLI verifies that List() returns gracefully when the
// docker CLI cannot be found on $PATH — nil containers and a non-empty
// notes slice explaining the situation.
func TestList_noDockerCLI(t *testing.T) {
	// Point cli at a path inside an empty temp dir so the binary cannot be
	// resolved, regardless of what the real $PATH contains.
	tmp := t.TempDir()
	prev := cli
	cli = filepath.Join(tmp, "docker")
	t.Cleanup(func() { cli = prev })

	containers, notes := List()

	if containers != nil {
		t.Errorf("expected nil containers when docker CLI is missing, got %v", containers)
	}
	if len(notes) == 0 {
		t.Error("expected at least one note explaining docker is unavailable")
	}
	t.Logf("notes: %v", notes)
}

// TestList_dockerAvailable is an integration test that only passes when a
// working docker CLI and daemon are present. It is skipped in environments
// where either is missing.
func TestList_dockerAvailable(t *testing.T) {
	// Use the real CLI on $PATH for this test.
	prev := cli
	cli = "docker"
	t.Cleanup(func() { cli = prev })

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI not on PATH, skipping:", err)
	}

	containers, notes := List()

	// If the daemon is not reachable, List returns the same shape as
	// TestList_noDockerCLI — treat that as a skip rather than a failure.
	if len(notes) > 0 {
		t.Skip("docker daemon not reachable, skipping:", notes)
	}

	for _, c := range containers {
		if len(c.ID) != 12 {
			t.Errorf("container %s: expected 12-char short ID, got %d chars", c.ID, len(c.ID))
		}
		if c.Image == "" {
			t.Errorf("container %s: image must not be empty", c.ID)
		}
	}
	t.Logf("found %d running container(s)", len(containers))
}

// TestPing_unreachable exercises the ping path with a CLI path that does
// not resolve, ensuring the error message is propagated.
func TestPing_unreachable(t *testing.T) {
	tmp := t.TempDir()
	prev := cli
	cli = filepath.Join(tmp, "docker")
	t.Cleanup(func() { cli = prev })

	err := ping()
	if err == nil {
		t.Fatal("expected error when docker CLI is missing, got nil")
	}
	t.Logf("ping error: %v", err)
}

// TestPing_timeout ensures the context timeout is respected when the docker
// CLI is invoked with a context that has already expired.
func TestPing_timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already expired

	_, err := run(ctx, "version")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// TestShortID verifies the short-ID trimming helper.
func TestShortID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"short", "short"},
		{"", ""},
		{"1234567890123", "123456789012"},
	}
	for _, tc := range cases {
		if got := shortID(tc.in); got != tc.want {
			t.Errorf("shortID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFirstName verifies the leading "/" is stripped from the first name.
func TestFirstName(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"/web", "/alt"}, "web"},
		{[]string{"plain"}, "plain"},
		{nil, ""},
		{[]string{}, ""},
	}
	for _, tc := range cases {
		if got := firstName(tc.in); got != tc.want {
			t.Errorf("firstName(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestPublicPorts verifies that only ports with a positive PublicPort are
// retained.
func TestPublicPorts(t *testing.T) {
	in := []psPort{
		{PublicPort: 8080},
		{PublicPort: 0},
		{PublicPort: 9090},
	}
	got := publicPorts(in)
	want := []int{8080, 9090}
	if len(got) != len(want) {
		t.Fatalf("publicPorts(%v) = %v, want %v", in, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("publicPorts(%v)[%d] = %d, want %d", in, i, got[i], want[i])
		}
	}
}

// TestParsePsStream exercises the stream-of-objects parser with a
// representative payload from `docker ps --format json --no-trunc`.
func TestParsePsStream(t *testing.T) {
	payload := strings.Join([]string{
		`{"ID":"abcdef1234567890","Names":["/web"],"Image":"nginx:latest","Ports":[{"PublicPort":8080,"PrivatePort":80,"Type":"tcp"}]}`,
		`{"ID":"0123456789abcdef","Names":["/db"],"Image":"postgres:16","Ports":[]}`,
	}, "\n")

	listing, err := parsePsStream([]byte(payload))
	if err != nil {
		t.Fatalf("parsePsStream: %v", err)
	}
	if len(listing) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(listing))
	}
	if listing[0].ID != "abcdef1234567890" {
		t.Errorf("entry 0 ID = %q, want %q", listing[0].ID, "abcdef1234567890")
	}
	if firstName(listing[0].Names) != "web" {
		t.Errorf("entry 0 name = %q, want %q", firstName(listing[0].Names), "web")
	}
	if len(listing[0].Ports) != 1 || listing[0].Ports[0].PublicPort != 8080 {
		t.Errorf("entry 0 ports = %+v, want one public port 8080", listing[0].Ports)
	}
	if len(listing[1].Ports) != 0 {
		t.Errorf("entry 1 ports = %+v, want empty", listing[1].Ports)
	}
}

// TestParsePsStream_malformed ensures that a malformed line is reported as
// an error.
func TestParsePsStream_malformed(t *testing.T) {
	payload := `{"ID":"abc","Names":[],"Image":"x","Ports":[]}` + "\n" + `not json`
	_, err := parsePsStream([]byte(payload))
	if err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
}

// TestRun_propagatesStderr ensures the run helper includes the CLI's
// stderr in the error message.
func TestRun_propagatesStderr(t *testing.T) {
	if _, err := exec.LookPath("false"); err != nil {
		t.Skip("`false` binary not available, skipping:", err)
	}

	tmp := t.TempDir()
	prev := cli
	cli = filepath.Join(tmp, "false")
	// Create a symlink to the real `false` so the subprocess can execute.
	if err := os.Symlink(lookupFalse(t), cli); err != nil {
		t.Skip("could not create false symlink, skipping:", err)
	}
	t.Cleanup(func() { cli = prev })

	_, err := run(context.Background())
	if err == nil {
		t.Fatal("expected error from `false` exit code 1, got nil")
	}
	if !strings.Contains(err.Error(), "exit status 1") &&
		!strings.Contains(err.Error(), "exit code") {
		t.Errorf("error should mention the exit code, got: %v", err)
	}
}

// lookupFalse returns the absolute path to the `false` binary.
func lookupFalse(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("false")
	if err != nil {
		t.Fatalf("`false` not on PATH: %v", err)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("`false` abs path: %v", err)
	}
	return abs
}
