package model

// MCPServerManifest is the --format manifest view of a discovered MCP server,
// shaped after the MCP registry's server.json packages/transport split but
// scoped to what symscope can observe locally (no namespace ownership, no
// versioned schema URLs).
type MCPServerManifest struct {
	Name                 string            `json:"name"`
	Client               string            `json:"client"`
	ConfigPath           string            `json:"config_path"`
	Transport            ManifestTransport `json:"transport"`
	Packages             []ManifestPackage `json:"packages"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
}

// ManifestTransport mirrors server.json's transport block.
type ManifestTransport struct {
	Type string `json:"type"` // stdio | http
}

// ManifestPackage mirrors server.json's packages[] entry for a locally
// discovered server. Version is omitted because local config discovery
// cannot observe a package version.
type ManifestPackage struct {
	RegistryType string `json:"registryType"` // always "local-binary" for local discovery
	Identifier   string `json:"identifier"`   // the configured server name
	Version      string `json:"version,omitempty"`
}
