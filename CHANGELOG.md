# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] — scaffold

Go successor to the Rust `OpenScope` prototype. Builds and tests pass; reads work,
container discovery is stubbed.

### Added
- CLI (cobra): `scan`, `ports list|suggest`, `mcp list`, `clients list`,
  `containers`, `conflicts`, `serve`, `version`.
- Listening-port inventory with owning process via gopsutil (no lsof/netstat
  shell-out), free-port suggestion, multi-process port conflict detection.
- MCP-server discovery across Claude Desktop/Code, Cursor, VS Code, Windsurf, and
  project-local `.mcp.json`.
- MCP stdio server (`serve`) via `corekit/mcpserver`: tools `scan`, `ports_list`,
  `ports_suggest`, `mcp_list`, `conflicts`.
- corekit integration: `configkit`, `exitcodes`, `logkit`, `updatecheck`.

### Not yet implemented (see docs/roadmap.md)
- Docker container/port discovery (official docker client).
- Snapshot caching, `explain`, more AI clients, richer conflict analysis.
