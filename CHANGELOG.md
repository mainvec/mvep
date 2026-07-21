# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-07-21

### Changed
- Switched license from MPL-2.0 to Apache-2.0.
- Updated root `README.md` with project overview, install, quick start, and repository layout.
- Updated `mvgen/README.md` contributing and license sections for open-source release.
- Renamed `mvgen/gengen_next/` to `mvgen/gengen/` (now the sole active self-generator).

### Added
- GitHub Actions CI workflow (`.github/workflows/test.yml`) running `go test` and `go vet` on push/PR.
- `CHANGELOG.md`.
- `.claude/` and `.DS_Store` entries to `.gitignore` files.
- `// TODO: REWORK` notes to codegen templates that reference `github.com/mainvec/wo/` runtime packages, flagging them for a follow-up rework.

### Removed
- `mvn/` — orphaned WO Node runtime (no consumers, no `go.mod`, not importable).
- `mvgen/gengen/` (legacy) — old self-generator producing `mvpapi_legacy/`.
- `mvgen/mvpapi_legacy/` — old protobuf-based implementation and `mvgen` CLI.
- `mvgen/testold/` — abandoned test fixtures.
- `mvgen/testdata/draft-07/` — old draft-07 schema test fixtures.
- `.gitmodules` (empty), `dev-roadmap.txt`, `mvgen/.claude/` (local settings).
- Skipped `TestGenerateGOSRV` in `mvgen/mvgen_go_test.go` (referenced removed `07_pizzahub.jsonc`).
- Stale `wog/` and `docs/dev-docs/` entries from `.gitignore` files.

### Known limitations
- `go.mod` retains a `replace github.com/mainvec/mvp/mvpgo => ../../mvpgo/mvpgo` directive. The `github.com/mainvec/mvp/mvpgo` module path is currently unresolvable externally (no `github.com/mainvec/mvp` repository, no go-get meta tags served). Publishing `mvpgo` as a fetchable module is tracked as a separate follow-up before the public release.
- Codegen templates for server/starter/test code still reference `github.com/mainvec/wo/` packages. Full rework is deferred to a follow-up issue; templates are marked with `// TODO: REWORK` notes.