// Package containers discovers running containers and their published ports.
package containers

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/danieljustus/symaira-scope/internal/model"
)

// List returns running containers and their published ports.
//
// When Docker is not running or unavailable it returns nil containers and a
// note so callers never fail on a missing daemon.
func List() ([]model.Container, []string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, []string{"Docker client init failed: " + err.Error()}
	}

	ctx := context.Background()

	// Ping the daemon to verify it is reachable.
	if _, err := cli.Ping(ctx); err != nil {
		return nil, []string{"Docker not running or not reachable: " + err.Error()}
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, []string{"Docker container list failed: " + err.Error()}
	}

	out := make([]model.Container, 0, len(containers))
	for _, c := range containers {
		// Container names are prefixed with "/" — strip it.
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		// Collect published (host-mapped) port numbers.
		ports := make([]int, 0, len(c.Ports))
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, int(p.PublicPort))
			}
		}

		out = append(out, model.Container{
			ID:    c.ID[:12], // short ID like docker ps
			Name:  name,
			Image: c.Image,
			Ports: ports,
		})
	}

	return out, nil
}
