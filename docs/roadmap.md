# Roadmap

## v0.1 — scaffold (current)

- [x] CLI surface (scan/ports/mcp/clients/containers/conflicts/serve/version).
- [x] Listening-port inventory + process (gopsutil), free-port suggest, conflicts.
- [x] MCP discovery across Claude Desktop/Code, Cursor, VS Code, Windsurf, project.
- [x] MCP stdio server via corekit/mcpserver.
- [x] corekit: configkit/exitcodes/logkit/updatecheck.

## v0.2 — depth

- [ ] **Docker discovery** via the official `docker/docker` client (containers +
      published ports); feed into `conflicts`.
- [ ] **Snapshot cache** (corekit/fsutil atomic write + advisory lock, 5 min TTL,
      `--no-cache`).
- [ ] **More AI clients**: Cline, Continue, Goose, Aider, Roo Code, Zed; honor
      VS Code `.vscode/mcp.json` and per-workspace files.
- [ ] **Richer conflicts**: config-vs-runtime (a configured MCP/Docker port that's
      already occupied), not just multi-process bindings.
- [ ] `explain port <n>` / `explain server <name>` detail commands.
- [ ] Table/`--json` output modes (JSON is the current default).

## v0.3 — ecosystem synergy

- [ ] Power `symaira-terminal`'s MCP-hub: expose discovery so the terminal app's
      one-click "register vault/memory/seek/fetch" feature reuses symscope.
- [ ] `vault://` awareness when surfacing MCP server secrets/env.
- [ ] MCP health probe (does a configured stdio/http server actually start/respond?).

## Infra

- [ ] GoReleaser + Homebrew cask/formula in `../homebrew-tap`.
- [ ] golangci-lint config + CI gate.
- [ ] Windows/Linux CI matrix (cross-platform port backends).
