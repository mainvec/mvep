# Plan 060 — `mvep` command identity by descriptor name

**GitHub Issue**: [#60](https://github.com/mainvec/mvep/issues/60)

## Progress

- [ ] T1: Key `a.commands` by the descriptor `Name`
- [ ] T2: `mvep list` shows `Name` and the command description
- [ ] T3: Resolve `Name` in `exec`/`send`/`describe`
- [ ] T4: Tests + docs

## Problem / Goal

The `mvep` namespace is the machine/scripting surface of the generated CLI
(`mvep exec`, `send`, `list`, `describe`), and its command identity must match
what the **server** understands. The server identifies commands by their
descriptor **`Name`** (the CamelCase name, e.g. `StartServerCmd`) — see
`NewPackageFromDesc` (`descriptor.go`), where `InstanceOf(compName)` keys on
`c.Name`, and `Cmd.Cmd` on the wire carries that name.

Today the `mvep` surface keys by the **alias** instead (`commandName` /
`a.commands` in `app.go`). Two problems result:

1. **Wire mismatch.** `mvep send` sets `Cmd.Cmd` to the alias (e.g.
   `echo_cmd`), which a real server's `InstanceOf` does not accept (`EchoCmd`).
   The machine surface does not round-trip.
2. **Aliases collide across groups.** Aliases repeat across groups (`repo list`
   and `server list` both key as `list`), so the second registration overwrites
   the first, and `mvep list`/`describe`/`exec` resolve to whichever was last.

The fix is to make the `mvep` surface use the **descriptor `Name`** — unique,
stable, and wire-consistent. `mvep list` prints the `Name` together with the
command's description. The human flag path (`svc server start`) is unchanged:
groups and aliases remain a presentation concern there.

## Goals

- `a.commands` is keyed by the descriptor `Name` (e.g. `StartServerCmd`), so no
  two commands collide.
- `mvep list` prints each command's `Name` and its description.
- `mvep describe <Name>` / `mvep exec <Name>` / `mvep send` (`Cmd.Cmd = <Name>`)
  resolve by `Name`.
- The ugo command tree (flag path) is unchanged: `sub.Usage` stays the leaf
  alias/snake_case name, since groups are separate parent nodes.
- All existing tests pass; new tests cover a grouped descriptor.

## Non-goals

- Supporting group/alias paths in the `mvep` surface (grouped or bare) — the
  `mvep` surface is `Name` only.
- `snake_case(Name)` convenience aliases — omitted for simplicity; `Name` is
  the single canonical form.
- Changing the server-side identity space or the group model.

## Proposed Design

### T1 — Key `a.commands` by `Name`

`commandName()` currently returns the leaf alias (or snake_case of the name).
Split the concepts:

- **`a.commands` key** = the descriptor `Name` (`cmdDesc.Name`), used by
  `commandNames()` and by `exec`/`send`/`describe` resolution.
- **`sub.Usage`** = the leaf (`commandName`), unchanged, so the flag-path tree
  is untouched.

In `New()`, register `app.commands[cmdDesc.Name] = cmdDesc` but set
`sub.Usage = commandName(cmdDesc)`. The namespace-collision check stays on the
leaf/alias (a group named `mvep` is rejected separately).

### T2 — `mvep list` shows `Name` and description

`mvep list` prints `Name` and the command's `Desc` (from `CommandDesc.Desc`),
e.g. `StartServerCmd   Start an LLM server`. Under `--mvep-output json`, emit a
JSON array of `{name, description}` objects rather than bare strings, so the
machine form carries both fields.

### T3 — Resolve `Name` in `exec`/`send`/`describe`

`decode()` (the choke point for `exec`/`send`) and `registerDescribe` do
`a.commands[name]`, which now resolves `Name` directly. No ambiguity fallback
is needed — descriptor names are unique.

The `exec`/`describe` verbs keep their current single-arg (or ≤1-arg) guards:
`Name` has no `/`, so it stays one arg.

### T4 — Tests + docs

Rewrite `group_identity_test.go` (renamed to reflect `Name` identity) to assert:
- `mvep list` emits `Name` and description for grouped and root commands.
- `mvep describe StartServerCmd` / `mvep exec StartServerCmd` resolve.
- `mvep send` with `Cmd.Cmd = StartServerCmd` resolves.

Update the CLI README and `CHANGELOG.md`.

## Affected Modules

- `runtime/go/mvep/cli/app.go` — `New` loop (map key), `commandNames`.
- `runtime/go/mvep/cli/describe.go` — `registerDescribe` `Name` field.
- `runtime/go/mvep/cli/exec.go` — unchanged (single-arg `Name`).
- `runtime/go/mvep/cli/send.go` — unchanged (`Cmd.Cmd` is `Name`).
- `runtime/go/mvep/cli/*_test.go` — new grouped tests; existing tests unaffected.
- `runtime/go/mvep/cli/README.md`, `CHANGELOG.md` — docs.

## Tasks

### T1: Key `a.commands` by `Name`

**Outcome**: `a.commands` keyed by `cmdDesc.Name`; `sub.Usage` stays `commandName`.

**Verification**: `go test ./runtime/go/mvep/cli/...` passes; a grouped descriptor's `mvep list` shows `Name`s.

**Notes**: The namespace-collision check stays on the leaf.

### T2: `mvep list` shows `Name` and description

**Outcome**: `mvep list` prints `Name` + `Desc`; JSON mode emits `{name, description}` objects.

**Verification**: `mvep list` and `mvep list --mvep-output json` show both fields.

**Notes**: This gives the machine surface self-describing command names.

### T3: Resolve `Name` in `exec`/`send`/`describe`

**Outcome**: `decode` and `registerDescribe` resolve `Name`; no ambiguity fallback.

**Verification**: `mvep exec StartServerCmd` and `mvep describe StartServerCmd` resolve; `mvep send` with `Cmd.Cmd = StartServerCmd` resolves.

**Notes**: Names are unique, so no fallback is needed.

### T4: Tests + docs

**Outcome**: `group_identity_test.go` rewritten for `Name` identity; README + CHANGELOG updated.

**Verification**: New tests pass; existing tests still pass.

**Notes**: Closes the coverage gap from the issue.

## Risks and Compatibility

- **`mvep list` output changes** from bare aliases to `Name` + description. This
  is the intended fix; consumers scripting `mvep list` must switch to `Name`.
- **`mvep send` `Cmd.Cmd` changes** from alias to `Name`, making it wire-compatible
  with the server. Existing bare-alias `send` inputs no longer resolve — a
  behavior change, but the old form was never server-valid anyway.
- **Server identity space is untouched** — no wire-format or server change.
- **Human flag path is unchanged** — groups and aliases still work for `svc
  server start`.
