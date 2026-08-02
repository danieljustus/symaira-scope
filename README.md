# symaira-scope

[![CI](https://github.com/danieljustus/symaira-scope/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/danieljustus/symaira-scope/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/danieljustus/symaira-scope)](https://github.com/danieljustus/symaira-scope/releases/latest)
[![License](https://img.shields.io/github/license/danieljustus/symaira-scope)](LICENSE)

> Inventory local ports, containers, and MCP servers — for AI dev environments.

`symscope` shows, from one place: what's **listening** on your machine, which
**MCP servers** your AI clients (Claude Desktop/Code, Cursor, VS Code, Windsurf,
project-local) have configured, and your **Docker-published** ports. It
runs as a CLI and as an **MCP server**, so an agent can ask "what's on port 3000?"
or "give me three free ports" itself.

Part of the [Symaira](https://github.com/danieljustus) family (Go core, Apache-2.0, corekit-based).

> **Status: v0.3.1** — MCP tool results are now schema-valid for strict
> clients (TextContent serialization fix in corekit v0.8.0), surgical JSONC
> config editing for AI-client MCP configs (`mcp add`/`mcp remove`), a shared
> Reporter interface for JSON/NDJSON output, and a branded DMG for the macOS
> app. Cross-platform (macOS, Linux, Windows).

## Install

### Homebrew

```bash
brew install danieljustus/tap/symscope
```

### Go

Install the latest version from source:

```bash
go install github.com/danieljustus/symaira-scope/cmd/symscope@latest
```

The `symscope` binary is installed to Go's binary directory, which should be on
your `PATH`.

### Prebuilt releases

Download the archive for macOS, Linux, or Windows from the
[latest release](https://github.com/danieljustus/symaira-scope/releases/latest).

### Build from source

```bash
git clone https://github.com/danieljustus/symaira-scope && cd symaira-scope
go build -o symscope ./cmd/symscope
./symscope scan
```

## CLI

```text
symscope scan              # full snapshot: ports + MCP servers + containers (JSON)
symscope ports list        # listening TCP/UDP ports + owning process
symscope ports suggest     # free TCP ports  (--count --from --to)
symscope mcp list          # MCP servers discovered across AI clients
symscope mcp list --check-credentials  # flag env values that look like exposed secrets
symscope clients list      # which AI clients have an MCP config present
symscope containers        # running containers with published ports
symscope conflicts         # ports bound by more than one process
symscope watch             # stream ports/conflicts/MCP-config changes as NDJSON (--interval)
symscope serve             # run the MCP stdio server for agents
symscope version [--check]
```

Example:

```bash
$ symscope ports suggest --count 3
{ "free": [3000, 3001, 3002] }
```

## MCP integration

Register `symscope serve` with any MCP host:

```json
{ "mcpServers": { "symscope": { "command": "/abs/path/symscope", "args": ["serve"] } } }
```

Tools: `scan`, `ports_list`, `ports_suggest`, `mcp_list`, `conflicts`, `mcp_health`.

`scan` and `mcp list --check-credentials` include a `credential_warnings` field per
server when an `env` value looks like an exposed API key or token (e.g. `sk-...`,
`ghp_...`, JWTs, or long high-entropy strings). `vault://` references and obvious
placeholders are ignored.

## Documentation

- [AGENTS.md](AGENTS.md) — contributor/agent guidance
- [Releases](https://github.com/danieljustus/symaira-scope/releases) — prebuilt binaries and release notes

## Test

Run the full Go test suite with:

```bash
go test ./...
```

## License

Apache-2.0 © 2026 Daniel Justus. See [LICENSE](LICENSE).
