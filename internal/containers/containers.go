// Package containers discovers running containers and their published ports
// by shelling out to the local Docker CLI.
package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// cmdTimeout is the maximum time any single docker CLI call is allowed to
// take before being killed.
const cmdTimeout = 5 * time.Second

// cli is the path to the docker binary. Exposed as a var so tests can stub
// it out without touching the global $PATH.
var cli = "docker"

// List returns running containers and their published ports.
//
// When the docker CLI is missing, the daemon is unreachable, or the call
// fails for any reason, it returns nil containers and a non-empty notes
// slice explaining the situation. Callers never have to handle a Docker
// error as a hard failure.
func List() ([]model.Container, []string) {
	if err := ping(); err != nil {
		return nil, []string{"Docker not reachable: " + err.Error()}
	}

	raw, err := psJSON()
	if err != nil {
		return nil, []string{"Docker container list failed: " + err.Error()}
	}

	listing, err := parsePsStream(raw)
	if err != nil {
		return nil, []string{"Docker output parse failed: " + err.Error()}
	}

	out := make([]model.Container, 0, len(listing))
	for _, c := range listing {
		out = append(out, model.Container{
			ID:    shortID(c.ID),
			Name:  firstName(c.Names),
			Image: c.Image,
			Ports: publicPorts(c.Ports),
		})
	}

	return out, nil
}

// ping runs `docker version` and returns nil when the CLI is reachable and
// the daemon is responsive.
func ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	_, err := run(ctx, "version", "--format", "{{.Server.Version}}")
	return err
}

// psJSON returns the JSON output of `docker ps --format json --no-trunc`.
// The CLI emits one JSON object per line with no array wrapper.
func psJSON() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	return run(ctx, "ps", "--format", "json", "--no-trunc")
}

// run executes the docker CLI with the given args and returns its combined
// stdout/stderr. It uses the package-level `cli` variable so tests can stub
// the binary path.
func run(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, cli, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// shortID trims a container ID to its first 12 characters, matching the
// short format `docker ps` displays.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// firstName returns the first container name with the leading "/" stripped,
// or an empty string if the slice is empty.
func firstName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

// publicPorts extracts the host-side (public) port numbers from a slice of
// port mappings. Entries without a public port (e.g. internal-only
// exposures) are dropped.
func publicPorts(ports []psPort) []int {
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort > 0 {
			out = append(out, p.PublicPort)
		}
	}
	return out
}

// psEntry is one line of `docker ps --format json`. Only the fields we
// actually consume are declared.
type psEntry struct {
	ID    string   `json:"ID"`
	Names []string `json:"Names"`
	Image string   `json:"Image"`
	Ports []psPort `json:"Ports"`
}

// psPort is the nested port object inside a psEntry.
type psPort struct {
	PublicPort int `json:"PublicPort"`
}

// parsePsStream decodes the stream-of-objects output `docker ps --format json`
// emits (one JSON object per line, no array wrapper). It returns all entries
// it successfully decoded; an error is returned only when a line cannot be
// parsed, in which case the partially-decoded entries are still returned.
func parsePsStream(raw []byte) ([]psEntry, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	out := []psEntry{}
	for dec.More() {
		var entry psEntry
		if err := dec.Decode(&entry); err != nil {
			return out, err
		}
		out = append(out, entry)
	}
	return out, nil
}
