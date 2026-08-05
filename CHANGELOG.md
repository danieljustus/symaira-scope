# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.2] — 2026-08-05

### Fixed
- Fixed Windows configuration tests to isolate `USERPROFILE` alongside `HOME`.
- Fixed MCP config editing when adding a server to a document without an existing `mcpServers` key.

### Changed
- Centralized the CLI and MCP health-probe version source and aligned the macOS client with `symaira-appkit` v0.7.0.
- Added CLI/config and stdio health-probe coverage, raising overall Go coverage to 80.2%.
- Expanded the README with a terminal demo and a comparison of symscope's local discovery workflows.

### Infrastructure
- Hardened the macOS release workflow with Xcode 16.4 validation and an archive smoke test.

## [0.3.1] — 2026-08-01

### Fixed
- Hardened macOS release signing, App Store Connect notarization, embedded-engine signing, and DMG publishing.

### Changed
- Refreshed the private `symaira-appkit` resolution used by the macOS client to v0.2.0.

### Infrastructure
- Re-enabled CGO-free Go linting with the golangci-lint v2 configuration.
- Pinned the CodeQL action to v4.37.4 and configured a Dependabot cooldown.
- Added contributor guidance, issue forms, a pull-request template, and a Code of Conduct.
- Improved README onboarding and installation instructions.

## [0.3.0] — 2026-07-31

### Added
- Native macOS Swift GUI client, styled in obsidian and gold, embeddable as a module in symaira-hub.
- `watch` mode for streaming ports, conflicts, and MCP-config changes as NDJSON.
- Credential-leak detection for MCP server environment blocks.
- MCP config discovery for Kiro, Qoder, Copilot CLI, LM Studio, Google Antigravity, and Gemini CLI, plus a generic `--files` schema fallback.
- `versionkit` handshake integration and the `version --json` flag.

### Changed
- Discovery and read paths remain read-only and network-free; `mcp add` and `mcp rm` are the intentional config-mutation surface.
- Bumped `symaira-corekit` to v0.5.0 and `gopsutil` to v4.26.6.

### Infrastructure
- Added the native macOS GUI client module and branded DMG release asset.
- Planning documentation moved from the repository into GitHub Issues and Milestones.
- Bumped `actions/setup-go` to v7.

## [0.2.0] — 2026-07-20

### Added
- Native macOS Swift GUI client (`client/`), styled in obsidian and gold, embeddable as a module in symaira-hub
- `watch` mode: stream ports/conflicts/MCP-config changes as NDJSON (`--interval`)
- Credential-leak detection for MCP server env blocks (`mcp list --check-credentials`, `scan` includes `credential_warnings`)
- MCP config discovery for six more AI clients (Kiro, Qoder, Copilot CLI, LM Studio, Google Antigravity, Gemini CLI) plus a generic `--files` schema fallback
- `versionkit` handshake integration and `version --json` flag

### Changed
- `AGENTS.md` clarified: discovery/read paths stay read-only and network-free; `mcp add`/`mcp rm` remain the intentional, atomic config-mutation surface
- Bumped `symaira-corekit` to v0.5.0 and `gopsutil` to v4.26.6

### Infrastructure
- `docs/` is no longer tracked in git; planning docs live in GitHub Issues/Milestones
- `actions/setup-go` bumped to v7

## [0.1.2] — 2026-07-02

### Added
- Homebrew formula generation in GoReleaser config

### Changed
- Bump symaira-corekit to v0.2.1

### Infrastructure
- Sign and notarize macOS binaries with Developer ID
- Use canonical Apache-2.0 license text

## [0.1.1] — 2026-06-22

### Fixed
- Config permissions and `ports suggest` flag wiring (correctly reads config defaults)
- `mcp add` validation now requires at least `--command` or `--url`
- `mcp_health` tool is opt-in by default (returns "unknown" unless `probe=true`)
- Cache stats command deprecated in favor of `cache show`

### Changed
- Container discovery uses local Docker CLI instead of Docker SDK
- Scan collects ports, MCP servers, and containers concurrently
- Free port suggestion uses atomic allocation for better parallel performance

## [0.1.0] — 2026-06-18

First public release. Go CLI + MCP server that inventories local listening ports,
Docker-published ports, and MCP servers configured across AI clients.

### Features
- Full port inventory: listening TCP/UDP ports with owning process via gopsutil
- Free port suggestion (`ports suggest --count N --from --to`)
- Docker container discovery via local Docker CLI
- MCP server discovery across Claude Desktop/Code, Cursor, VS Code, Windsurf,
  Goose, Cline, Continue, Aider, Roo Code, Zed, and project-local `.mcp.json`
- MCP stdio server (`serve`) with tools: `scan`, `ports_list`, `ports_suggest`,
  `mcp_list`, `conflicts`
- Port conflict detection (multi-process + MCP-occupied)
- Snapshot caching with atomic writes, TTL, and advisory lock
- Explain commands (`explain port`, `explain server`) for human-readable output
- MCP hub commands (`mcp add`, `mcp remove`) for client config management
- MCP health probe with stdio and HTTP support
- Cross-platform CI (Ubuntu, macOS, Windows) and GoReleaser config

### Fixed
- Config atomic writes with backup
- Health probe command sanitization and trust model
- Config parsing errors now include client context
- Port-to-holder deduplication in conflict detection
- Parallelized free port scanning

### Infrastructure
- GoReleaser config for cross-platform builds (darwin/linux/windows × amd64/arm64)
- golangci-lint config (lint job temporarily disabled for Go 1.26 compatibility)
- Dependabot for Go modules and GitHub Actions
- CodeQL security analysis workflow
