# MVEP Monorepo Consolidation + Naming Standardization

**GitHub Issue**: [#3](https://github.com/mainvec/mvep/issues/3)

> Status: **IN PROGRESS** — branch `chore/3-mvep-monorepo-consolidation`.

## Progress

- [ ] T1: Move mvpgo runtimes into the mvep repo (preserve history)
- [ ] T2: Rename generator `mvgen` → `toolkit`
- [ ] T3: Rename Go runtime module + package (`mvp` → `mvep`)
- [ ] T4: Rewire toolkit deps + add root `go.work`
- [ ] T5: Rename npm package `@mainvec/mvpjs` → `@mainvec/mvep`
- [ ] T6: Rename CLI `mvp` → `mvep`
- [ ] T7: Naming pass (struct tag, output dir, spec label)
- [ ] T8: Add `mvepspec/0.2` schema + keep `mvpspec/0.2` alias
- [ ] T9: Update generator templates
- [ ] T10: Regenerate self-code via gengen
- [ ] T11: Merge CI/CD + versioning
- [ ] T12: Docs, CHANGELOG, archive mvpgo repo

## Problem / Goal

Two tightly-coupled repos with a structural mismatch and overloaded naming:

| | Repo | Go module | npm |
|---|---|---|---|
| **mvep** | `github.com/mainvec/mvep` | `github.com/mainvec/mvep/mvgen` | — (emits JS/TS) |
| **mvpgo** | `github.com/mainvec/mvpgo` | `github.com/mainvec/mvp/mvpgo` ⚠️ | `@mainvec/mvpjs` |

Problems:
- **Broken runtime module path.** `mvpgo`'s repo is `github.com/mainvec/mvpgo` but its module declares `github.com/mainvec/mvp/mvpgo`. No repo exists at `github.com/mainvec/mvp`, so `go get github.com/mainvec/mvp/mvpgo` cannot resolve externally. This forces a `replace github.com/mainvec/mvp/mvpgo => ../../mvpgo/mvpgo` hack in mvep and is noted in the CHANGELOG as a known limitation. Every consumer of generated Go code inherits the problem.
- **Version drift / lockstep coupling.** mvep pins `mvpgo v0.4.1`; mvpgo is already at `v0.5.0`. The generator emits code that imports the runtime and hardcodes its import path in templates, so generator and runtime must stay compatible — but they release independently today.
- **Naming sprawl.** "mvp" plays seven distinct roles (platform, generator, protocol name, Go package, CLI, output-dir + struct tag, spec/schema), inviting confusion with "Minimal Viable Product".
- **Go version skew** (1.24 vs 1.23.2) and duplicated shared dep (`ugo v0.5.1`, `protobuf`).

Success: a single **MVEP (Mainvec Engineering Platform)** monorepo with a resolvable runtime module path, unified tooling/CI, hybrid versioning, and one cohesive "mvep" naming scheme.

## Goals

- Consolidate `mvpgo` (Go + TS runtimes) into the existing `github.com/mainvec/mvep` repo.
- Fix the runtime Go module path so `go get` resolves against the real repo (path == repo).
- Eliminate the overloaded "mvp" identifier in favor of "mvep" across code, CLI, tag, dirs, and spec label.
- Rename the generator component to `toolkit`.
- Preserve backward compatibility for existing spec files pinned to the old schema URL.
- Establish hybrid versioning and merged CI.

## Non-goals

- Migrating downstream consumers. They will need import-path + CLI + npm bumps; tracked as a separate follow-up issue. **The consumer footprint is larger than a first glance suggests** — the audit (2026-07-22) found:
  - **droy** — 7 Go modules (`droy-api`, `droy-cli`, `droy-dashboard` backend + `mvpapi/go`, `droy-desktop`, `droy-engine`, `droy-mobile`, `droy-syncin`), TS in `droy-dashboard` UI, and CLI scripts (`droy_update_apis.sh`, `generate_api.sh`).
  - **girafa** — 1 Go module + `generate_api.sh`.
  - **linkvec** — 8+ Go modules (`linkvec-core`, `linkvec-broker`, `linkvec-bench`, `linkvec-relay`, `linkvec-desktop`, `linkvec-mobile`, `linkvec-tui`, `linkvec-tui2`), TS in `linkvec-ui` + `linkvec-desktop` frontend, `generate_api.sh`, **and a `linkvec/go.work` that explicitly `use`s `../mvpgo/mvpgo`** (must be repointed to `../mvep/runtime/go`).
  - **mboxy** — imports the *older* module path `github.com/mainvec/mvpgo/mvp` at a **published** `v0.4.0` with **no** `replace`. Lower priority (won't break on local moves); only needs attention if/when it upgrades. Confirms the fix direction — the module path was originally repo-matching before it was renamed to the broken `github.com/mainvec/mvp/mvpgo`.
  - **agents/skills/mvp-codegen** — the shared codegen skill/docs reference the old Go path, `mvp.` selectors, `@mainvec/mvpjs`, and `mvp generate`. Must be updated so future codegen guidance emits correct names.
- **Rollout ordering (applies to the follow-up, not this plan):**
  1. Do **not** remove/move the local `mvpgo/` and `mvpjs/` directories until every consumer migrates — all `replace`/`file:` paths and `linkvec/go.work` resolve to `../mvpgo/mvpgo` / `../mvpgo/mvpjs` and would break simultaneously. Sequence: consolidate → migrate consumers → then retire local dirs.
  2. Go consumers can absorb the package-selector change (`mvp.` → `mvep.`) in hand-written files with an import alias `mvp "github.com/mainvec/mvep/runtime/go/mvep"`, keeping a one-line import swap per file; generated `*_package.go` files self-correct on regenerate.
- Changing the wire protocol, encoding, or transport behavior.
- Rewriting `ugo` or altering the shared dependency.
- Hard-cutting the hosted `mvpspec/0.2` schema URL (kept as an alias).

## Proposed Design

**Naming architecture** (kills all seven "mvp" roles):

- **MVEP** = Mainvec Engineering Platform = the monorepo umbrella.
- **toolkit/** = the generator (was `mvgen`), module `github.com/mainvec/mvep/toolkit`.
- **runtime/{go,ts}** = the protocol implementation.
  - Go: module `github.com/mainvec/mvep/runtime/go`, package `mvep` (import `.../runtime/go/mvep`).
  - TS: npm `@mainvec/mvep`.
- **`mvep`** = the CLI (unified binary; subcommands `gen`/`init`/`validate`).
- **"MVEP spec"**, struct tag `mvep:`, output-dir convention `mvepapi/`.

**Target layout**

```
mvep/                          github.com/mainvec/mvep
├── toolkit/                   module github.com/mainvec/mvep/toolkit          (own version)
├── runtime/
│   ├── go/                    module github.com/mainvec/mvep/runtime/go       (import .../runtime/go/mvep)
│   └── ts/                    npm @mainvec/mvep
├── go.work                    (toolkit + runtime/go — replaces the replace-hack)
└── .github/workflows/
```

**Key decisions**

- **Multi-module** monorepo (separate `go.mod` for `toolkit` and `runtime/go`), enabling independent tagging and smaller dependency graphs.
- **Hybrid versioning**: the Go and TS runtimes share a version number/tag; `toolkit` versions independently.
  - Tags: `toolkit/vX.Y.Z` (independent), `runtime/go/vX.Y.Z` + `runtime/ts/vX.Y.Z` (shared number).
- **Struct tag rename is safe/cosmetic.** Verified nothing reads the `mvp:` tag — `ugo/oencoding` is a registry interface only; JSON encoding delegates to `encoding/json` (`json:` tag), and protobuf uses generated `.pb.go` descriptors. The wire contract lives in `json`/`cbor` tags + protobuf field numbers, not this key. No `ugo` change required.
- **Schema alias (additive).** The generator maps schema URLs → embedded files via `supportedSchemaResources` in `toolkit/mvgen.go` (already inconsistent: `0.1` uses `mvepspec`, `0.2` uses `mvpspec`). Add an `mvepspec/0.2` entry (copy the schema, update its `$id`) and keep the `mvpspec/0.2` entry so existing specs keep validating.
- **Local dev** uses a root `go.work` tying `toolkit` + `runtime/go`, removing the `replace` directive.

**Rejected alternatives**

- _Single root Go module_: would couple all releases and force one Go version; rejected in favor of multi-module.
- _Keeping `github.com/mainvec/mvp/mvpgo`_: leaves the path unresolvable; rejected.
- _Hard-cut the schema URL_: breaks external spec files; rejected in favor of an alias.
- _Keeping the tag key as `mvp:`_: unnecessary once verified nothing reads it.

## Affected Modules

- `mvep/mvgen` → `mvep/toolkit` (generator: templates, gengen, CLI, schema resources, docs).
- `mvpgo/mvpgo` → `mvep/runtime/go` (Go runtime, package rename).
- `mvpgo/mvpjs` → `mvep/runtime/ts` (TS runtime, npm rename).
- `.github/workflows/` in both repos (merged).
- Root of `mvep` repo (`go.work`, README, CHANGELOG).
- **Out of scope (follow-up):** `droy`, `girafa`, and any generated consumer repos.

## Tasks

### T1: Move mvpgo runtimes into the mvep repo (preserve history)

**Outcome**: `mvpgo/mvpgo` relocated to `mvep/runtime/go` and `mvpgo/mvpjs` to `mvep/runtime/ts` with git history preserved; mvpgo's CI workflows moved under `mvep/.github/workflows/`.

**Verification**: `git log --follow` shows history on moved files; both trees present at the new paths; no build attempted yet.

**Notes**: Prefer `git subtree add` (simple, keeps history under a prefix) over a `git filter-repo` merge.

### T2: Rename generator `mvgen` → `toolkit`

**Outcome**: `mvep/mvgen/` renamed to `mvep/toolkit/`; `go.mod` module → `github.com/mainvec/mvep/toolkit`; internal imports and `gengen` output paths updated.

**Verification**: `go build ./...` in `toolkit/` succeeds; no residual `mainvec/mvep/mvgen` import remains.

**Notes**: gengen entry point and `mvgen_gen_plain_main.go` output dir references must be updated.

### T3: Rename Go runtime module + package (`mvp` → `mvep`)

**Outcome**: `runtime/go/go.mod` module → `github.com/mainvec/mvep/runtime/go`; `package mvp` → `package mvep` across all runtime files; self-referential imports point to `.../runtime/go/mvep`; Go version aligned to 1.24.

**Verification**: `go build ./...` and `go test ./...` pass in `runtime/go`; `grep -r "package mvp\b"` returns nothing.

**Notes**: Update `client/`, `server/`, `util/` sub-packages that import the runtime package.

### T4: Rewire toolkit deps + add root `go.work`

**Outcome**: `toolkit/go.mod` requires `github.com/mainvec/mvep/runtime/go` (old `github.com/mainvec/mvp/mvpgo` require + `replace` removed); root `go.work` ties `toolkit` + `runtime/go`.

**Verification**: `go build ./...` works with no `replace` directive; `go work sync` clean.

**Notes**: Depends on T2 and T3.

### T5: Rename npm package `@mainvec/mvpjs` → `@mainvec/mvep`

**Outcome**: `runtime/ts/package.json` `name` → `@mainvec/mvep`; `repository`, `homepage`, `bugs` URLs point at the mvep repo; version aligned to the shared runtime number.

**Verification**: `npm run build` + `npm test` pass; `npm pack --dry-run` shows correct name/metadata.

**Notes**: npm name is independent of repo location; JS consumers migrate separately (follow-up).

### T6: Rename CLI `mvp` → `mvep`

**Outcome**: `mvpapi/cmd/mvp` → `mvepapi/cmd/mvep`; `Usage`, flag help, and command references say `mvep`.

**Verification**: `go build -o mvep ./mvepapi/cmd/mvep` produces the binary; `mvep --help` shows `mvep` usage.

**Notes**: Depends on T2. Coordinate with T7's `mvpapi` → `mvepapi` rename.

### T7: Naming pass (struct tag, output dir, spec label)

**Outcome**: struct tag `mvp:` → `mvep:` (cosmetic — verified no reader); output-dir convention `mvpapi/` → `mvepapi/`; "MVP Spec" → "MVEP spec" in docs/help text.

**Verification**: `grep -rn 'mvp:"'` returns only regenerated `mvep:` tags; sample generation emits `mvepapi/`.

**Notes**: Tag rename is safe — see Decision Log entry on the `ugo` check.

### T8: Add `mvepspec/0.2` schema + keep `mvpspec/0.2` alias

**Outcome**: `resources/mvpspec/0.2/schema/2026-01-15.json` copied to `resources/mvepspec/0.2/schema/2026-01-15.json` with `$id` set to `.../mvepspec/0.2/...`; a `supportedSchemaResources` entry added for `mvepspec/0.2`; the existing `mvpspec/0.2` entry retained as an alias.

**Verification**: validating a spec pinned to `mvepspec/0.2` passes; validating one pinned to the old `mvpspec/0.2` still passes.

**Notes**: Mechanism confirmed in `toolkit/mvgen.go` `supportedSchemaResources`; `0.1` already uses `mvepspec`.

### T9: Update generator templates

**Outcome**: Go template import → `github.com/mainvec/mvep/runtime/go/mvep`; Go struct template tag `mvp:` → `mvep:`; JS templates import → `@mainvec/mvep`; emitted `$schema` → `mvepspec/0.2`.

**Verification**: rendering templates for a sample spec yields the new imports/tags/schema.

**Notes**: Files: `resources/codegen_templates/go/go_package_code.txt`, `resources/codegen_templates/go/go_structs_code.txt`, `resources/codegen_templates/js/*`.

### T10: Regenerate self-code via gengen

**Outcome**: `gengen` re-run from `mvgen_plain.jsonc` so `toolkit/mvepapi` reflects the new imports, package, tag, and CLI; diff committed.

**Verification**: a second `gengen` run yields zero diff (idempotent); `go build ./...` passes.

**Notes**: Depends on T6, T7, T9.

### T11: Merge CI/CD + versioning

**Outcome**: merged workflows — Go build/vet/test for `toolkit` + `runtime/go`, vitest for `runtime/ts`; npm-publish triggered by `runtime/ts/v*` tags; tag conventions documented.

**Verification**: CI passes on a branch; a dry-run publish job succeeds on a PR.

**Notes**: Hybrid tags — `toolkit/vX.Y.Z` independent; `runtime/go` + `runtime/ts` share a number.

### T12: Docs, CHANGELOG, archive mvpgo repo

**Outcome**: READMEs, `CHANGELOG.md`, `MVP_SKILL.md`/`AGENT.md`/`MVP_ROADMAP.md` updated to MVEP naming; `mvpgo` repo archived with a redirect pointer.

**Verification**: no residual `mvgen`/`mvpgo`/`mvpjs` naming in docs except the intentional `mvpspec` alias note.

**Notes**: Do last, after code and CI are green.

## Risks and Compatibility

- **Breaking for downstream Go consumers.** Import path changes from `github.com/mainvec/mvp/mvpgo/mvp` to `github.com/mainvec/mvep/runtime/go/mvep`. Consumers (`droy`, `girafa`, generated repos) will not build until bumped — handled as a follow-up. Optional mitigation: a thin compatibility shim republishing the old path.
- **Breaking for JS consumers.** `@mainvec/mvpjs` → `@mainvec/mvep`; consumers update their dependency + imports.
- **CLI name change.** Scripts invoking `mvp` must switch to `mvep`.
- **Schema URL.** Mitigated — `mvpspec/0.2` kept as an alias; existing spec files keep validating.
- **Struct tag.** Low risk — verified cosmetic; no serializer reads the key.
- **Rollback.** Until the `mvpgo` repo is archived, the previous state remains available; the consolidation lands as a reviewable PR.

## Verification

1. `go build ./...`, `go test ./...`, `go vet ./...` pass in `toolkit` and `runtime/go`.
2. `npm run build` + `npm test` pass in `runtime/ts`.
3. `mvep generate` on a sample spec produces Go importing `github.com/mainvec/mvep/runtime/go/mvep`, struct tag `mvep:`, JS importing `@mainvec/mvep`, and `$schema` using `mvepspec/0.2`.
4. A spec pinned to the old `mvpspec/0.2` schema URL still validates (alias works).
5. `go get github.com/mainvec/mvep/runtime/go` resolves (path == repo); no `replace` directive needed.
6. A second `gengen` run yields a zero diff (idempotent).
7. Grep shows no residual `mvgen`, `mvpgo`, `mvpjs`, `github.com/mainvec/mvp/mvpgo`, `package mvp`, `mvp:"`, or `cmd/mvp` — only the intentional `mvpspec` alias remains.

## Rollout

- Land as a single reviewable PR against `mvep` `main` (issue-numbered branch per feature-workflow).
- After merge and green CI, cut the first monorepo tags: `runtime/go/vX.Y.Z` + `runtime/ts/vX.Y.Z` (shared) and `toolkit/vX.Y.Z`.
- Publish `@mainvec/mvep` from the `runtime/ts/v*` tag.
- Archive the `mvpgo` repo with a README pointer to `mainvec/mvep`.
- Open the downstream-migration follow-up issue (droy, girafa, generated repos).

## Decision Log

- **2026-07-22** — Consolidate into the existing `mvep` repo (not a new repo, not into `mvpgo`). Rationale: matches the MVEP naming and keeps the generator's history as the monorepo root.
- **2026-07-22** — Multi-module layout `toolkit/` + `runtime/{go,ts}`. Rationale: independent tagging, smaller dep graphs, closest to current structure.
- **2026-07-22** — Runtime Go module → `github.com/mainvec/mvep/runtime/go`. Rationale: makes the module path resolvable (fixes the broken `github.com/mainvec/mvp/mvpgo`).
- **2026-07-22** — Generator renamed to `toolkit` (over `toolchain`). Rationale: lower churn; README already used "MVP Toolkit".
- **2026-07-22** — CLI unified as `mvep` with subcommands. Rationale: removes the overloaded `mvp` command name.
- **2026-07-22** — npm renamed to `@mainvec/mvep`. Rationale: consistent MVEP branding.
- **2026-07-22** — Hybrid versioning: runtime Go+TS share a tag/number; `toolkit` independent. Rationale: guarantees runtime cross-language compatibility while letting the generator evolve on its own cadence.
- **2026-07-22** — Schema handling: add `mvepspec/0.2`, keep `mvpspec/0.2` as an alias. Rationale: existing spec files pin the old URL; additive mapping avoids breakage (0.1 already used `mvepspec`).
- **2026-07-22** — Struct tag `mvp:` → `mvep:` confirmed safe. Rationale: workspace-wide search found no reader of the `mvp` tag; `ugo/oencoding` is a registry interface only, JSON uses `encoding/json` (`json:` tag), protobuf uses generated `.pb.go`. No wire-format impact, no `ugo` change.
