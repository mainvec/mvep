# Plan 014 — Publish `@mainvec/mvep@0.7.0` to npm

- **Issue**: #14 — chore: publish `@mainvec/mvep@0.7.0` to npm
- **Branch**: `chore/14-publish-mvep-ts-070`

## Problem / Goal

Downstream consumers (droy, and any TS consumer) currently depend on the TS
runtime via a `file:` relative path because `@mainvec/mvep` was never published
to npm. `runtime/ts/package.json` is already at `0.7.0`, `runtime/go/v0.7.0` is
tagged, and the `npm-publish.yml` workflow is wired to publish on
`runtime/ts/v*` tag pushes. The only blocker is the missing `NPM_TOKEN` repo
secret (issue #5 held `runtime/ts/v0.6.0` for exactly this reason).

Goal: ship the first npm release so consumers can `npm install @mainvec/mvep@0.7.0`.

## Goals

- Configure the `NPM_TOKEN` repository secret so the publish workflow can authenticate.
- Tag and push `runtime/ts/v0.7.0` to trigger `.github/workflows/npm-publish.yml`.
- Confirm `@mainvec/mvep@0.7.0` is published and installable from a clean project.
- Update `docs/consumer-migration.md` to remove the "not yet on npm" note.

## Non-goals

- Changing runtime code, package shape, or the workflow itself.
- Version bump beyond the already-set `0.7.0` in `package.json`.
- Migrating any consumer app (that is per-app migration work, tracked separately).

## Proposed Design

This is a release/ops chore, not a code change. The pipeline already exists and
is correct:

1. **Secret** — add `NPM_TOKEN` as a GitHub Actions secret on `mainvec/mvep`.
   Use a **granular automation** npm token scoped to `@mainvec` (read+write),
   per `runtime/ts/NPM_PUBLISH.md`. Stored via `gh secret set NPM_TOKEN`.
2. **Tag** — `runtime/ts/v0.7.0` annotated tag on the current `main` HEAD
   (`a362e7f`, which already carries `runtime/go/v0.7.0`). No commit needed;
   `package.json` version is already `0.7.0`.
3. **Publish** — pushing the tag triggers `npm-publish.yml`: `npm ci` →
   `npm run build` → `npm test` → `npm publish --dry-run` →
   `npm publish --access public` (using `NPM_TOKEN` as `NODE_AUTH_TOKEN`).
4. **Docs** — after publish, remove the "not yet on npm" / `file:` path note in
   `docs/consumer-migration.md` and point consumers at `@mainvec/mvep@0.7.0`.

## Affected Modules

- `mvep/.github/workflows/npm-publish.yml` — no change; just consumes the new secret.
- `mvep/docs/consumer-migration.md` — doc update (T2).
- `mvep/plans/014-publish-mvep-ts-070.md` — this plan.

## Risks and Compatibility

- **No npm token yet** — the publish job will fail with `ENEEDAUTH` if the tag
  is pushed before the secret is set. Mitigation: set the secret **first** (T1).
- **403 on `npm view`** is expected pre-publish for an uncreated scoped package;
  it is not a token problem. Verify post-publish instead.
- **2FA** — if the npm account requires 2FA at publish, the token must be an
  automation-type (granular automation qualifies). Covered by the token choice.
- **Re-publish guard** — `package.json` is `0.7.0`; if a prior partial publish
  ever happened, `npm publish` would 409. `npm view` after the run disambiguates.
- **No code/ABI change** — purely a release; consumers already on `file:` paths
  can switch to the registry version without behavior change.

## Verification

- `gh run list --workflow=npm-publish.yml` shows a green run for the tag push.
- `npm view @mainvec/mvep version` → `0.7.0` (run from a machine without the
  forbidden-policy quirk, or via `npm view` with a token).
- `npm install @mainvec/mvep@0.7.0` succeeds in a throwaway project and
  `import { ... } from "@mainvec/mvep"` resolves.

## Rollout

1. Set secret (T1).
2. Push tag (T1) → workflow publishes.
3. Confirm publish (T1 verification).
4. Open PR with the docs update (T2), merge to `main`.

## Decision Log

- Ship `0.7.0` directly (not `0.6.0` then `0.7.0`): `package.json` is already at
  `0.7.0` and `runtime/go/v0.7.0` is tagged; no value in a never-published `0.6.0`.
- No workflow edits needed — `npm-publish.yml` already targets `runtime/ts/v*`
  and uses `NPM_TOKEN` as `NODE_AUTH_TOKEN`.

## Progress

- [x] T1 — Set `NPM_TOKEN` secret and push `runtime/ts/v0.7.0` tag; confirm publish.
- [x] T2 — Update `docs/consumer-migration.md` to point at the published package.

## Tasks

### T1 — Set `NPM_TOKEN` secret and push `runtime/ts/v0.7.0` tag; confirm publish

- **Outcome**: `@mainvec/mvep@0.7.0` is installable from npm.
- **Verification**: green `npm-publish.yml` run; `npm view @mainvec/mvep version`
  returns `0.7.0`; `npm install @mainvec/mvep@0.7.0` works in a clean project.
- **Notes**:
  - Operator (user) must generate the npm granular automation token and paste it
    into `gh secret set NPM_TOKEN --repo mainvec/mvep` — the model must not handle
    the token value. See `runtime/ts/NPM_PUBLISH.md` §Prerequisites.
  - Tag on current `main` HEAD (`a362e7f`); `package.json` already `0.7.0`.
  - Command: `git tag -a runtime/ts/v0.7.0 -m "@mainvec/mvep 0.7.0" && git push origin runtime/ts/v0.7.0`.
  - **First push failed** (403): the initial granular token lacked 2FA-bypass;
    regenerated the token with bypass enabled (or used a classic automation
    token), re-set the secret, deleted + re-pushed the tag, and the second run
    (`29999171158`) succeeded.
  - **Verified published**: `https://registry.npmjs.org/@mainvec%2fmvep/latest`
    → `"version":"0.7.0"`, `dist-tags.latest = 0.7.0`. (Local `npm view` returns
    403 due to a local npm config/policy quirk, not a registry problem.)

### T2 — Update `docs/consumer-migration.md` to point at the published package

- **Outcome**: consumers see `@mainvec/mvep@0.7.0` as the install target, not a
  `file:` path.
- **Verification**: the "not yet on npm" row/paragraph in the Target versions
  table and TS section is removed or rewritten.
- **Notes**: update the `| TS runtime | …` row in the Target versions table and
  any TS-section reference to a `file:` path.