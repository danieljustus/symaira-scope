// Package scan aggregates the individual inventories into one snapshot.
package scan

import (
	"sync"
	"time"

	"github.com/danieljustus/symaira-scope/internal/containers"
	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/model"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

// Build produces a full inventory snapshot. The port, MCP-config, and
// container collectors run concurrently so that a slow or unreachable Docker
// daemon does not block the entire scan.
func Build() (model.Snapshot, error) {
	var (
		mu             sync.Mutex
		allNotes       []string
		portsResult    []model.Port
		portsErr       error
		serversResult  []model.MCPServer
		serversNotes   []string
		contResult     []model.Container
		contNotes      []string
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		p, err := ports.ListListening()
		mu.Lock()
		defer mu.Unlock()
		portsResult = p
		portsErr = err
	}()

	go func() {
		defer wg.Done()
		s, n := mcpcfg.Discover(mcpcfg.DefaultSources())
		mu.Lock()
		defer mu.Unlock()
		serversResult = s
		serversNotes = n
	}()

	go func() {
		defer wg.Done()
		c, n := containers.List()
		mu.Lock()
		defer mu.Unlock()
		contResult = c
		contNotes = n
	}()

	wg.Wait()

	if portsErr != nil {
		return model.Snapshot{}, portsErr
	}

	allNotes = append(serversNotes, contNotes...)

	return model.Snapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Ports:       portsResult,
		MCPServers:  serversResult,
		Containers:  contResult,
		Notes:       allNotes,
	}, nil
}
