# Agent Instructions — symaira-scope

Cross-platform Go CLI **and** MCP server that inventories local listening ports,
Docker-published ports, and the MCP servers configured across AI clients. Public
repo, MIT-licensed. Part of the Symaira family — see `../AGENTS.md` /
`../ECOSYSTEM.md` for cross-repo conventions.

## Build & Test

```bash
go build ./...
go test ./...
go run ./cmd/symscope scan
```

## Layout

```
cmd/symscope/           CLI entrypoint (cobra)
internal/model/         data types (snake_case JSON)
internal/ports/         listening ports + free-port suggest + conflicts (gopsutil)
internal/mcpcfg/        MCP-server discovery across AI client configs
internal/containers/    container discovery (stub → docker client)
internal/scan/          aggregate snapshot
internal/mcptools/      MCP tool registration (corekit/mcpserver)
internal/config/        configkit loader
```

## Hard Rules

- **CGO-free** (`CGO_ENABLED=0`) for clean cross-compilation (macOS/Linux/Windows).
- **Reuse corekit, don't re-implement**: MCP transport (`mcpserver`), config
  (`configkit`), exit codes/errors (`exitcodes`), logging (`logkit`), update
  checks (`updatecheck`). Never vendor or fork these.
- **Never commit a `replace ../symaira-corekit`** in go.mod — it breaks CI. Pin
  the published version (currently `v0.1.1`).
- **Zero stdout pollution in `serve`**: stdout carries only JSON-RPC frames; all
  logs go to stderr via slog (`logkit`).
- **Read-only & local**: discovery never mutates client configs and makes no
  network calls (except the explicit `version --check`).
- **JSON is snake_case**; keep Go fields idiomatic with `json:"..."` tags.

## Conventions (ecosystem)

- Binary: `symscope`. Config: `~/.config/symscope/config.toml`, env `SYMSCOPE_*`.
- Exit codes via `corekit/exitcodes` (wrap errors with `exitcodes.Wrap`).
- Release: GoReleaser → `../homebrew-tap` (mirror symfetch/symseek).

See `docs/roadmap.md` for built-vs-planned and `docs/architecture.md` for design.

## macOS Client (`client/`)

- SwiftUI app: local SPM package `SymscopeClient` (SymscopeKit) + XcodeGen app
  target (`cd client && xcodegen generate`, scheme `Symscope`). Local builds
  need `DEVELOPER_DIR` pointing at Xcode (CommandLineTools lack XCTest).
- Depends on the shared **symaira-appkit** package, pinned exact (`0.1.0`) in
  BOTH `client/Package.swift` (SymscopeKit → SymairaCLIRunner, SymairaToolKit)
  and `client/project.yml` (app → SymairaTheme). Keep the two pins in sync.
- `Views/Styles.swift` sources shared brand tokens from SymairaTheme; the
  scope-specific values (bgCard/borderGlass variants, glows) and modifiers
  stay local on purpose for pixel-identical rendering.
- Do not reintroduce app-local Process/Theme plumbing; extend symaira-appkit
  instead. Migration context: see `../docs/symaira-appkit-migration.md`
  (Welle 1).
