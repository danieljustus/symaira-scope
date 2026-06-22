# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
