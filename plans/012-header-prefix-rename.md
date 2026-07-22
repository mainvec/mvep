# Plan 012 — Rename HTTP header prefix `x-mvp-` → `x-mvep-`

GitHub issue: mainvec/mvep#12

## Problem / Goal
The MVEP HTTP transport prefixes custom command headers with `x-mvp-`, a leftover
from the old "MVP" branding. Rename it to `x-mvep-` for consistency with the rest of
the platform.

## Goals
- Rename the custom-header prefix to `x-mvep-` in the Go and TS runtimes.
- Keep client and server in lock-step (single coordinated flip; deployments are minimal).
- Update runtime tests and docs; bump the runtime to v0.7.0.

## Non-goals
- The control headers `x-mainvec-cmd` / `x-mainvec-error` keep their `x-mainvec-` prefix.
- No backward/dual-accept compatibility: a coordinated breaking flip was chosen.
- Downstream consumers that hardcode `x-mvp-auth` (droy, linkvec, mboxy) are NOT changed
  here — they still run the old `mvpgo` runtime (`x-mvp-`); flipping their headers now
  would break them. Those edits ride with each consumer's runtime migration. girafa has
  no hardcoded prefix and picks up `x-mvep-` on a runtime bump.
- The `mvpgo` repo (being archived) is left unchanged.

## Proposed Design
- Go: change `const HeaderPrefix = "x-mvep-"` in `runtime/go/mvep/envelope.go`. Write and
  read paths in `http_transport.go` already use the constant symmetrically.
- TS: change `export const HEADER_PREFIX = 'x-mvep-'` in `runtime/ts/src/envelope.ts`;
  rebuild `dist/`.
- Update comments ("custom MVP headers" → "custom MVEP headers").

## Affected Modules
- `runtime/go/mvep` (envelope.go, http_transport.go usage, new test).
- `runtime/ts` (envelope.ts, tests, dist, package.json version).
- Docs: `runtime/go/README.md`, `toolkit/MVEP_SKILL.md`; agents-repo `mvep-codegen` skill (separate commit).

## Risks and Compatibility
- Breaking wire change: a peer on `x-mvep-` cannot exchange custom headers (auth,
  request-id) with a peer on `x-mvp-`. Mitigated by minimal deployments + coordinated bump.
- Consumers must upgrade client and server together to runtime ≥ v0.7.0.

## Verification
- Go: new httptest round-trip test asserts outgoing header key is `x-mvep-auth` and a
  response `x-mvep-*` header is surfaced stripped. `go test ./...`, `go vet`, gofmt.
- TS: updated unit tests expect `x-mvep-`; `npm run build` + `npm test`.
- grep: no residual `x-mvp-` in the mvep repo except intentional history/CHANGELOG.

## Rollout
- Merge PR; tag `runtime/go/v0.7.0`. Hold `runtime/ts/v0.7.0` until `NPM_TOKEN` is set.
- Downstream: bundle the `x-mvep-` header edits into each consumer's runtime migration.

## Decision Log
- Coordinated breaking flip (Option A) over dual-accept — deployments are minimal (user).
- Version: runtime v0.7.0 (breaking, pre-1.0 minor bump) (user).

## Progress
- [x] T1 — Go: failing httptest round-trip test for `x-mvep-` prefix
- [x] T2 — Go: flip `HeaderPrefix` constant + comment
- [x] T3 — TS: update unit tests to `x-mvep-`
- [x] T4 — TS: flip `HEADER_PREFIX` + comment; rebuild dist
- [x] T5 — Bump `runtime/ts` package.json to 0.7.0 + CHANGELOG
- [x] T6 — Update docs (runtime README, MVEP_SKILL)

## Tasks

### T1 — Go failing test
Outcome: `TransportCmdReq` round-trip test asserts write emits `x-mvep-<k>` and read strips it.
Verification: test fails against the old `x-mvp-` constant, passes after T2.
Notes: uses `httptest` (localhost).

### T2 — Go flip constant
Outcome: `HeaderPrefix = "x-mvep-"`.
Verification: `go test ./...`, `go vet ./...`, gofmt clean.

### T3 — TS test update
Outcome: `envelope.test.ts`, `client.test.ts`, `http-transport.test.ts` expect `x-mvep-`.
Verification: tests fail against old const, pass after T4.

### T4 — TS flip constant + dist
Outcome: `HEADER_PREFIX = 'x-mvep-'`; `dist/` rebuilt.
Verification: `npm run build` + `npm test`.

### T5 — Version + CHANGELOG
Outcome: `runtime/ts/package.json` → 0.7.0; CHANGELOG notes the breaking header rename.
Verification: version reflected; changelog entry present.

### T6 — Docs
Outcome: `runtime/go/README.md` curl examples + `toolkit/MVEP_SKILL.md` use `x-mvep-`.
Verification: grep shows no stray `x-mvp-` (except history).
