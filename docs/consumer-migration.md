# Consumer Runtime Migration Guide v2 — `mvpgo` → `mvep`

> **Supersedes** `consumer-migration.md`. This v2 incorporates lessons learned
> from the linkvec migration (PR #60, #62) and adds the gaps discovered in
> practice. The original guide is kept unchanged for reference.

How to upgrade an app that consumes the old `mvpgo` runtime (and the `mvp` CLI /
`@mainvec/mvpjs`) to the consolidated **MVEP** platform: the `mvep` runtime, the
`mvep` CLI (from `toolkit`), and `@mainvec/mvep`.

This procedure is the one used to migrate **girafa** (the reference consumer) and
**linkvec** (the first multi-module consumer). Use it for **droy**, and any other
consumer. Per-app inventories are in the appendices.

> ⚠️ **Breaking wire change.** Runtime `v0.7.0` renamed the HTTP custom-header
> prefix `x-mvp-` → `x-mvep-`. A peer on the new runtime cannot exchange custom
> headers (auth, request-id) with a peer still on `x-mvp-`. **Upgrade a service and
> all of its clients together**, and flip any hardcoded `x-mvp-` in the same change.

---

## Target versions

| Component | Install / pin |
|-----------|---------------|
| CLI (`mvep`) | `go install github.com/mainvec/mvep/toolkit/mvepapi/cmd/mvep@v0.5.2` |
| Go runtime | `github.com/mainvec/mvep/runtime/go v0.7.0` |
| TS runtime | `@mainvec/mvep@0.7.0` (published on npm) |

## What changed (mapping)

| Concern | Old | New |
|---------|-----|-----|
| Go module | `github.com/mainvec/mvp/mvpgo` | `github.com/mainvec/mvep/runtime/go` |
| Go import (runtime) | `github.com/mainvec/mvp/mvpgo/mvp` | `github.com/mainvec/mvep/runtime/go/mvep` |
| Server subpkg | `…/mvpgo/mvp/server` | `…/runtime/go/mvep/server` |
| Client subpkg | `…/mvpgo/mvp/client` | `…/runtime/go/mvep/client` |
| Go package selector | `mvp.` | `mvep.` |
| Go import alias | `mvpclient "…/mvpgo/mvp/client"` | `mvepclient "…/runtime/go/mvep/client"` |
| Go version | `v0.4.1` + local `replace` | `v0.7.0` (published — drop the `replace`) |
| npm package | `@mainvec/mvpjs` | `@mainvec/mvep` |
| CLI | `mvp` | `mvep` (subcommands `generate`/`gen`/`init`/`validate` unchanged) |
| Struct tag | `mvp:"…"` | `mvep:"…"` (rewritten automatically on regen) |
| Protection marker | `// NOMVGEN` / `// NOWOGEN` | `// NOMVEP` (legacy markers still honored) |
| HTTP header prefix | `x-mvp-` | `x-mvep-` |
| Output dir convention | `mvpapi/` | `mvepapi/` (optional rename; existing dirs keep working) |

The control headers `x-mainvec-cmd` / `x-mainvec-error` are **unchanged**.

---

## Prerequisites

1. Install the new CLI and confirm it runs:
   ```bash
   go install github.com/mainvec/mvep/toolkit/mvepapi/cmd/mvep@v0.5.2
   mvep -v      # → v0.5.2
   ```
2. Ensure a clean (or intentionally-staged) working tree. If the app has
   **uncommitted work in generated dirs**, read "Protect hand-edited files" first —
   regeneration can overwrite generated files that lack a protection marker.
3. The `mvep` repo is public and go-gettable, so **no `GOPRIVATE` and no `replace`**
   is needed for the Go runtime.

---

## Go migration (per module)

Do this for **every** `go.mod` that references the old runtime (see appendices for
the full list per app).

> **Step ordering matters.** Steps 1–4 (edit go.mod, repoint imports, regenerate)
> must complete before step 5 (`go mod tidy`). If any source file still imports
> `mvpgo` when you tidy, the command will fail because the module is no longer in
> `go.mod`.

1. **Update the generate script(s)** — replace the CLI name only; flags are
   compatible (`-in`/`--in`, `-lang`, `-outdir`, `-format=plain`, `gen`/`generate`):
   ```diff
   - mvp generate -in "$SPEC" -lang go -outdir ./go -format=plain
   + mvep generate -in "$SPEC" -lang go -outdir ./go -format=plain
   ```

2. **Update `go.mod`** — drop the old require + local `replace`, add the published
   runtime:
   ```diff
   require (
   -   github.com/mainvec/mvp/mvpgo v0.4.1
   +   github.com/mainvec/mvep/runtime/go v0.7.0
   )
   - replace github.com/mainvec/mvp/mvpgo => ../../mvpgo/mvpgo
   ```
   Keep unrelated `replace`s (e.g. `ugo`).

   **Indirect requires:** For modules that list `mvpgo` only as an *indirect*
   dependency (e.g. consumer modules that depend on the app's core module), replace
   the `// indirect` line manually — `go mod tidy` alone won't remove it if the
   generated code still imports `mvpgo`:
   ```diff
   -   github.com/mainvec/mvp/mvpgo v0.4.1 // indirect
   +   github.com/mainvec/mvep/runtime/go v0.7.0 // indirect
   ```

3. **Hand-written runtime imports** (server/client/impl code you wrote): update the
   import path and selector:
   ```diff
   - import "github.com/mainvec/mvp/mvpgo/mvp"
   - import "github.com/mainvec/mvp/mvpgo/mvp/server"
   + import "github.com/mainvec/mvep/runtime/go/mvep"
   + import "github.com/mainvec/mvep/runtime/go/mvep/server"
   ```
   Then `mvp.` → `mvep.` in those files. (To minimize churn you may alias
   `mvep "github.com/mainvec/mvep/runtime/go/mvep"` — but prefer the clean rename.)

   > **Import aliases** *(v2 addition)*: If the code aliases the import (e.g.
   `mvpclient "github.com/mainvec/mvp/mvpgo/mvp/client"`), rename the alias too
   (e.g. `mvepclient`) and update every `mvpclient.` reference in that file —
   `mvpclient.NewClient`, `mvpclient.ClientConfig`, etc. Note that struct *field
   names* that happen to look like package selectors (e.g. `mvpClient`) are **not**
   package references — they don't need renaming, though you may rename them for
   consistency.

4. **Regenerate** so the generated code picks up the new import
   (`…/runtime/go/mvep`), `mvep.` selectors, and `mvep:` tags:
   ```bash
   cd mvpapi && bash generate_api.sh && cd ..
   ```
   > If a spec's generated dir has **uncommitted edits in files lacking a marker**,
   > see "Protect hand-edited files". For girafa we did a **surgical** migration
   > (regen to a temp dir, then copy only the clean pure-generated files) to avoid
   > clobbering dirty hand-edited entry points. Prefer that when the tree is dirty.

   > **Ordering** *(v2 addition)*: Hand-written imports (step 3) and regeneration
   > (step 4) must happen **before** `go mod tidy` (step 5). If generated or
   > hand-written files still import `mvpgo`, `go mod tidy` will fail because the
   > module is no longer in `go.mod`.

5. **Tidy** (after steps 3 and 4, when no source file imports `mvpgo` anymore):
   ```bash
   go mod tidy      # pulls github.com/mainvec/mvep/runtime/go@v0.7.0
   ```

6. **Flip hardcoded header prefixes** `x-mvp-` → `x-mvep-` (see appendices for
   locations). This is mandatory and must land with the runtime bump.

7. **Update project docs** *(v2 addition)*: Search `AGENTS.md`, `README.md`,
   `CONTRIBUTING.md`, `RELEASE.md`, and any other docs for `mvpgo`, `mvp generate`,
   `mvp/client`, `mvp/server`, or `@mainvec/mvpjs` references. These aren't
   breaking (docs don't compile) but cause confusion if left stale. Also update
   references to the dependency in any "Technology Stack" or "Development
   Environment" sections.

8. **Check build tooling** *(v2 addition)*: Search `.goreleaser.yaml`, install
   scripts (`install/*.sh`), test scripts (`tests/**/*.sh`), CI workflows
   (`.github/workflows/`), and `.gitignore` for references to `mvp` CLI invocations
   or `mvpapi/` paths. If you also rename the directory (see below), these all need
   updating. Even without the rename, any `mvp generate` calls in scripts must
   become `mvep generate`.

9. **Verify**:
   ```bash
   go build ./... && go vet ./... && go test ./...
   ```

### `go.work` (multi-module repos)

If the repo has a root `go.work` that lists the sibling `mvpgo` checkout, remove it:
```diff
use (
    ...
-   ../mvpgo/mvpgo
)
```
With the published `v0.7.0` require in each module, nothing else is needed. Only add
`use ../mvep/runtime/go` if you intend to develop the runtime locally alongside the app.

---

## TypeScript migration

1. **Dependency** in `package.json`:
   ```diff
   - "@mainvec/mvpjs": "file:../../mvpgo/mvpjs"
   + "@mainvec/mvep": "^0.7.0"
   ```
   `@mainvec/mvep` is published on npm, so prefer the registry version. If you need
   to develop the runtime locally alongside the app, a `file:` path to
   `mvep/runtime/ts` still works as a temporary override:
   `"@mainvec/mvep": "file:../../mvep/runtime/ts"`.
   (The TS package builds `dist/` on install/publish.)

2. **Imports**:
   ```diff
   - import { newClient, staticAuthHeaderInterceptor } from '@mainvec/mvpjs';
   + import { newClient, staticAuthHeaderInterceptor } from '@mainvec/mvep';
   ```

3. **Build aliases** (vite/tsconfig) that point at `@mainvec/mvpjs`:
   ```diff
   - '@mainvec/mvpjs': fileURLToPath(new URL('./node_modules/@mainvec/mvpjs/dist/index.js', import.meta.url))
   + '@mainvec/mvep': fileURLToPath(new URL('./node_modules/@mainvec/mvep/dist/index.js', import.meta.url))
   ```

4. **Flip hardcoded `x-mvp-`** (e.g. dev-proxy headers) → `x-mvep-`.

5. Reinstall + rebuild + test:
   ```bash
   npm install && npm run build && npm test
   ```
   > **Lock files** *(v2 addition)*: `npm install` regenerates `package-lock.json`
   > with the new dependency — don't hand-edit the lock file. Committed `dist/`
   > bundles that embed `x-mvp-`/`@mainvec/mvpjs` are build artifacts — they refresh
   > on rebuild; don't hand-edit them either.

---

## Protect hand-edited generated files

The generator **skips** any file whose **first line** is `// NOMVEP` (or legacy
`// NOMVGEN` / `// NOWOGEN`). Files it will otherwise overwrite: `*_package.go`,
`*.plain.go`, `*_commands.go`, `*_main_cmd.go`, `*_version.go`, and the `api/` dir.

- If you hand-edited any of those, ensure the marker is on line 1 **before** you
  regenerate, or your edits will be lost.
- If the working tree is dirty in generated dirs, prefer the **surgical** approach:
  regenerate into a temp dir, diff, and copy over only the clean pure-generated
  files — leaving your dirty/customized files untouched.

> **Editor language server** *(v2 addition)*: After `git mv`-ing the `mvpapi/`
> directory (if you rename), the Go language server (gopls) will show stale
> "could not import … no required module provides package" errors for the old
> paths. Restart gopls (VS Code: "Go: Restart Language Server") or close and
> reopen the affected files to clear the cache. The build itself is unaffected.

---

## Verification checklist (per app)

- [ ] `mvep -v` → `v0.5.2`.
- [ ] Every `go.mod` requires `github.com/mainvec/mvep/runtime/go v0.7.0`; no
      `mvpgo` `require`/`replace` remains (including `// indirect` lines).
- [ ] `go.work` no longer lists `../mvpgo/mvpgo`.
- [ ] Generate scripts call `mvep`, not `mvp`.
- [ ] Regenerated code imports `…/runtime/go/mvep`, uses `mvep.` and `mvep:` tags.
- [ ] No `github.com/mainvec/mvp/mvpgo` or `mvp.` selectors remain in source.
- [ ] Import aliases (e.g. `mvpclient`) renamed to `mvep` equivalents.
- [ ] `@mainvec/mvpjs` → `@mainvec/mvep` in every `package.json`, import, and alias.
- [ ] No hardcoded `x-mvp-` remains in source/config (dist bundles regenerate).
- [ ] `go build ./... && go vet ./... && go test ./...` pass in each module.
- [ ] TS `npm run build && npm test` pass.
- [ ] Project docs (`AGENTS.md`, `README.md`, `CONTRIBUTING.md`) updated — no stale
      `mvp`/`mvpgo` references.
- [ ] Build tooling (`.goreleaser.yaml`, install scripts, CI, `.gitignore`) checked
      for stale `mvp`/`mvpapi` references.
- [ ] Server(s) and all clients deployed on runtime ≥ `v0.7.0` together.

## Rollback

The change is a coordinated version bump. To roll back, revert the `go.mod` /
`package.json` edits and the header-prefix flips together (restore `x-mvp-` and the
old runtime pin). Do not split client/server across the prefix boundary.

---

## Appendix A — droy (no `go.work`; 9 Go modules with `mvpgo`)

**`go.mod` to update** (drop `mvpgo` require+replace, add `runtime/go v0.7.0`):
`droy-api`, `droy-cli`, `droy-cli/tools/droy-migrate`, `droy-dashboard/backend`,
`droy-dashboard/mvpapi/go`, `droy-desktop`, `droy-engine`, `droy-mobile/gomobile`,
`droy-syncin`. (Note: several also `replace github.com/mainvec/mvpgo => ../../mvpgo`
— drop that too.)

> **Indirect requires** *(see Go migration step 2)*: `droy-cli`, `droy-desktop`,
> `droy-engine`, and `droy-mobile/gomobile` list `mvpgo` as `// indirect`
> (`github.com/mainvec/mvp/mvpgo v0.4.1 // indirect`). `go mod tidy` won't rewrite
> these while generated code still imports `mvpgo`, so replace the line manually
> with `github.com/mainvec/mvep/runtime/go v0.7.0 // indirect`.

**Generate scripts** (`mvp` → `mvep`):
`droy-dashboard/mvpapi/generate_api.sh` (`mvp gen …`),
`droy-api/droy_update_apis.sh` (`mvp generate …`).

**Hand-written Go runtime imports** to repoint:
`droy-api/mvpapi/go/api/droy_package.go`,
`droy-dashboard/backend/backend.go` (+ `/mvp/server`),
`droy-dashboard/backend/dash_client.go` (+ `/mvp/client`),
`droy-dashboard/mvpapi/go/api/dashboard_package.go`.

> **Import alias** *(see Go migration step 3)*: Unlike linkvec, droy's
> `dash_client.go` imports the client package **unaliased**
> (`"github.com/mainvec/mvp/mvpgo/mvp/client"`, used as `client.`), so there's no
> `mvpclient` alias to rename — just repoint the path. The struct field
> `mvpClient *client.Client` is a field name, not a package selector; leave it
> (or rename for consistency).

**Hardcoded `x-mvp-auth` (flip to `x-mvep-auth`)** — Go:
`droy-engine/droyd/config_sync_handlers.go:408`,
`droy-engine/droyd/session.go:473`,
`droy-engine/droyd/syncin_handlers.go` (≈5 occurrences: 1170, 1207, 1243, …).
TS comment: `droy-dashboard/mvpapi/js/api/client/dash_client.ts:57`.

**TypeScript** (`@mainvec/mvpjs` → `@mainvec/mvep`):
`droy-dashboard/frontend/droy-dashboard-ui/package.json` (`file:` dep),
`…/vite.config.ts` (alias),
`droy-dashboard/mvpapi/js/api/client/{dash_client.ts,dash_package.ts}` (imports).
(`droy-dashboard/backend/frontend/dist/**` and `…/droy-dashboard-ui/dist/**` are
built bundles — regenerate, don't edit.)

## Appendix B — linkvec (root `go.work`; 9 main Go modules with `mvpgo`)

> **Status**: Migrated in PR #60 (runtime) and PR #62 (directory rename).

**`go.work`**: remove `../mvpgo/mvpgo` from the `use (…)` block.

**`go.mod` to update**: `linkvec-core`, `linkvec-broker`, `linkvec-relay`,
`linkvec-bench`, `linkvec-tui`, `linkvec-tui2`, `linkvec-desktop`,
`linkvec-mobile/gomobile` (and any others requiring `mvpgo`).

**Generate script**: `linkvec-core/mvpapi/generate_api.sh` (4× `mvp generate …`).

**Hand-written Go runtime imports** to repoint:
`linkvec-core/mvpapi/linkvec/go/api/linkvec_package.go`,
`linkvec-core/mvpapi/linkvecd/go/api/linkvecd_package.go`,
`linkvec-core/mvpapi/linkvecd/go/linkvecd_impl.go` (+ `/mvp/server`),
`linkvec-core/mvpapi/linkvecd_client/client.go` (+ `/mvp/client`).

> **Import alias** *(v2 addition)*: `linkvecd_client/client.go` aliases
> `mvpclient "github.com/mainvec/mvp/mvpgo/mvp/client"` → rename to `mvepclient`
> and update all `mvpclient.` references (~3 call sites). The struct field
> `mvpClient` is a field name, not a package selector — it can stay as-is or be
> renamed for consistency.

**Hardcoded `x-mvp-auth` (flip to `x-mvep-auth`)**:
`linkvec-ui/vite.config.ts:41` (dev-proxy header),
`tests/e2e-tun/run-test.sh:310,316` (curl `-H "x-mvp-auth: …"`).
(`linkvec/plans/044-*.md` mentions it historically — leave as history.)

**TypeScript**: `linkvec-ui/package.json` (`file:` dep) + `linkvec-ui/src/api/httpService.ts` (import).
(`linkvec-ui/dist/**`, `linkvec-desktop/frontend/dist/**` are built bundles.)

> **Skip the `linkvec/.kilo/worktrees/*` copies** — those are throwaway tool
> worktrees, not the main tree. Migrate them only if you still use them.

## Appendix C — girafa (already migrated — reference)

Done in mainvec/girafa#2: single Go module, `runtime/go v0.6.0`, no hardcoded
`x-mvp-`. To pick up `x-mvep-`, just bump its require to `runtime/go v0.7.0` and
`go mod tidy` (no header edits needed).

---

## Changelog (v1 → v2)

| Addition | Where |
|----------|-------|
| Import alias rename guidance (`mvpclient` → `mvepclient`) | Go migration step 3, Appendix B |
| Indirect `mvpgo` requires must be manually replaced | Go migration step 2 |
| Step reordering: tidy **after** import repoint + regen | Go migration steps 3–5, ordering callout |
| Project docs update step (`AGENTS.md`, `README.md`, etc.) | Go migration step 7 |
| Build tooling check step (goreleaser, scripts, CI, gitignore) | Go migration step 8 |
| `npm install` regenerates lock file — don't hand-edit | TS migration step 5 |
| gopls stale cache after `git mv` — restart language server | Protect hand-edited files section |
| Expanded verification checklist (aliases, docs, tooling) | Verification checklist |
| Import alias row in mapping table | What changed table |
| droy backfill: indirect requires + unaliased client import notes | Appendix A |