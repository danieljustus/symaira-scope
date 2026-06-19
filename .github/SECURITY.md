# Security Policy

Thank you for helping keep `symaira-scope` and its users safe. This document
explains how to report a vulnerability, what to expect from us, and which
versions are supported.

## Supported Versions

`symaira-scope` follows [Semantic Versioning](https://semver.org/). Security
fixes are released for the following versions:

| Version  | Supported           |
| -------- | ------------------- |
| latest   | :white_check_mark:  |
| < latest | :x:                 |

In practice this means: the most recent release line receives security fixes
and the previous release line only receives fixes for severe issues at the
maintainer's discretion. Older lines are not patched — please upgrade.

## Reporting a Vulnerability

**Please do not open a public GitHub Issue for security-sensitive reports.**

Use [GitHub's private vulnerability reporting](../../security/advisories/new)
on this repository. That opens a private advisory thread visible only to you
and the maintainers, which is the channel we use to coordinate a fix and a
disclosure date.

If you cannot or do not want to use GitHub Advisories, you can email
`security@danieljustus.dev`. Either way, encrypting sensitive details with
the maintainer's PGP key is welcome but not required.

### What to include

A good report gives us enough to reproduce and assess impact:

- The affected version, commit SHA, or release tag.
- A clear description of the issue and its security impact.
- A reproducible proof of concept, or precise steps to reproduce.
- The environment (`symscope scan --help` output, OS, Go version) when relevant.

## What to Expect

Once we receive a report we will:

1. **Acknowledge** within **3 business days** with a tracking reference.
2. **Triage** within **7 business days**: confirm the issue, classify severity,
   and decide on a fix path.
3. **Coordinate a fix and a disclosure date** with you. We aim to ship a
   patched release within **30 days** for high/critical issues and within
   **90 days** for lower-severity issues. Complex issues may take longer; we
   will keep you informed.
4. **Credit** you in the release notes and the security advisory unless you
   ask to remain anonymous.
5. **Disclose** by publishing a GitHub Security Advisory once a fix is
   available, or sooner by mutual agreement.

We follow the principle of **coordinated disclosure**: please give us a
reasonable amount of time to ship a fix before publishing details.

## Scope

In scope:

- Code in this repository (`cmd/`, `internal/`, `docs/`).
- The `symscope` binary and the `symscope serve` MCP server.
- Build, test, and release pipelines under `.github/`.

Out of scope:

- Third-party dependencies. Please report those upstream; we follow
  Dependabot alerts for transitive Go modules and bump promptly.
- Denial-of-service attacks that require local code execution.
- Issues that require a hostile, already-running binary on the user's machine
  (e.g. local privilege escalation inside `symscope` itself is in scope;
  abuse of a symscope-launched subprocess is treated like any other local
  process).

## Safe Harbor

We will not pursue legal action against researchers who:

- Make a good-faith effort to avoid privacy violations, data destruction,
  or service disruption.
- Only interact with accounts they own or have explicit permission to access.
- Stop testing immediately and report a vulnerability once they suspect data
  has been exposed.
- Do not exploit a vulnerability beyond what is necessary to demonstrate it.

## Acknowledgements

We are grateful to everyone who reports vulnerabilities responsibly. Reporters
are credited in the corresponding GitHub Security Advisory and release notes
unless they prefer to remain anonymous.

## Policy Changes

This policy may be updated from time to time. Material changes are tracked in
the git history of this file and noted in the next release's `CHANGELOG.md`.

---

_Last reviewed: 2026-06-19_
