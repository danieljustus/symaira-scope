// Package containers discovers running containers and their published ports.
package containers

import "github.com/danieljustus/symaira-scope/internal/model"

// List returns running containers and their published ports.
//
// STATUS: stub (v0.1). A real implementation will use the official Docker Go
// client (github.com/docker/docker/client) against the local Docker-compatible
// socket. See docs/roadmap.md.
func List() ([]model.Container, []string) {
	return nil, []string{
		"Docker discovery not yet implemented; planned via the official docker/docker client.",
	}
}
