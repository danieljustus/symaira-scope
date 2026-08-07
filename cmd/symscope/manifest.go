package main

import "github.com/danieljustus/symaira-scope/internal/model"

// toManifestEntries reshapes discovered servers into the registry-inspired
// manifest shape (transport/packages/environmentVariables split).
func toManifestEntries(servers []model.MCPServer) []model.MCPServerManifest {
	entries := make([]model.MCPServerManifest, 0, len(servers))
	for _, s := range servers {
		entries = append(entries, model.MCPServerManifest{
			Name:       s.Name,
			Client:     s.Client,
			ConfigPath: s.ConfigPath,
			Transport:  model.ManifestTransport{Type: manifestTransportType(s.Transport)},
			Packages: []model.ManifestPackage{{
				RegistryType: "local-binary",
				Identifier:   s.Name,
			}},
			EnvironmentVariables: s.Env,
		})
	}
	return entries
}

// manifestTransportType maps a discovered transport to the registry's
// stdio|http enum; SSE servers are reached over HTTP.
func manifestTransportType(transport string) string {
	if transport == "http" || transport == "sse" {
		return "http"
	}
	return "stdio"
}
