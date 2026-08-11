# 040 — CLI Command Groups (Nested Subcommands)

- Issue: [#40](https://github.com/mainvec/mvep/issues/40) (`feat(cli): command groups for nested subcommands`)
- Related: [plan 023](023-cli-generation-complete-reusable.md) item **C2** (this
  plan delivers it), [plan 025](archived/2026-08-10-025-runtime-cli-builder.md)
  (the descriptor and `cli.New` this builds on), [#23](https://github.com/mainvec/mvep/issues/23)
  (type-coverage catalogue, unaffected)
- Branch: `feat/040-cli-command-groups`

## Problem / Goal

`cli.New` builds a strictly flat command tree. Its own doc comment says so —
*"Every CommandDesc becomes a ugo subcommand under the root"* — and the loop
ends in `root.AddCommand(sub)` with no intermediate parent. A package with
twenty-three commands therefore produces twenty-three siblings under `--help`,
in one undifferentiated list.

Nothing in the pipeline can express otherwise. `toolkit.CommandDef` is
`{Id, Title, Alias, Desc, Fields, ResultFields}` and `mvep.CommandDesc` is
`{Name, Alias, Desc, New, Fields, Result}`. Neither carries a group.

Consumers cannot retrofit it either. Reparenting a generated command requires
detaching it from root, and `ugo/cli` has no `RemoveCommand`, no exported
`Commands()` accessor, and keeps `commands []*Command` / `parent *Command`
unexported. Calling `AddCommand` a second time on a command returned by `Find`
does not move it — `AddCommand` sets `sub.parent = c` and appends, so the
command stays listed under root while `CommandPath()` reports the new path.
The only workaround is hand-built wrapper commands that delegate, with
duplicate help entries.

The goal is to let a spec declare `"group": "server"` on a command and get
`svc server start` — reusing ugo's existing nesting rather than building a
parallel mechanism.

## Goals

- A spec can place any command under a group, at arbitrary depth.
- Group parents carry their own title, description, aliases and hidden flag.
- Help output, dispatch, and persistent-flag inheritance work through groups
  with **no change to `ugo/cli`**.
- Ordering of groups and of commands within a group is deterministic.
- A spec that declares no group generates byte-identical output to today.
- A group that cannot be built is a **generate-time error**, consistent with
  plan 025's strictness decision.
- The spec additions are language-neutral, so a future JS CLI generator reads
  the same data (plan 023 D4).

## Non-goals

- **Help categories** — a flat command list visually grouped by heading, the
  kubectl style. That is a different feature from nesting: the invocation stays
  `svc start_llm_server` and only the help rendering changes.
  `ugo/cli.Command.Annotations` is documented as the hook for it
  (*"key/value pairs that can be used by applications to identify or group
  commands"*), and it is currently unused. If both are ever wanted they should
  share the `group` field, not compete for it.
- **Inferring groups** from common prefixes in command aliases. Too magical,
  and it would silently restructure existing CLIs on upgrade.
- **Group-level persistent flags declared in the spec.** A consumer can already
  attach them by hand via `app.Root().Find(...)` then `PersistentFlags()`.
  Expressing them in the spec is a separate question tangled up with flag
  inheritance semantics.
- Shell completion, `oneOf`, TypeScript parity. Unchanged from plan 025.

## Why `ugo/cli` needs no change

This was the open question going in, and the answer is that every capability is
already present and merely unreached. Verified against `ugo` v0.7.0:

| Requirement | Where it already works |
|---|---|
| Arbitrary nesting | `AddCommand` works on any `*Command`, not just root (`command.go:113`); `parent` chain is maintained |
| Dispatch into a nested command | `Execute` calls `findCommand(root, args)`, which walks the tree flag-aware, so `svc --output json server start --model x` resolves correctly regardless of flag position (`framework.go`) |
| A group parent prints help instead of erroring | `Execute`: when `RunE == nil && Run == nil` it calls `(&CommandHelp{cmd}).WriteHelp(stderr)` and returns nil |
| Help lists a non-runnable group | `CommandHelp.Available()` returns true when `HasSubCommands()` is true, even if `Runnable()` is false (`help.go`) |
| Help shows the full path | `CommandPath()` and `UseLine()` recurse through `parent`; `HelpTemplate` already branches on `.HasSubCommands` |
| Persistent flags reach leaves through a group | `mergedFlags(cmd)` combines the command's own flags with `InheritedFlags()`, which walks ancestors (`command.go:94`) |
| Group aliases | `Command.Aliases` + `findSub` alias matching |
| Hidden groups | `Command.Hidden`, honoured by `Available()` |

So this is entirely a `toolkit` + `runtime/go` change. No `ugo` release, no
version bump, no companion issue.

## Proposed Design

### Spec surface

Two additions, both optional.

**1. `group` on a command** — a path, `/`-separated, empty or absent meaning
root:

```json
"StartLLMServerCmd": {
  "title": "Start an LLM Server Instance",
  "alias": "start",
  "group": "server",
  ...
}
```

Depth is expressed by the path (`"server/keys"`), so arbitrary nesting needs no
further spec change. Note that `alias` becomes the *leaf* name — `svc server
start`, not `svc server start_llm_server` — which is the point of grouping.

**2. `commandGroups` at the top level** — metadata for the groups, keyed by
full path:

```json
"commandGroups": {
  "server":      { "title": "LLM Servers", "desc": "Start, stop and inspect servers" },
  "server/keys": { "title": "API Keys", "aliases": ["key"] }
}
```

Entirely optional. A group referenced by a command but absent here is created
with the path segment as its name and no description. Intermediate segments are
auto-created the same way, so `"group": "a/b"` works without declaring `a`.

### Toolkit structs

| Struct | Add |
|---|---|
| `CommandDef` | `Group string \`json:"group,omitempty"\`` |
| `SrvDef` | `CommandGroups omap.OMap[string, GroupDef] \`json:"commandGroups,omitempty"\`` |
| new `GroupDef` | `{ Title, Desc string; Alias []string; Hidden bool }` |

`FieldDef.Tags` was defined in the JSON schema but absent from the Go struct,
so `tags: ["required"]` validated and was then silently dropped on unmarshal
(fixed in #23/#24). T2 carries an explicit round-trip test so `group` cannot
repeat that.

### Descriptor

| Type | Add |
|---|---|
| `mvep.CommandDesc` | `Group string` — full path, empty for root |
| `mvep.PackageDesc` | `Groups []GroupDesc` — ordered, includes auto-created intermediates |
| new `mvep.GroupDesc` | `{ Path, Name, Title, Desc string; Aliases []string; Hidden bool }` |

Ordered slices, never maps — same reasoning as plan 025: `omap.OMap[K,V]` is a
plain map and ordering only exists through its iterators, so codegen must emit
through them and the descriptor must hold the result in order.

`Groups` is a flat list carrying the full path rather than a nested tree.
Flattening keeps the emitted literal simple and lets `cli.New` build the tree in
one pass; the parent of `a/b` is found by trimming the last segment.

### `cli.New`

One change to the existing loop. For each `CommandDesc`, resolve its parent —
root when `Group` is empty, otherwise a `*cli.Command` memoised per path and
created on demand in `desc.Groups` order — and `AddCommand` the leaf to that
parent instead of to root.

Group parents are built with `Usage` = last path segment, `Short` = `Title`,
`Long` = `Desc`, plus `Aliases` and `Hidden`, and **no `RunE`**, so ugo prints
their help automatically. They also get the same unknown-subcommand guard the
root already carries:

```go
Args: func(cmd *cli.Command, args []string) error {
    if len(args) > 0 {
        return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
    }
    return nil
},
```

Without it, `svc server bogus` is treated as a positional argument and silently
prints help — the same trap the root guard exists to close.

`app.commands` stays keyed for lookup; the key becomes the full path so two
groups can each hold a `list` command.

### Generate-time validation

Failing at generation, not at `cli.New`, follows plan 025: the error reaches the
developer holding the spec rather than an end user at startup. Rejected:

- a group path segment equal to a sibling command's name or alias
- a group alias equal to a sibling command's name or alias, or to a sibling
  group's name or alias
- two commands resolving to the same name under the same parent
- an empty path segment, or a leading/trailing `/`
- a `commandGroups` entry no command references (a typo that would otherwise
  produce a silently empty group)

Every message names the spec path, the group, and the colliding command.

## Affected Modules

| Module | Change |
|---|---|
| `toolkit/toolkit.go` | `CommandDef.Group`, `SrvDef.CommandGroups`, `GroupDef` |
| `toolkit/toolkit_go.go` | Group ordering helpers; descriptor emission |
| `toolkit/toolkit_runner.go` | Group validation in the existing strict-check pass |
| `toolkit/resources/codegen_templates/go/go_package_code.txt` | Emit `Groups` and `CommandDesc.Group` |
| `toolkit/resources/mvepspec/0.2/schema/*` | `group`, `commandGroups` properties |
| `toolkit/toolkit_javascript.go` | Tolerate the new fields; no behaviour change |
| `runtime/go/mvep/descriptor.go` | `GroupDesc`, `CommandDesc.Group`, `PackageDesc.Groups` |
| `runtime/go/mvep/cli/app.go` | Parent resolution in `New` |
| `runtime/go/mvep/cli/README.md` | Document groups |
| `toolkit/MVEP_SKILL.md`, `toolkit/MVEP_ROADMAP.md`, `docs/cli-builder-migration.md` | Spec syntax, roadmap status, adoption notes |
| `runtime/ts` | Untouched |
| `ugo` | **Untouched** |

Import direction is preserved: `toolkit → runtime/go`, never the reverse.

## Risks and Compatibility

| Risk | Mitigation |
|---|---|
| **Adding `group` changes how users invoke a command.** `svc start_llm_server` becomes `svc server start` | Opt-in per command; absent `group` is byte-identical to today. No automatic compat alias is generated — see the Decision Log. A consumer wanting one can add a `Hidden` root command reusing the leaf's `RunE`, reached via `app.Root().Find([]string{"server","start"})` |
| **Generated code references descriptor fields absent from the published runtime** | Same constraint plan 025 hit at rollout step 0: `runtime/go` must ship `GroupDesc` and be required by `toolkit/go.mod` **before** any toolkit release emits it |
| The JSON schema rejects unknown properties, so a spec using `group` fails on an older toolkit | Additive optional properties; document the minimum toolkit version. Whether to extend `0.2/schema/2026-01-15.json` in place or cut a new dated schema is Open Question 1 |
| `group` is defined in the schema but dropped on unmarshal | Exactly the `tags` defect (#23). T2 is a round-trip test, written before the emission work |
| Group parents make `--help` output churn for existing consumers | Only for specs that opt in. Golden test asserts ungrouped output is unchanged |
| Emitting `Groups` bloats generated output | A data-only static initialiser, one entry per group, not per command |
| A group name shadows a command name | Generate-time error naming both |

Wire compatibility is unaffected: grouping is a CLI presentation concern.
`GetName()`, routes, envelopes and encodings are untouched, and `Group` is not
consulted by the server or client.

## Verification

- `cd toolkit && go test ./...`; `cd runtime/go && go test ./...`
- **Round-trip:** a spec with `group` and `commandGroups` unmarshals into
  `SrvDef` with both populated — guards against the `tags` class of defect.
- **Golden:** a fixture spec with groups at depth 1 and 2 produces `--help`
  output that is byte-stable across repeated runs in one process.
- **Backward compat:** an existing fixture spec with no groups generates output
  byte-identical to the committed baseline.
- **Dispatch:** `svc server start --model x` reaches `StartLLMServerCmd` with
  the field bound; the same invocation with the flag *before* the subcommand
  (`svc --output json server start`) also resolves, exercising ugo's flag-aware
  `findCommand`.
- **Group help:** `svc server` with no subcommand prints the group's help and
  exits 0.
- **Unknown subcommand:** `svc server bogus` returns an error naming the group
  path, and does not print help as if it succeeded.
- **Inheritance:** a persistent flag registered on `app.Root()` appears under
  *Global Flags* in `svc server start --help` and binds when supplied.
- **Aliases and hidden:** a group alias resolves; a hidden group is absent from
  root help but still invocable.
- **Negative:** each validation case in
  [Generate-time validation](#generate-time-validation) fails `mvep generate`
  with a non-zero exit and a message naming the spec path.
- **JS tolerance:** `--lang js` over the grouped fixture succeeds and its output
  is unchanged by the new metadata.

## Rollout

0. **Release ordering is a hard constraint.** `runtime/go` ships `GroupDesc`
   and `CommandDesc.Group` first, and `toolkit/go.mod` is bumped to it, before
   any toolkit release emits them. Until that bump lands, the toolkit's own
   generated `mvepapi/api/mvep_package.go` must not be regenerated.
1. T1–T3 land the spec, struct and descriptor surface. Nothing changes for a
   spec that declares no group.
2. T4–T5 land emission and validation.
3. T6–T7 land the `cli.New` tree construction and its tests.
4. T8 documents it and marks plan 023's C2 delivered.

No migration: grouping is inert until a spec adds `group`.

## Decision Log

- **Reuse `ugo/cli` nesting; change nothing upstream.** The audit above found
  dispatch, help rendering, non-runnable parents, flag inheritance, aliases and
  hidden commands all already working. Adding a mvep-side grouping mechanism, or
  patching ugo, would duplicate what exists.
- **`group` is a path string, not a single token.** A `/`-separated path buys
  arbitrary depth for the same field, and the flat `Groups` list plus
  last-segment-trimming keeps the builder simple. A repeated `groups: []` array
  was considered and rejected as harder to read in a spec and no more capable.
- **Group metadata lives in a top-level `commandGroups` map**, not inferred from
  the first command that mentions the group. Inference makes a group's title
  depend on command ordering, which is exactly the kind of implicit coupling the
  descriptor work removed.
- **Groups are auto-created when referenced but undeclared.** Declaring
  `commandGroups` should be for adding a description, not a precondition for
  using a group. The inverse — a declared group nobody references — *is* an
  error, because it is almost always a typo.
- **No automatic flat-name compatibility alias.** Considered emitting the
  pre-group name as a hidden root command. Rejected: adding `group` is a
  deliberate change to your own CLI's surface, and silently keeping both names
  makes `--help` and reality disagree while giving no signal about when the old
  form goes away. The manual escape hatch is three lines and stays under the
  consumer's control.
- **Validation at generate time**, matching plan 025.
- **`Groups` is a flat ordered slice, not a tree.** Simpler literal, one-pass
  construction, and consistent with the ordered-slices-never-maps rule.
- **Help categories are explicitly out of scope**, but `group` is deliberately
  named so that a later categories feature can consume the same field rather
  than introducing a second, competing one.

## Open Questions

1. Extend `0.2/schema/2026-01-15.json` in place, or cut a new dated schema?
   In-place keeps one schema and is additive for specs, but a spec using `group`
   then fails validation on an older toolkit with a confusing message rather
   than a version mismatch.
2. Should `commandGroups` support a group-level `example`, feeding
   `Command.Example`? Cheap to add now, awkward to retrofit.
3. Does the group path participate in `app.commands` keys only, or should
   `CommandDesc.Group` also be exposed through any future discovery endpoint?
4. Should a group be allowed to be `Hidden` while its children are not — and if
   so, are the children still reachable? ugo permits it; the plan currently
   assumes yes and leaves it untested.

## Progress

- [x] T1 — Spec: `group`, `commandGroups`, schema
- [x] T2 — Toolkit structs and unmarshal round-trip test
- [ ] T3 — Descriptor: `GroupDesc`, `CommandDesc.Group`, `PackageDesc.Groups`
- [ ] T4 — Codegen emission with deterministic ordering
- [ ] T5 — Generate-time validation and collision errors
- [ ] T6 — `cli.New` builds the nested tree
- [ ] T7 — Tests: golden, dispatch, help, inheritance, backward compat
- [ ] T8 — Documentation

## Tasks

### T1 — Spec: `group`, `commandGroups`, schema

Add the optional `group` property to a command and the optional top-level
`commandGroups` object to the JSON schema, with a fixture spec under
`toolkit/testdata/` exercising depth 1 and depth 2, an alias, and a hidden
group. Resolve Open Question 1 here.

**Notes:**
- Open Question 1 resolved: extend `0.2/schema/2026-01-15.json` in place
  (additive optional properties), and keep the no-extension copy
  (`resources/mvepspec/0.2/schema/2026-01-15`) in sync — fixtures that pin the
  file-path `$schema` (no `.json`) resolve through it on disk.
- Fixture `13_command_groups.jsonc` covers: `server` (depth 1), `server/keys`
  (depth 2), an alias (`server/keys` → `key`), a hidden group (`hidden`, with a
  `SecretCmd` so it is referenced and thus valid), and a root command
  (`UngroupedCmd`) for backward-compat.
- Schema field/def descriptions are kept neutral (no "CLI", no plan number):
  groups may be reused beyond the CLI.

### T2 — Toolkit structs and unmarshal round-trip test

Add `CommandDef.Group`, `SrvDef.CommandGroups` and `GroupDef`. **Write the
round-trip test first**: unmarshal the T1 fixture and assert both fields are
populated. This is the guard against repeating the `tags` drop-on-unmarshal
defect (#23).

**Notes:**
- Test-first: `TestCommandGroupsRoundTrip` written and confirmed failing
  (undefined fields) before the structs were added.
- `GroupDefs omap.OMap[string, GroupDef]` mirrors the existing `CommandDefs`;
  added a `GroupDefs.Get`.
- Verified `go test .` passes after the round-trip test; no regressions.

### T3 — Descriptor types

Add `mvep.GroupDesc`, `CommandDesc.Group` and `PackageDesc.Groups` to
`runtime/go/mvep/descriptor.go`. Additive; existing descriptors keep compiling.
Tag and release `runtime/go` before T4 ships — see Rollout step 0.

### T4 — Codegen emission

Emit `Groups` and each command's `Group` from `go_package_code.txt`, iterating
through `omap` iterators so ordering is defined. Auto-created intermediate
groups are materialised here, not in `cli.New`, so the descriptor is complete on
its own.

### T5 — Generate-time validation

Implement the checks in
[Generate-time validation](#generate-time-validation) inside the existing strict
pass in `toolkit_runner.go`. One test per rejected case, each asserting a
non-zero exit and a message naming the spec path.

### T6 — `cli.New` builds the nested tree

Resolve each command's parent, memoising group commands by path and creating
them in `desc.Groups` order. Group parents get `Short`/`Long`/`Aliases`/`Hidden`
and the unknown-subcommand `Args` guard, and no `RunE`. Key `app.commands` by
full path.

### T7 — Tests

Every item in [Verification](#verification) above. The backward-compat golden
test matters most: it is what proves an existing consumer sees no change.

### T8 — Documentation

Update `runtime/go/mvep/cli/README.md` with a groups section,
`toolkit/MVEP_SKILL.md` with the spec syntax, and mark **C2** delivered in plan
023. Note in `toolkit/MVEP_ROADMAP.md` that help categories remain a separate,
unbuilt item. Add a short section to `docs/cli-builder-migration.md` for
consumers adopting groups.
