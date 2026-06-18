# symaira-scope Architecture

## Overview

`symscope` answers "what's running and configured on this machine, for AI dev
work?" — listening ports, MCP servers across AI clients, and (soon) containers.
It is a CLI and an MCP server sharing one core.

```
cmd/symscope (cobra)  →  internal/* services  →  corekit (mcpserver/configkit/…)
```

## Packages

- `internal/ports` — enumerates listening sockets via **gopsutil** (cross-platform,
  no `lsof`/`netstat` shell-out), suggests free TCP ports, and detects ports held
  by more than one process.
- `internal/mcpcfg` — reads well-known AI-client config files and extracts their
  `mcpServers` entries (read-only, no network). `DefaultSources()` is the client
  table; `FoundClients()` reports presence.
- `internal/containers` — **stub**; future Docker discovery via the official
  `docker/docker` client.
- `internal/scan` — aggregates the above into one `model.Snapshot`.
- `internal/mcptools` — registers MCP tools on a `corekit/mcpserver.Server` and
  serves stdio with graceful shutdown.
- `internal/config` — `corekit/configkit` loader (`~/.config/symscope`, `SYMSCOPE_*`).
- `internal/model` — snake_case JSON DTOs.

## Why Go (not Rust)

`symscope` is the Go rewrite of the Rust `OpenScope` prototype. It's a port/MCP
inventory CLI that calls OS APIs — no Rust-specific advantage was being used,
while Go lets it reuse the ecosystem's shared `corekit` (MCP transport, config,
exit codes, logging, update checks) and the shared GoReleaser/Homebrew pipeline.
That collapses ~70% of the boilerplate and makes it a first-class family member.

## corekit reuse map

| Concern | corekit |
|---|---|
| MCP stdio JSON-RPC | `mcpserver` |
| Config (TOML + XDG + env) | `configkit` |
| Exit codes / typed errors | `exitcodes` |
| Structured logging (stderr) | `logkit` |
| Update check | `updatecheck` |

## Distribution

CGO-free build → GoReleaser → Homebrew tap, cross-compiled for macOS/Linux/Windows.
