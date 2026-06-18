package containers

import (
	"testing"

	"github.com/docker/docker/client"
)

// TestList_noDocker verifies that List() returns gracefully when Docker is not
// running — nil containers and a non-empty notes slice explaining the situation.
func TestList_noDocker(t *testing.T) {
	// Override the Docker host to a nonexistent socket so the ping fails.
	t.Setenv("DOCKER_HOST", "unix:///nonexistent/docker.sock")

	containers, notes := List()

	if containers != nil {
		t.Errorf("expected nil containers when Docker is unavailable, got %v", containers)
	}
	if len(notes) == 0 {
		t.Error("expected at least one note explaining Docker is unavailable")
	}
	t.Logf("notes: %v", notes)
}

// TestList_dockerAvailable is an integration test that only passes when Docker
// is running locally. It is skipped in CI environments without a daemon.
func TestList_dockerAvailable(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker client init failed, skipping:", err)
	}
	if _, err := cli.Ping(t.Context()); err != nil {
		t.Skip("Docker daemon not reachable, skipping:", err)
	}

	containers, notes := List()

	// On a machine with Docker, we may or may not have running containers,
	// but the call should succeed without error notes.
	for _, n := range notes {
		if n != "" {
			t.Logf("note: %s", n)
		}
	}

	// Verify returned containers have valid short IDs (12 chars).
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
