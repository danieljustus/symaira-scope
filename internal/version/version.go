// Package version holds the single source of truth for the symscope version
// string. Release builds inject the real version via ldflags
// (-X ...internal/version.Version=vX.Y.Z); local and `go install` builds
// report "dev".
package version

// Version is the symscope version reported by the CLI and the MCP health
// probe. It is overwritten at build time by GoReleaser and `make install`.
var Version = "dev"
