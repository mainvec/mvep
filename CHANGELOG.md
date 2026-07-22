# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-07-22

### Changed
- **Monorepo consolidation.** Merged the `mvpgo` repository (Go + TypeScript runtimes) into this repo to form the MVEP (Mainvec Engineering Platform) monorepo. Layout is now `toolkit/` (generator) + `runtime/{go,ts}` (runtimes).
- **Generator renamed** `mvgen` → `toolkit` (module `github.com/mainvec/mvep/toolkit`, package `toolkit`).
- **Go runtime module path fixed/renamed** `github.com/mainvec/mvp/mvpgo` → `github.com/mainvec/mvep/runtime/go` (now resolvable: module path matches repo path). Go package `mvp` → `mvep`.
- **npm package renamed** `@mainvec/mvpjs` → `@mainvec/mvep`.
- **CLI renamed** `mvp` → `mvep` (`toolkit/mvepapi/cmd/mvep`).
- **Naming standardization** across the platform: output dir `mvpapi` → `mvepapi`, struct tag `mvp:` → `mvep:`, and "MVP" → "MVEP" in docs/spec labels.
- Generated Go source is now gofmt-normalized by the generator.

### Added
- `mvepspec/0.2` schema as the new canonical `$schema` URL, with `mvpspec/0.2` retained as a resolvable **alias** so existing spec files keep validating.
- Root `go.work` tying `toolkit` + `runtime/go` for local multi-module development.
- Merged CI: `go.yml` (toolkit + runtime/go), `js.yml` (runtime/ts), and `npm-publish.yml` (publishes `@mainvec/mvep` on `runtime/ts/v*` tags).
- `release.yml`: builds the `mvep` CLI for linux/darwin/windows (amd64/arm64) with the version injected via `-ldflags -X main.version`, verifies the stamp, and attaches binaries to a GitHub Release on `toolkit/v*` tags.
- Hybrid version resolution for the `mvep` CLI and for generated CLIs: linker-injected `main.version` → module version from `runtime/debug.ReadBuildInfo()` (`go install …@vX.Y.Z`) → static fallback (`"dev"` for `mvep`; the `<NAME>_VERSION` constant for generated CLIs). Replaces the previously hardcoded version.
- `NOMVEP` codegen protection marker (legacy `NOMVGEN`/`NOWOGEN` still honored).

### Fixed
- The CLI-main generation now honors the `NOMVEP`/`NOMVGEN`/`NOWOGEN` markers instead of unconditionally overwriting hand-customized entry points.
- Resolved the previously-unresolvable runtime module path (was `github.com/mainvec/mvp/mvpgo`).

### Versioning
- Hybrid scheme: the Go and TS runtimes share a version/tag (`runtime/go/vX.Y.Z`, `runtime/ts/vX.Y.Z`); the generator versions independently (`toolkit/vX.Y.Z`).

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