# Plan 022 — CLI generation: complete and reusable

> **STATUS: DRAFT — wish list / requirements catalogue for discussion.**
>
> This document is a prioritised wish list first and a delivery plan second.
> It is deliberately written project-independent: the driving scenario is the
> generic one of *"a spec-defined API whose CLI is a thin client over a
> daemon"*, which is the shape every MVEP adopter with a long-running server
> ends up in.
>
> Nothing here is approved for implementation yet. The `## Tasks` section is a
> proposed decomposition, not a commitment.

- **Issue**: https://github.com/mainvec/mvep/issues/23
- **Branch**: `feat/23-cli-generation-complete-reusable`
- **Companion issue**: https://github.com/mainvec/ugo/issues/5 (`feat/5-cli-persistent-flags-error-propagation`)
- **Target release**: `toolkit` next minor + `ugo` next minor (companion)
- **Depends on**: a set of `github.com/mainvec/ugo` `cli` package changes
  (see [Prerequisites in `ugo/cli`](#prerequisites-in-ugocli)). Those are
  hard blockers for several items and should land first.

---

## Problem / Goal

The MVEP toolkit generates a CLI for every spec, but that CLI is a
**scaffold, not a product**. Two independent defects make it unusable as
shipped, and adopters discover both only after they have built on it:

1. **It is not complete.** Only `string`, `boolean` and `int32` fields become
   flags. `recRef`, `map`, `repeated`, and every other scalar type are
   dropped. When a *required* field is dropped, the generated command cannot
   be invoked correctly at all — and the only signal is a comment in the
   generated source. Generation still succeeds and the binary still builds.

2. **It is not reusable.** The CLI is emitted as `package main` with a
   hardcoded `commandRunner = <pkg>.GetCommandRunner()`. There is no way for
   a host program to supply its own runner. Since `GetCommandRunner()`
   resolves to the `*_impl.go` stubs in the API package, and the API package
   is by construction the dependency-graph leaf, **a CLI that talks to a
   daemon can never live in the generated package** — the client would create
   an import cycle. Adopters are forced to hand-copy the entire generated
   command tree into a separate module, where it immediately begins to drift
   from the spec.

On top of those, the generated CLI discards every command result
(`_, err := commandRunner.RunXxxCmd(...)`) and never sets a non-zero exit
code, so it cannot be used in a script even for the commands it *can* express.

The goal of this plan is to make a generated CLI something an adopter can
ship: **complete** (no spec construct is silently unrepresentable),
**reusable** (embeddable in any host, with any runner), and **well-behaved**
(prints results, exits correctly).

## Goals

- A generated CLI can be imported and embedded by a host program in another
  module, with a caller-supplied `*api.PkgCommandRunner`.
- The same generated CLI serves both an in-process runner and a remote,
  client-backed runner, with no template forking and no duplicated command
  tree.
- Every field type in the spec has a defined, documented CLI representation.
  No field is ever silently dropped.
- Command results are rendered by default, in both human-readable and
  machine-readable form.
- The CLI honours the standard process contract: exit codes, stdout/stderr
  separation, signal handling.
- Spec additions introduced here are **language-neutral**, so a future JS/TS
  CLI generator can consume them unchanged.
- Existing specs keep generating today's output until they opt in.

## Non-goals

- **A JS/TS CLI generator.** There is no CLI template for `--lang js` today
  and this plan does not add one. It does require that every spec addition be
  language-neutral and that the JS templates continue to ignore them
  gracefully, so that generator can be added later without a spec migration.
  See [JS/TS accommodation](#jsts-accommodation).
- **Replacing `ugo/cli` with `cobra`.** The repo carries dormant
  `go_cli_cobra_*.txt` templates; this plan does not revive them. See the
  Decision Log.
- Reworking the runtime (`runtime/go/mvep`). This is a `toolkit/` + `ugo/`
  plan. It does not touch the wire protocol, `PackageHandler`, or the server.
- Interactive/TUI modes, prompting, or colour output.
- Roadmap items that this plan *depends on* but does not deliver — enums,
  default values, validation rules (`min`/`max`/`pattern`). See
  [Relationship to the roadmap](#relationship-to-the-roadmap).

---

## Current State (verified 2026-08-06)

Everything in this section was read from source, not assumed.

### Generation entrypoint

`ExecuteGenerate(ctx, cmdIn, cmdOutdir, cmdLang, skipCmd, format)` —
`toolkit/toolkit_runner.go:29`. The complete set of user-facing knobs is
`--in`, `--outdir`, `--lang`, `--format`, `--skip-cmd`.

CLI artifacts are written to `<outdir>/cmd/<srvDef.Name>/` as
`<name>_main_cmd.go` and `<name>_version.go`
(`toolkit/toolkit_runner.go:159-202`), from
`toolkit/resources/codegen_templates/go/go_cli_main.txt` and
`go_cli_version.txt`. Both honour the `NOMVEP` / `NOMVGEN` / `NOWOGEN`
markers (added 2026-07-22).

`--skip-cmd` is all-or-nothing: it suppresses every CLI artifact.

### What the template hardcodes

| Line (`go_cli_main.txt`) | Hardcoded |
|---|---|
| 8 | `package main` |
| 42 | `commandRunner = {{.NAME}}.GetCommandRunner()` |

There is no `SetCommandRunner`, no `NewCLI(runner)`, no configurable output
package, and no hook between argument parsing and dispatch.

`NewCli()` *is* exported, but in `package main`, which makes it unimportable.

### Type coverage

Flag variables are only declared for three types
(`go_cli_main.txt:68-75`): `string`, `boolean`, `int32`.

| Spec construct | Today |
|---|---|
| `string`, `boolean`, `int32` | Flag generated |
| `int64`, `uint32`, `sint32`, `float`, `double`, `bytes`, `timestamp`, `duration`, `uuid` | No flag |
| `recRef` | Flag emitted **commented out**, with `//CLI WARNING: unsupported field type ... Defaulting to string type` |
| `repeated` (any type) | Skipped, with `//WARNING: unsupported repeated field type` |
| `map` | No flag |
| `recDef`, `oneOf` | No flag |

Both warning paths also call `PrintStdErr`, but generation still exits 0, so
nothing in CI notices.

### Result and error handling

Generated handlers are uniformly:

- the result is discarded — `_, err := commandRunner.RunXxxCmd(ctx, cmd)`
- the error is printed to `ctx.Errout()`
- **no exit code is set**, so a failed command still exits 0

### What already works well

- `CommandDef.Alias` → CLI command name (`go_cli_main.txt:77`).
- `FieldDef.Alias` → CLI flag name (`go_cli_main.txt:95-97`).
- `NOMVEP` protection on both CLI artifacts.
- Per-command `Version` with an automatic `-v` flag.

### Spec structures (`toolkit/toolkit.go`)

```
CommandDef  { Id, Title, Alias, Desc, Fields, ResultFields }          // :35
RecordDef   { Name, Title, Desc, Fields }                             // :44
FieldDef    { Id, Title, Alias, Desc, Fnum, Type, Repeated,
              RecRef, MapValueType }                                  // :75
SrvDef      { Id, Name, Namespace, Title, Base, Desc, Version,
              Commands, Records, GenOpts, ProtocOpts }                // :92
GenOptsDef  = omap.OMap[string, string]                               // :90
```

Two consequences that constrain every proposal below:

1. **`FieldDef` has no `Tags` field.** The JSON schema *does* define
   `tags` (`resources/mvepspec/0.2/schema/2026-01-15.json:290`), so
   `tags: ["required"]` validates — and is then dropped on unmarshal. **No
   codegen path can currently see which fields are required.** This is a
   pre-existing defect that blocks required-flag enforcement, and it is
   already on the roadmap as *"Required/optional semantics"*.

2. **`GenOptsDef` is a flat `map[string]string`.** A nested
   `gen_options: { cli: { flagStyle: ... } }` object is not representable
   without changing that type. See
   [Spec additions](#proposed-spec-additions) for the flat-key approach.

### `ugo/cli` capability audit

Source: `/Users/hi/Development/mainvec/ugo/cli`.

**Present, and better than assumed:**

| Capability | API |
|---|---|
| Nested subcommands | `AddCommand` works on *any* `Command`, not just root (`command.go:68-73`) |
| Positional arg validation | `PositionalArgs` + `MinArgs` / `MaxArgs` / `ExactArgs` / `RangeArgs` (`command.go:127-159`) |
| Rich help metadata | `Command.Long`, `.Example`, `.Aliases`, `.Hidden`, `.Annotations` (`command.go:4-50`) |
| Customisable help | `HelpTemplate` + `HelpFuncs` (`help.go:15-43`) |
| Framework returns errors | `Framework.Run(ctx) error`, never calls `os.Exit` (`framework.go:48`) |
| Custom int32 flags | `Int32Var` / `Int32` (`flag.go:50-72`) |
| Extension hooks | `Initializer`, `Preprocessor` |

**Missing, and blocking:**

| Gap | Impact |
|---|---|
| **Persistent/inherited flags** | A global `--output` or `--endpoint` cannot exist alongside nested subcommands |
| **`Run` cannot return an error** — `Execute` ignores its result and returns `nil` (`framework.go:103-104`) | Exit codes are impossible |
| No `Int64Var` / `Float64Var` / `DurationVar` / `StringSliceVar` / `BytesVar` | Blocks full type coverage |
| No required-flag enforcement | Blocks `required` semantics |
| No long-form shorthand pairing (`-v` / `--verbose`) | stdlib `flag` treats them as separate flags |
| No mutually-exclusive flag groups | Blocks `oneOf` |
| No completion hooks, no help categories | Blocks discoverability items |

---

## Design decisions (settled)

These were decided before drafting and are recorded here so the wish list
reads unambiguously. Rationale is in the [Decision Log](#decision-log).

| # | Decision |
|---|---|
| D1 | The generated CLI supports **both** in-process and client-backed runners, selected purely by which runner the host injects. One template, no forking. |
| D2 | `recRef` flattening is **on by default, depth-limited to 1**, with `--<field>-json` and `--<field>-file` escape hatches always generated. |
| D3 | The `cli` package changes ship as a **new `ugo` minor**, not a fork vendored into mvep. |
| D4 | **Go-first.** Spec additions are language-neutral so a JS CLI generator can be added later without a spec migration, but no JS CLI is generated by this plan. |

---

## Wish list

Priorities: **P0** = the CLI is not usable without it. **P1** = adopters will
hand-write it repeatedly until it exists. **P2** = polish.

Each item states an acceptance criterion so it can become an issue verbatim.

### A. Reusable, embeddable generated CLI — P0

The blocking defect. Until A1–A3 land, every adopter with a daemon must
hand-copy the generated command tree.

| # | Wish | Acceptance criteria |
|---|---|---|
| **A1** | Emit the CLI as an **importable library package** rather than `package main` | Generated file declares a named package derived from config; a host in another module can import it |
| **A2** | **Injectable runner**: `NewCLI(runner *api.PkgCommandRunner, opts ...Option) *cli.Framework` | No package-level `commandRunner` var; no `GetCommandRunner()` call anywhere in the generated CLI; passing a stub runner in a test dispatches to the stub |
| **A3** | Configurable CLI package path and name | New gen option; when unset, output is byte-identical to today |
| **A4** | **Split entrypoint from wiring.** Command wiring regenerates always; a thin `main.go` is emitted **once** and never overwritten | Re-running the generator over a hand-edited entrypoint leaves it untouched, without requiring a `NOMVEP` marker |
| **A5** | **Lifecycle hooks** as options: `WithPreRun` (establish a connection / authenticate, may return the runner), `WithPostRun`, `WithErrorMapper` | A host can connect to a daemon before dispatch and map domain errors to exit codes without editing generated code |
| **A6** | **Per-command include/exclude**, so a host can hand-write one command and generate the other forty | Excluded commands emit no wiring but still participate in the completeness check (H2) |

### B. Complete type coverage — P0

Today a required `recRef` field yields a command that **cannot be invoked
correctly**, and generation still succeeds. This is the correctness hole.

| # | Wish | Acceptance criteria |
|---|---|---|
| **B1** | Flags for **all scalar types**: `int64`, `uint32`, `sint32`, `float`, `double`, `bytes`, `uuid`, `timestamp`, `duration` | Each has a documented parse format. `timestamp` accepts RFC3339; `duration` accepts Go duration syntax; `bytes` accepts base64 or `@path` |
| **B2** | **`repeated` fields** → repeatable flags (`--tag a --tag b`), with optional comma-splitting | Works for repeated scalars *and* repeated `recRef` (the latter via B3's JSON/file forms) |
| **B3** | **`recRef` fields** → three complementary forms, per D2: (a) flattened dotted flags at depth 1, e.g. `--config.model-path`; (b) `--config-json '<json>'`; (c) `--config-file <path>`, where `-` means stdin | Nested structs are fully addressable. Precedence is documented and deterministic: file < json < individual flags. Depth beyond the limit is reachable only via (b)/(c) |
| **B4** | **`map` fields** → repeatable `--label key=value`, typed by `valueType` | Duplicate keys resolve last-wins, documented |
| **B5** | **`oneOf` fields** → a mutually-exclusive flag group | Supplying two members is a usage error, not a silent overwrite |
| **B6** | **Never silently skip.** An unrepresentable field fails generation | Generator exits non-zero naming spec path, command, and field — unless the field is explicitly opted out in the spec. A `--strict=false` escape hatch preserves today's warn-and-continue behaviour for migration |

**B6 is the highest-value item in this plan.** A warning that lands only as a
comment in generated code is indistinguishable from no warning at all.

### C. CLI ergonomics and naming — P1

`ugo/cli` already supports nesting, positional validation, `Long`, `Example`,
`Aliases` and `Hidden`. Most of this section is generator-side plumbing to
*reach* capabilities that already exist.

| # | Wish | Acceptance criteria |
|---|---|---|
| **C1** | **Naming policy** gen options: flag style `kebab \| camel \| snake`, command style `kebab \| snake` | Sets the convention for a whole surface without an `alias` on every field. Explicit `alias` always wins. Default preserves current output |
| **C2** | **Command groups** → nested subcommands. `"group": "server"` on a command yields `svc server start` | Generator-side only; `AddCommand` already supports it. Ungrouped commands stay at root |
| **C3** | **Positional args** from spec | A field marked positional generates `svc server stop <id>` and wires the matching `ExactArgs`/`RangeArgs` validator |
| **C4** | **Required enforcement** — needs `FieldDef.Tags` (or roadmap `required: true`) to reach codegen first | Missing required flag → usage error, exit 2, before dispatch |
| **C5** | **Shorthand flags** from spec | Collision detected at generation time, not runtime |
| **C6** | **Hidden and deprecated** commands and flags | Hidden maps to `Command.Hidden`; deprecated emits a stderr notice on use. Shares the roadmap's deprecation marker |
| **C7** | `title` → `Short`, `desc` → `Long`, new `example` → `Command.Example` | Generated `--help` is useful with no hand-editing |

### D. Result rendering — P0

A CLI that discards every result is not a CLI. This is the single largest
volume of code an adopter currently hand-writes.

| # | Wish | Acceptance criteria |
|---|---|---|
| **D1** | **Print the result by default** | No `_, err :=` anywhere in generated code |
| **D2** | **`--output json\|yaml\|table\|template`** as a persistent flag | JSON output is valid and field-stable; `template` accepts a Go template string |
| **D3** | **Table hints in spec** — which `resultFields` are columns, their order and headers | A repeated `recRef` result renders as a table with zero hand-written formatting |
| **D4** | **Display hints** — `bytes`→human sizes, `timestamp`→local/relative, `duration`→`1h2m` | Opt-in per field; raw values always recoverable via `--output json` |
| **D5** | **`--quiet`** prints only the primary identifier | Output is pipeline-friendly for shell scripting |

### E. Process contract — P0

| # | Wish | Acceptance criteria |
|---|---|---|
| **E1** | **Documented exit code taxonomy**, honoured: `0` success, `1` command error, `2` usage error, `3` transport/connection error, `4` auth error | An automated test asserts each code |
| **E2** | **Handlers can fail** — requires `ugo` U2 | An error from a handler propagates out of `Framework.Run` to the entrypoint |
| **E3** | **stdout = data, stderr = diagnostics**, strictly | `svc list --output json > f.json` yields a file containing only JSON |
| **E4** | **Signal handling** — SIGINT/SIGTERM cancel the command context | A long-running command aborts cleanly and exits with the conventional code |
| **E5** | **Whole-payload stdin** — `--input-file -` reads a complete command payload as JSON | `cat cmd.json \| svc apply --input-file -` works. Composes with B3 |

### F. Client-backed CLI generation — P0 for daemon architectures

The toolkit already generates a typed client *and* the `PkgCommandRunner`
shape. The adapter between them is the missing link that every adopter with a
daemon writes by hand — and must extend on every spec change.

| # | Wish | Acceptance criteria |
|---|---|---|
| **F1** | Generate a **client-backed runner**: `NewCommandRunner(c *Client) *api.PkgCommandRunner` in the generated client package | One line wires a CLI to a remote daemon. A new spec command is covered automatically with no hand-edit |
| **F2** | Optional **standard connection flags** — `--endpoint <uri>`, `--token`, `--tls-ca` — as persistent flags with env binding and documented precedence | Opt-in via gen option; hosts can override defaults and env var names |
| **F3** | **Local and remote from one binary** — the same generated CLI dispatches in-process or over the wire | Selected by the injected runner (A2). No duplicated command tree |

**F1 alone eliminates the largest category of hand-written glue**, and is what
makes A2 pay off in practice.

### G. Discoverability — P2

| # | Wish | Acceptance criteria |
|---|---|---|
| **G1** | Generated **shell completions** (bash/zsh/fish), including enum-valued flags | `svc completion zsh` emits a working script |
| **G2** | Generated **man pages / markdown reference** | Overlaps the roadmap's `mvp docs`; should share one renderer rather than grow a second |

### H. Generator ergonomics and safety — P1

| # | Wish | Acceptance criteria |
|---|---|---|
| **H1** | **Granular skip** — per-artifact and per-command, replacing all-or-nothing `--skip-cmd` | Can skip the entrypoint while still regenerating wiring. `--skip-cmd` keeps its current meaning |
| **H2** | **Compile-time completeness assertion** — generated code that fails to build if a spec command has neither CLI coverage nor an explicit exclusion | Adding a command to the spec cannot silently bypass the CLI |
| **H3** | **Warnings are loud** — generation diagnostics go to stderr and, under `--strict`, exit non-zero. Never *only* a code comment | CI catches an unrepresentable field |
| **H4** | **Deterministic output** — stable ordering of commands and flags | Regeneration produces no spurious diffs |
| **H5** | **Golden tests** for the CLI templates | Template changes are reviewable as diffs of generated output |

### I. Env and config binding — P2

| # | Wish | Acceptance criteria |
|---|---|---|
| **I1** | Per-flag **env var binding** from spec | Documented precedence: flag > env > config > default |
| **I2** | Optional **config-file binding** for flag defaults | Path is host-supplied via an option, not baked in |

---

## Prerequisites in `ugo/cli`

Companion issue in `github.com/mainvec/ugo`. **U1 and U2 are hard blockers**
for the P0 items and are mildly breaking, so they should land together in one
minor.

| # | Change | Blocks | Note |
|---|---|---|---|
| **U1** | **Persistent / inherited flags** (parent → child) | A5, C2, D2, E5, F2, I1 | Entirely absent today. Without it, a global `--output` cannot coexist with nested subcommands |
| **U2** | **`Run` returns `error`**, and `Execute` propagates it | E1, E2 | `Execute` currently ignores `Run`'s result and returns `nil` (`framework.go:103-104`). Mildly breaking: changes the `Run` field's signature |
| **U3** | Flag wrappers: `Int64Var`, `Float64Var`, `DurationVar`, `StringSliceVar`, `BytesVar` | B1, B2 | Only `Int32Var` was custom-added; the rest come from stdlib `flag` or need `.Var()` adapters |
| **U4** | **Required-flag** enforcement | C4 | Pairs with making `FieldDef.Tags` visible |
| **U5** | **Long-form shorthand pairing** (`-v` ⇄ `--verbose`) | C5 | stdlib `flag` treats them as unrelated flags |
| **U6** | **Mutually-exclusive flag groups** | B5 | |
| **U7** | **Completion hooks** | G1 | |
| **U8** | **Help categories / command groups in help output** | C2 | `HelpTemplate` is already customisable, so this may be additive only |

---

## Proposed spec additions

All additions are **optional** and **language-neutral**. A spec that uses none
of them generates today's output.

### Toolkit struct changes (`toolkit/toolkit.go`)

| Struct | Add | For |
|---|---|---|
| `FieldDef` | `Tags []string \`json:"tags,omitempty"\`` | **Fixes the existing drop-on-unmarshal defect.** Unblocks C4 |
| `FieldDef` | `Short`, `Env`, `Positional`, `CLISkip`, plus display hints | C3, C5, D3, D4, I1, B6 |
| `CommandDef` | `Group`, `Example`, `Hidden`, `Deprecated` | C2, C6, C7 |
| `SrvDef` / `GenOptsDef` | see below | A3, C1, F2 |

Because `FieldDef` and `CommandDef` are plain structs, additive JSON fields
are backward compatible. The JSON schema in
`toolkit/resources/mvepspec/0.2/schema/` must be extended in lockstep, since
unknown properties are currently rejected at validation time.

### The `gen_options` constraint

`GenOptsDef` is `omap.OMap[string, string]` (`toolkit.go:90`) — a **flat
string→string map**. A nested `gen_options: { cli: { ... } }` object cannot be
unmarshalled into it.

Two options:

- **Flat `cli_*` keys** — `cli_package`, `cli_flag_style`, `cli_command_style`,
  `cli_transport`, `cli_strict`. Zero type changes, consistent with the
  existing `go_package` / `go_api_package` / `format` keys. **Recommended.**
- **Widen `GenOptsDef`** to a typed struct or `map[string]any`. Cleaner
  long-term but touches every existing spec's unmarshal path and both the pb3
  and plain generators.

This plan assumes the flat-key approach; see Open Question 2.

### JS/TS accommodation

Per D4 this plan generates no JS CLI. To keep that door open at zero cost:

- Every field above is expressed in the spec and schema, not in Go-specific
  options, so a future `js_cli_*.txt` template can read the same data.
- The JS generator (`toolkit/toolkit_javascript.go`) must **ignore unknown
  CLI metadata without erroring** — verified by a test that runs `--lang js`
  over a spec exercising every new field.
- Any Go-only concern (package path, flag style) lives under a `go_`-prefixed
  or `cli_`-prefixed option that a JS generator can ignore, rather than being
  baked into shared field semantics.

---

## Relationship to the roadmap

Several `toolkit/MVEP_ROADMAP.md` Phase 1–2 items are **prerequisites or
close overlaps**. This plan should not duplicate them.

| Roadmap item | Relationship |
|---|---|
| *Required/optional semantics* (promote `tags:["required"]` to `required: true`) | **Prerequisite for C4.** This plan needs the value to reach codegen at all; whether it arrives as `Tags []string` or a first-class `Required bool` is that item's call. Recommend that item lands first and this plan consumes it |
| *Deprecation markers* | **Shared with C6.** One implementation, consumed by CLI help |
| *Enum type* | **Strong synergy.** Enums give completion candidates (G1) and flag validation for free. Not a blocker |
| *Default values* | **Synergy.** A spec default becomes the flag default |
| *Field-level docs passthrough* | **Overlaps C7.** Same `title`/`desc` plumbing |
| *Validation rules* (`min`/`max`/`pattern`) | **Synergy with C4.** Could extend to parse-time flag validation. Out of scope here |
| *`mvp docs`* | **Overlaps G2.** Should share one renderer, not grow a second |

**Recommendation:** slot this plan *after* the roadmap's *Required semantics*
item, or absorb the minimal part of it (`FieldDef.Tags`) as task T1 here.

---

## Affected Files

### `toolkit/`

| File | Change |
|---|---|
| `toolkit.go` | `FieldDef`, `CommandDef` additions; `GenOptsDef` accessors |
| `toolkit_runner.go` | `ExecuteGenerate` signature/options; CLI output path and package resolution; granular skip; strict mode |
| `toolkit_go.go` | Template data: flattened `recRef` flag sets, type→flag mapping, naming policy, grouping, completeness metadata |
| `toolkit_javascript.go` | Tolerate unknown CLI metadata (no behaviour change) |
| `resources/codegen_templates/go/go_cli_main.txt` | Substantially rewritten: library package, injectable runner, full type coverage, result rendering, error propagation |
| `resources/codegen_templates/go/go_cli_entrypoint.txt` | **New** — write-once thin `main.go` (A4) |
| `resources/codegen_templates/go/go_cli_render.txt` | **New** — output formatting helpers (D1–D5) |
| `resources/codegen_templates/go/go_client_runner.txt` | **New** — client-backed `PkgCommandRunner` (F1) |
| `resources/mvepspec/0.2/schema/*` | New optional properties; bump dated schema |
| `testdata/`, `*_test.go` | Golden tests (H5) |
| `MVEP_SKILL.md`, `MVEP_ROADMAP.md` | Document new options; mark absorbed roadmap rows |

### `ugo/` (companion repo)

| File | Change |
|---|---|
| `cli/command.go` | Persistent flags (U1); required flags (U4); flag groups (U6) |
| `cli/framework.go` | `Run` returns error; `Execute` propagates (U2) |
| `cli/flag.go` | Additional flag types (U3); shorthand pairing (U5) |
| `cli/help.go` | Command categories (U8) |
| `cli/completion.go` | **New** — completion hooks (U7) |

### Not touched

`runtime/go/mvep/**`, `runtime/ts/**`, the wire protocol, `PackageHandler`,
`server`, and `client` transport behaviour.

---

## Risks and Compatibility

| Risk | Severity | Mitigation |
|---|---|---|
| **U2 changes the `Run` field signature** — every existing `ugo/cli` consumer breaks | High | Ship in one `ugo` minor with U1; provide a `RunE`-style additive field first and deprecate `Run`, so migration is mechanical |
| **B6 turns today's silent warnings into generation failures** — existing specs with `recRef`/`repeated` fields start failing | High | Default `cli_strict` to `false` for specs on the current schema version; opt in on the new schema version. Emit a loud, actionable warning either way |
| **A1's package change breaks anyone who built on the `main` package** | Medium | Only applies when the new gen option is set; unset reproduces today's `package main` output byte-for-byte |
| **JSON schema rejects unknown properties**, so specs using new fields fail on older toolkits | Medium | Bump the dated schema URL; document the minimum toolkit version. Old specs keep validating against old schema |
| **Flattened `recRef` flag names collide** with a sibling scalar field | Medium | Detect at generation time and fail with both field paths named; `alias` provides the manual override |
| **Depth-1 flattening is insufficient** for deeply nested records | Medium | Accepted by D2 — `--<field>-json` / `--<field>-file` are always generated as the complete escape hatch |
| **Scope**: this touches templates, spec, schema, and a second repo | Medium | Wave sequencing below; each wave independently shippable and useful |
| **`GetCommandRunner()` stays in the API package** and adopters keep using it | Low | Intentional. A2 makes injection *possible*, not mandatory; the default path is unchanged |

---

## Verification

1. **Golden tests** (H5) over a fixture spec exercising every field type,
   `repeated`, `map`, `recRef` at depth 1 and 2, `oneOf`, grouping, positional
   args, and aliases. Generated output is committed and diffed.
2. **Round-trip test**: generate a CLI from the fixture spec, build it, drive
   it with a stub runner, assert flags → command struct for every type.
3. **Backward-compat test**: regenerate an unmodified existing spec with no
   new options; assert output is byte-identical to the committed baseline.
4. **Strict-mode test**: a spec with an unrepresentable field exits non-zero
   under `--strict` and exits zero with a stderr warning without it.
5. **Injection test**: a host package in a *separate module* imports the
   generated CLI and supplies its own runner — proves A1/A2 and that the
   import cycle is gone.
6. **Client-runner test** (F1): a generated client-backed runner drives every
   command against a test server; a newly added spec command is covered with
   no hand-edit.
7. **Exit code test** (E1): each documented code is asserted.
8. **Output test** (D2): `--output json` is parsed back and compared to the
   result struct; stdout contains no diagnostics.
9. **JS tolerance test** (D4): `--lang js` over the fixture spec succeeds and
   its output is unchanged by the new CLI metadata.
10. **Completeness test** (H2): removing a command's wiring fails the build.
11. `go test ./...` in `toolkit/` and `ugo/`; `go vet` clean.

---

## Rollout

Four waves. Each is independently shippable and independently useful.

### Wave 1 — Make it usable (P0 core)

`ugo`: U1, U2, U3. `toolkit`: A1, A2, A3, B6, D1, E1, E2, E3, plus
`FieldDef.Tags`.

*After this wave a generated CLI is embeddable, prints results, exits
correctly, and refuses to silently drop a field.* This is the wave that
removes the need to hand-copy the command tree.

### Wave 2 — Make it complete

`toolkit`: B1–B5, F1, F2, H1, H2, H3. `ugo`: U4, U6.

*After this wave no spec construct is unrepresentable, and a daemon-backed
CLI needs no glue.*

### Wave 3 — Make it good

`toolkit`: C1–C7, D3–D5, A4, A5, A6, H4, H5. `ugo`: U5, U8.

### Wave 4 — Polish

`toolkit`: G1, G2, I1, I2. `ugo`: U7.

**Migration:** no existing spec changes behaviour until it opts in via a new
`cli_*` gen option or the new schema version. The one deliberate exception is
U2 in `ugo`, which is a source-breaking change for direct `ugo/cli` consumers
and is why it is confined to a single minor with a deprecation path.

---

## Decision Log

**D1 — Both local and client-backed runners, one template.**
Considered generating two CLI variants (a local one and a client one).
Rejected: it doubles the template surface and guarantees drift between them.
Since `PkgCommandRunner` is already the dispatch seam, the *only* difference
between local and remote is which runner is constructed. Making the runner an
argument (A2) collapses the two variants into one. F1 then supplies the remote
runner, and the host picks. This also makes the CLI trivially testable with a
stub runner.

**D2 — `recRef` flattening default-on, depth 1, with JSON/file escape
hatches.**
Considered: (a) `--x-json` only, (b) unlimited flattening, (c) depth-limited
flattening plus escape hatches. Rejected (a): forcing users to hand-write JSON
for a two-field nested config is hostile, and it is the single most common
shape — a config record on a "create"/"start" command. Rejected (b): deep
flattening produces unusable names like `--config.security.tls.cert-path` and
makes collisions near-certain. Chose (c): depth 1 covers the common case with
good ergonomics, and the JSON/file forms are always generated so nothing is
ever unreachable. The depth limit is configurable for adopters who want more.

**D3 — `ugo` minor, not a vendored fork.**
Considered forking `cli` into `toolkit/`. Rejected: mvep already depends on
`ugo`, a fork doubles the maintenance surface, and the `cli` changes (U1–U8)
are generically useful to every `ugo` consumer, not just generated CLIs.
The cost is cross-repo sequencing, which the wave plan already accounts for.

**D4 — Go-first, JS-compatible.**
Considered adding a JS CLI generator in the same plan. Rejected on scope: no
JS CLI template exists today, so this would be greenfield work on top of an
already large plan. The mitigation is cheap — keep every spec addition
language-neutral and add one test asserting the JS generator tolerates the new
metadata — so a JS CLI can be added later with no spec migration.

**D5 — Not reviving the cobra templates.**
`go_cli_cobra_main.txt`, `go_cli_cobra_root.txt` and `go_cli_cobra_add.txt`
exist but are unreferenced by `toolkit_runner.go`. Switching to cobra would
deliver U1–U8 for free but adds a heavy third-party dependency to every
generated CLI, abandons the `ugo` investment, and changes the generated help
and flag syntax for existing users. Since the `ugo/cli` gaps are individually
small and generically useful, extending `ugo` is the better trade. The dormant
cobra templates should be deleted or explicitly marked experimental as a
side-cleanup.

---

## Open Questions

1. **Does `FieldDef.Tags` land here or in the roadmap's "Required semantics"
   item?** The defect (validated then dropped) is independent of the CLI and
   affects any codegen that would want required-ness. *Recommendation:* land
   the minimal `Tags []string` fix here as T1 since it blocks C4, and let the
   roadmap item promote it to a first-class `Required bool` later.

2. **Flat `cli_*` gen option keys, or widen `GenOptsDef` to a typed struct?**
   *Recommendation:* flat keys now (zero type churn, consistent with existing
   `go_package` / `format`), revisit widening if the option count grows past
   roughly ten.

3. **Should `--output table` be the default, or `--output text`?** A generated
   table needs D3 hints to be good; without hints it degrades to key-value.
   *Recommendation:* default to a human `text` renderer that uses table
   formatting when hints exist and key-value otherwise, so D3 is a
   progressive enhancement rather than a gate on D1.

4. **How much of A5's hook surface is really needed?** `WithPreRun` is
   clearly required for connection setup. `WithPostRun` and
   `WithErrorMapper` are speculative. *Recommendation:* ship `WithPreRun` and
   `WithErrorMapper` in Wave 3, defer `WithPostRun` until an adopter asks.

5. **Does the completeness assertion (H2) belong in generated code or in
   `mvep validate`?** Generated code catches it at build time in the adopter's
   CI; `validate` catches it earlier but only if run. *Recommendation:* both,
   sharing one metadata source.

---

## Progress

- [ ] T1 — `FieldDef.Tags` unmarshal fix
- [ ] T2 — `ugo/cli`: persistent flags + `Run` error propagation (U1, U2)
- [ ] T3 — `ugo/cli`: additional flag types (U3)
- [ ] T4 — CLI template golden-test harness
- [ ] T5 — Library package + injectable runner (A1, A2, A3)
- [ ] T6 — Result rendering and exit codes (D1, D2, E1–E3)
- [ ] T7 — Strict mode and loud diagnostics (B6, H3)
- [ ] T8 — Full scalar and repeated type coverage (B1, B2)
- [ ] T9 — `recRef`, `map`, `oneOf` coverage (B3, B4, B5)
- [ ] T10 — Client-backed runner generation (F1, F2)
- [ ] T11 — Granular skip and completeness assertion (H1, H2)
- [ ] T12 — Naming policy, grouping, positional args (C1, C2, C3)
- [ ] T13 — Help metadata, shorthands, hidden/deprecated (C5, C6, C7)
- [ ] T14 — Write-once entrypoint and lifecycle hooks (A4, A5, A6)
- [ ] T15 — Documentation and roadmap reconciliation

---

## Tasks

Test-first throughout: every task begins with a failing test or a golden
fixture that captures the desired output before the template or struct change.

### T1 — `FieldDef.Tags` unmarshal fix

**Outcome:** `tags` from the spec reaches codegen instead of being dropped.

**Verification:** a test unmarshals a spec with `tags: ["required"]` and
asserts the parsed `FieldDef` carries it. No generated output changes yet.

**Notes:** Pure defect fix, independently mergeable. Coordinate with the
roadmap's *Required semantics* item so the two do not conflict.

### T2 — `ugo/cli`: persistent flags and error propagation

**Outcome:** U1 and U2 in `github.com/mainvec/ugo`.

**Verification:** a parent command declares a persistent flag, a nested child
reads it; a failing handler's error surfaces from `Framework.Run`.

**Notes:** Companion repo. Additive `RunE`-style field first, `Run` deprecated,
so existing consumers migrate mechanically. Blocks Wave 1 in `toolkit`.

### T3 — `ugo/cli`: additional flag types

**Outcome:** U3 — `Int64Var`, `Float64Var`, `DurationVar`, `StringSliceVar`,
`BytesVar`.

**Verification:** parse tests per type, including the documented `timestamp`,
`duration` and `bytes` input formats.

### T4 — CLI template golden-test harness

**Outcome:** H5 — a fixture spec exercising every construct, with committed
generated output.

**Verification:** the harness fails on any unintended template diff. Baseline
captures **current** output first, so subsequent tasks show reviewable diffs.

**Notes:** Do this before touching templates. It is what makes every later task
reviewable.

### T5 — Library package and injectable runner

**Outcome:** A1, A2, A3.

**Verification:** a host package in a separate module imports the generated
CLI and injects a stub runner; with no new gen option set, output is
byte-identical to the T4 baseline.

**Notes:** The keystone task — it is what removes the import cycle that forces
adopters to hand-copy the command tree.

### T6 — Result rendering and exit codes

**Outcome:** D1, D2, E1, E2, E3.

**Verification:** `--output json` round-trips to the result struct; stdout
carries no diagnostics; each documented exit code is asserted.

**Notes:** Depends on T2 for E1/E2.

### T7 — Strict mode and loud diagnostics

**Outcome:** B6, H3.

**Verification:** an unrepresentable field exits non-zero under `--strict` and
warns to stderr without it.

**Notes:** Land before T8/T9 so those tasks shrink the set of fields that trip
it, rather than the reverse.

### T8 — Scalar and repeated type coverage

**Outcome:** B1, B2. Depends on T3.

**Verification:** golden diff plus per-type parse tests.

### T9 — `recRef`, `map` and `oneOf` coverage

**Outcome:** B3, B4, B5.

**Verification:** flattened flags, `--x-json` and `--x-file` all produce the
same command struct; documented precedence is asserted; name collisions fail
generation; two `oneOf` members is a usage error.

**Notes:** Implements D2. Largest single task — consider splitting `recRef`
from `map`/`oneOf` if it grows.

### T10 — Client-backed runner generation

**Outcome:** F1, F2.

**Verification:** the generated runner drives every command against a test
server; adding a command to the fixture spec covers it with no hand-edit.

**Notes:** Together with T5 this is the pair that makes daemon-backed CLIs
zero-glue.

### T11 — Granular skip and completeness assertion

**Outcome:** H1, H2, A6.

**Verification:** removing a command's wiring fails the build; per-artifact
skip works; plain `--skip-cmd` retains its current meaning.

### T12 — Naming policy, grouping, positional args

**Outcome:** C1, C2, C3. Requires the schema bump.

**Verification:** golden fixtures for each flag/command style; a grouped spec
produces nested subcommands; a positional field wires the right validator.

### T13 — Help metadata, shorthands, hidden and deprecated

**Outcome:** C5, C6, C7. C4 depends on T1 plus U4.

**Verification:** `--help` golden output; shorthand collision fails
generation; a deprecated command warns on use.

### T14 — Write-once entrypoint and lifecycle hooks

**Outcome:** A4, A5.

**Verification:** regenerating over a hand-edited entrypoint leaves it
untouched with no `NOMVEP` marker; a `WithPreRun` hook supplies the runner.

### T15 — Documentation and roadmap reconciliation

**Outcome:** `MVEP_SKILL.md` documents the new options and the CLI
representation of every field type. `MVEP_ROADMAP.md` marks absorbed rows.
`CHANGELOG.md` entries for both repos. Dormant cobra templates deleted or
marked experimental (D5).

**Verification:** a fresh adopter can wire a daemon-backed CLI from the docs
alone, without reading the templates.
