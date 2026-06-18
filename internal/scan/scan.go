// Package scan aggregates the individual inventories into one snapshot.
package scan

import (
	"time"

	"github.com/danieljustus/symaira-scope/internal/containers"
	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

// Build produces a full inventory snapshot.
func Build() (model.Snapshot, error) {
	p, err := ports.ListListening()
	if err != nil {
		return model.Snapshot{}, err
	}
	servers, notes := mcpcfg.Discover(mcpcfg.DefaultSources())
	c, containerNotes := containers.List()

	allNotes := append(notes, containerNotes...)

	return model.Snapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Ports:       p,
		MCPServers:  servers,
		Containers:  c,
		Notes:       allNotes,
	}, nil
}
