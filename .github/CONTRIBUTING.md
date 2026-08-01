# Contributing to symaira-scope

Thank you for helping improve `symaira-scope`, the cross-platform Go CLI and
MCP server for inspecting local listening ports, Docker-published ports, and
AI-client MCP configurations.

## Before You Start

- Check existing issues and pull requests before opening a new one.
- For security vulnerabilities, use the private process in
  [SECURITY.md](SECURITY.md), not a public issue.
- For bugs and feature requests, use the repository's issue forms and include
  enough context for someone else to reproduce or evaluate the request.
- Never include passwords, API keys, access tokens, private configuration
  values, or other secrets in issues, pull requests, logs, or test fixtures.

## Development Workflow

1. Fork or clone the repository and create a focused branch from `main`.
2. Keep changes scoped to one problem. Preserve the CLI's JSON/snake_case
   contract and the repository's read-only discovery behavior unless a change
   explicitly requires otherwise.
3. Run formatting and the relevant checks locally.
4. Open a pull request that explains the change, its scope, verification, and
   related issue(s).

The Go module requires Go 1.26.4 or a compatible newer Go toolchain. The main
implementation is under `cmd/` and `internal/`; the optional macOS client is
under `client/`.

## Required Verification

Before submitting a pull request, run:

```bash
gofmt -w cmd internal
git diff --check
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
go vet ./...
```

If formatting a broad tree would touch unrelated work, format only the Go
files you changed and confirm that `gofmt -l` reports no changed Go files.

GitHub pull requests run the schema-alignment check and the Ubuntu Go build,
test, and vet checks. The build is required to remain CGO-free. macOS and
Windows builds are exercised on main pushes and the scheduled full suite; the
currently disabled lint job is not a required pull-request check while the
project moves to Go 1.26-compatible tooling.

## Optional macOS Client Checks

On macOS with Xcode installed, the client package can be checked without a
private repository, Apple Developer account, signing credential, or other
private prerequisite:

```bash
cd client
swift test
xcodegen generate
```

`swift test` uses the public, exact-pinned `symaira-appkit` package. Running
`xcodegen generate` refreshes the local Xcode project; do not commit generated
or unrelated project changes unless they are part of the contribution.

## Pull Requests

- Use a clear title and describe the user-facing or maintenance impact.
- Keep the diff reviewable and update documentation/templates when behavior
  or contributor workflow changes.
- Include tests or a concise explanation when tests are not applicable.
- Link the issue with `Closes #N`, `Fixes #N`, or `Refs #N` as appropriate.
- Respond to review feedback and keep the branch up to date when requested.

By participating, you agree to follow the [Code of Conduct](../CODE_OF_CONDUCT.md).
