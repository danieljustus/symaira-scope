# symaira-scope

> Inventory local ports, containers, and MCP servers — for AI dev environments.

`symscope` shows, from one place: what's **listening** on your machine, which
**MCP servers** your AI clients (Claude Desktop/Code, Cursor, VS Code, Windsurf,
project-local) have configured, and — soon — your **Docker-published** ports. It
runs as a CLI and as an **MCP server**, so an agent can ask "what's on port 3000?"
or "give me three free ports" itself.

Part of the [Symaira](../ECOSYSTEM.md) family (Go core, MIT, corekit-based).

> **Status: v0.1 scaffold.** Builds, tests pass. Ports/MCP-discovery/conflicts/
> free-port-suggest work; container discovery is an honest stub. Not released yet.
> This is the Go successor to the earlier Rust `OpenScope` prototype.

## Install (from source)

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
symscope clients list      # which AI clients have an MCP config present
symscope containers        # running containers (stub in v0.1)
symscope conflicts         # ports bound by more than one process
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

Tools: `scan`, `ports_list`, `ports_suggest`, `mcp_list`, `conflicts`.

## Documentation

- [docs/architecture.md](docs/architecture.md) — design & data flow
- [docs/roadmap.md](docs/roadmap.md) — built vs planned
- [AGENTS.md](AGENTS.md) — contributor/agent guidance

## License

MIT © 2026 Daniel Justus.
