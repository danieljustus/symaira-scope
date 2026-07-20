## What's changed

### Features
- #53 Add native macOS Swift GUI client (obsidian/gold styling)
- #56 Expose `SymscopeFeature` module for embedding the client in symaira-hub
- #60 Integrate `versionkit` and add `version --json`
- #61 Bump hub `expectedSchemaVersion` to 1 (versionkit handshake)
- #63 Add `watch` mode: stream ports/conflicts/MCP-config changes as NDJSON — closes #55
- #68 Detect likely-exposed credentials in MCP server env blocks — closes #66
- #69 Discover six more AI clients (Kiro, Qoder, Copilot CLI, LM Studio, Google Antigravity, Gemini CLI) and add a generic `--files` schema fallback — closes #64, #65

### Docs
- #70 Clarify that discovery stays read-only; `mcp add`/`mcp rm` is the intentional mutation surface — closes #67
- #73 Document the `watch` command in the README and refresh the status line — closes #71, #72

### Dependencies
- #58 Bump `symaira-appkit` to v0.1.2 (client)
- #62 Bump `symaira-appkit` to v0.2.0 (client)
- #74 Bump `symaira-corekit` to v0.5.0 and `gopsutil` to v4.26.6
- #75 Bump `actions/setup-go` to v7

### Closed Issues
- #55 watch mode
- #64 Discover MCP config for Kiro, Qoder, Copilot CLI, LM Studio, Google Antigravity, Gemini CLI
- #65 Generic `--files` config schema fallback for unsupported/custom MCP clients
- #66 Detect likely-exposed credentials in MCP server env blocks
- #67 AGENTS.md and docs/architecture.md still claim symscope is strictly read-only
- #71 Document the watch command in the README
- #72 Refresh the README status line for the next release

**Full Changelog**: https://github.com/danieljustus/symaira-scope/compare/v0.1.2...v0.2.0
