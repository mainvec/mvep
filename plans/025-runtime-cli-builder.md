# Plan 025 — Runtime CLI Builder

**Issue:** [#25](https://github.com/mainvec/mvep/issues/25) — feat: runtime CLI builder — build CLIs from a spec instead of generating them
**Requirements catalogue:** [#23](https://github.com/mainvec/mvep/issues/23) / [plans/023-cli-generation-complete-reusable.md](023-cli-generation-complete-reusable.md)

## Problem / Goal

The toolkit generates a CLI command tree into `cmd/<srv>/<srv>_main_cmd.go`. That output is
unusable in three independent ways, catalogued in plan 023:

1. **Incomplete.** Only `string`, `boolean`, and `int32` become flags. `recRef`, `map`,
   `repeated`, and nine other scalar types are silently dropped. A dropped *required* field
   makes the command uninvokable, with only a comment in the generated source to show for it.
2. **Not reusable.** It is emitted as `package main` with `commandRunner = <pkg>.GetCommandRunner()`
   hardcoded. A CLI that talks to a daemon rather than running commands in-process creates an
   import cycle, so adopters hand-copy the whole tree into a separate module where it drifts.
3. **Unscriptable.** `_, err := commandRunner.RunXxxCmd(...)` discards the result and never sets
   an exit code.

Closing those gaps inside Go text templates means encoding flag parsing, nested-record
flattening, map syntax, required-ness, and result rendering as template logic — the hardest
possible place to write, test, and extend it.

**Goal:** build the CLI at runtime from the parsed spec, as an ordinary Go library that
implementers import, configure, and extend.

## Goals

- A `runtime/go/mvep/cli` library that turns a parsed spec plus an `mvep.Package` into a working
  CLI with no code generation.
- Complete coverage of every spec field type, with unsupported constructs failing loudly at
  construction rather than vanishing.
- One CLI that drives either a local `CommandRunner` or a remote `client.PackageClient`.
- Implementer extension points: persistent global flags, hand-written subcommands, per-command
  hooks, result renderers.
- Correct exit codes, so MVEP CLIs are scriptable.
- `mvep` itself rebuilt on the library from its own embedded spec — self-hosting is the proof.
- `runtime/go` gains **zero** new dependencies.

## Non-goals

- Async job submission from the CLI (`--async` + `GetJobStatus` polling). Deferred; the runtime
  support exists but CLI ergonomics are a separate design.
- Shell completion (`ugo/cli` U7).
- TypeScript parity in `runtime/ts`.
- `oneOf` mutually-exclusive flag groups (023 B5) — the spec has no `oneOf` construct yet.
- Removing the existing codegen path. It is preserved behind `--cli=legacy`.

## Proposed Design

### Enabling constraint: import direction

`toolkit` already depends on `runtime/go`; the reverse is forbidden and stays forbidden. That
means the spec model can move **into** the runtime and be imported by toolkit, with no cycle.
This is what makes a runtime-parsed spec viable at all.

### Layering the parser and validator

Toolkit's spec handling is three layers with very different dependency costs. Only the
dependency-free ones move:

| Layer | Symbols | Dependencies | Destination |
| --- | --- | --- | --- |
| 1. Parse | `readAndRemoveJSONComments`, `removeCStyleComments`, `removeCppStyleComments` | none | move to runtime |
| 2. Semantic validation | `validateSrvDef`, `validateRecords`, `validateRecord`, `validateCommands`, `validateCommand`, `validateFieldDefs`, `ValidationResult`, `DefaultValidationResult` | none | move to runtime |
| 3. JSON Schema validation | `validateJSONSchemaContent`, `addResource`, `HTTPURLLoader`, `newHTTPURLLoader`, `supportedSchemaResources`, `jsonValidationResult` | `santhosh-tekuri/jsonschema/v6`, `//go:embed resources`, live `http.Client` | **stays in toolkit** |

Layer 3 is excluded from the runtime deliberately. `newHTTPURLLoader` registers `http`/`https`
loaders and only four `$schema` URLs are pre-registered — anything else is fetched over the
network with a 15s timeout. Putting a data-driven outbound request into the startup path of
every MVEP-built CLI buys nothing: the embedded spec was already schema-validated at
`mvep generate` time by the same toolchain that emitted the structs the CLI decodes into.
Schema errors are actionable by spec *authors* at generate time, not by CLI *end users*.

Layer 2 is included because the CLI genuinely needs it: `recRef` flag flattening dereferences
`#/recordsDefs/X`, and `validateFieldDefs` is exactly the check that the target resolves.
Without it a spec typo becomes a panic during flag registration.

Toolkit keeps every current name as a type alias (`type SrvDef = spec.SrvDef`), so no consumer
breaks, and `BuildSrvDefFromJSON` becomes *schema-validate, then delegate to `spec.Parse`*.
Toolkit remains the strict gate.

### Binding flags to commands without reflection

`PackageHandler.executeCmd` already establishes the pattern: `Package.InstanceOf(name)` returns
a zero-valued typed command struct, the encoder decodes a payload into it, and `CommandRunner`
runs it. The CLI reuses that exact path:

```
flags -> map[string]any (keyed by spec field name) -> JSON -> enc.Decode(payload, InstanceOf(cmd)) -> Executor.Run
```

This means **no reflection anywhere**, and it works identically for plain and protobuf struct
formats because both carry `json` tags. It also makes `--<field>-json` an escape hatch that
composes naturally: the raw JSON is merged into the same map.

`InstanceOf` also returns `*XxxCmdResult` types; the spec's `Commands` list is what decides
which names become subcommands, so results are filtered out for free.

### Swappable executor

`mvep.CommandRunner.RunCmd(ctx, any) (any, error)` and
`client.PackageClient.SendCmd(ctx, any) (any, error)` already have identical shapes. A
one-method `Executor` interface unifies them, so the same CLI binary can run commands in-process
or against a daemon by construction-time choice.

### Renderer seam

The spec walk produces a parser-agnostic intermediate model (`commandNode`, `flagSpec`), which a
renderer translates into the actual CLI framework. `ugo/cli` is the intended renderer, but it has
eight gaps (023 U1–U8). The seam lets a minimal stdlib-`flag` renderer unblock all downstream
work until `ugo` v0.7.0 lands.

## Affected Modules

- **`runtime/go`** — new `mvep/spec` and `mvep/cli` packages. Minor version bump. Public API
  surface grows; nothing existing changes.
- **`toolkit`** — spec model becomes aliases; `ExecuteGenerate` gains a `--cli` mode; dormant
  cobra templates removed; `mvepapi` CLI rebuilt on the library. Minor version bump.
- **`github.com/mainvec/ugo`** (external) — `cli` package gains U1–U6 and U8.

## Risks and Compatibility

| Risk | Mitigation |
| --- | --- |
| **Spec/struct drift.** An embedded spec can name a field the compiled struct lacks — a runtime error where codegen would have given a compile error. | `mvep generate` is the gate: structs and spec come from the same file in the same run. If this bites, a later "bake the spec into a Go constant" mode makes it a build-time concern again, and costs ~10 lines because the constant's type is `spec.SrvDef`. |
| **JSON round-trip fidelity.** `bytes`, `timestamp`, `duration`, `uuid`, and `int64` have different JSON encodings across plain vs protojson (notably int64-as-string). The CLI's emitted JSON must decode correctly under whichever encoder the package uses. | Encode through the package's own `oenc` encoding rather than raw `encoding/json`. Test explicitly against `runtime/go/test/api` (iunet, protobuf format) as well as a plain-format package. |
| **Moving the spec model is API-visible for toolkit consumers.** | Type aliases preserve every name. The existing toolkit test suite is the regression gate. |
| **`ugo/cli` gaps block delivery.** | Renderer seam plus a stdlib-`flag` fallback; `ugo` work proceeds in parallel. |
| **Help output nondeterminism.** Commands and fields live in ordered maps; iteration order drives flag and command ordering in help text. | Reuse the existing `SortFieldsByFnum` ordering discipline; assert ordering in golden help-output tests. |
| **Runtime dependency creep.** A careless import in `spec` or `cli` pulls jsonschema or a CLI framework into every consumer's module graph. | `go list -m all` assertion in CI; `RemoteExecutor` isolated so `cli` does not force a `client` dependency. |

Backward compatibility: additive throughout. `--cli=legacy` reproduces today's generated output
byte for byte. Existing generated CLIs keep compiling against the unchanged runtime API.

## Progress

- [ ] T1 — `runtime/go/mvep/spec`: model types and parser
- [ ] T2 — Move semantic validation into `spec`
- [ ] T3 — Toolkit delegates to `spec` via type aliases
- [ ] T4 — `Executor` interface and local/remote adapters
- [ ] T5 — Spec-to-CLI intermediate model and renderer seam
- [ ] T6 — Flag binding pipeline and `App.Run`
- [ ] T7 — Scalar and repeated flag coverage
- [ ] T8 — Map and `recRef` flag coverage
- [ ] T9 — Strict construction on unsupported constructs
- [ ] T10 — Required flags from `FieldDef.Tags`
- [ ] T11 — Global flags, custom subcommands, overrides
- [ ] T12 — Per-command pre/post hooks
- [ ] T13 — Result renderers and exit codes
- [ ] T14 — `ugo/cli` U1–U6 and U8 (external)
- [ ] T15 — Toolkit `--cli=runtime|legacy|none`
- [ ] T16 — Dogfood: rebuild `mvep`'s own CLI on the library
- [ ] T17 — Documentation

## Tasks

### T1 — `runtime/go/mvep/spec`: model types and parser

**Outcome:** A new `runtime/go/mvep/spec` package holds `SrvDef`, `CommandDef`, `RecordDef`,
`FieldDef`, `FieldDataType` with its constants, and the four ordered-map types with their `Get`
methods. Comment stripping moves across. Public API: `Parse(io.Reader) (*SrvDef, error)`,
`ParseFile(string) (*SrvDef, error)`, `MustParseBytes([]byte) *SrvDef`.

**Verification:** `cd runtime/go && go test ./mvep/spec/...` over copies of the toolkit fixtures
covering comments, maps, refs, and results. `go list -m all` shows no new modules.

**Notes:** `removeCppStyleComments` is deliberately scanner-based rather than regex — the
original author noted a regex breaks on `http://`. Move it verbatim; do not "simplify" it.

### T2 — Move semantic validation into `spec`

**Outcome:** `validateSrvDef` and its helpers, plus the `ValidationResult` interface and
`DefaultValidationResult`, live in `spec`. `Parse` runs them after unmarshal. `jsonValidationResult`
stays in toolkit, implementing the moved interface.

**Verification:** A spec with `$ref: "#/recordsDefs/Missing"` fails `spec.Parse` with a message
naming the field and the missing record.

**Notes:** The `ValidationResult` interface is dependency-free, which is what makes the split
clean — toolkit's jsonschema-backed implementation satisfies the moved interface unchanged.

### T3 — Toolkit delegates to `spec` via type aliases

**Outcome:** `toolkit` declares `type SrvDef = spec.SrvDef` (and the rest), keeps JSON Schema
validation and `//go:embed resources`, and reimplements `BuildSrvDefFromJSON` as schema-validate
then `spec.Parse`.

**Verification:** `cd toolkit && go test ./...` fully green with no test edits — in particular
`03_basic_wo_invalid.jsonc` must still fail schema validation, proving generate-time strictness
did not weaken.

**Notes:** Requires a `runtime/go` version bump in `toolkit/go.mod`. Under `go.work` this
develops locally against the workspace copy; the bump matters at release time.

### T4 — `Executor` interface and local/remote adapters

**Outcome:** `type Executor interface { Run(ctx context.Context, cmd any) (any, error) }` with a
local adapter over `mvep.CommandRunner`. The remote adapter over `client.PackageClient` lives in
a subpackage so `mvep/cli` does not drag `mvep/client` into consumers that only run locally.

**Verification:** One table test runs the same command through both adapters against an
`httptest` server and asserts identical results.

**Notes:** Confirms the design claim that local and remote have identical signatures today.

### T5 — Spec-to-CLI intermediate model and renderer seam

**Outcome:** The spec walk produces `commandNode`/`flagSpec` values. A `renderer` interface
translates them into a concrete CLI framework; a stdlib-`flag` renderer ships first.

**Verification:** Golden test asserting the intermediate model built from
`toolkit_plain.jsonc` — three commands, correct aliases, correct flag names and help text.

**Notes:** Only names in `spec.Commands` become subcommands, which filters the `*CmdResult`
types that `InstanceOf` also returns.

### T6 — Flag binding pipeline and `App.Run`

**Outcome:** `cli.New(spec, pkg, executor, opts...)` and `App.Run(ctx, args)`. Parsed flags
become a `map[string]any` keyed by spec field name, encode through the package's encoding, and
decode into `pkg.InstanceOf(cmdName)` before execution.

**Verification:** End-to-end test: `[]string{"generate", "--in", "x.json", "--lang", "go"}`
produces a `*GenerateCmd` with those fields set and reaches the executor.

**Notes:** Mirrors `PackageHandler.executeCmd`. Keep the two implementations reading alike so
divergence is obvious.

### T7 — Scalar and repeated flag coverage

**Outcome:** `int64`, `uint32`, `sint32`, `float`, `double`, `bytes` (base64), `timestamp`
(RFC3339), `duration`, `uuid` all bind. `repeated` fields accept a repeatable flag.

**Verification:** Table test per type: argv in, expected payload JSON out, decoded struct
asserted. Run against both a plain-format and a protobuf-format package.

**Notes:** The int64-as-string protojson difference is the likeliest failure here — see the
round-trip fidelity risk.

### T8 — Map and `recRef` flag coverage

**Outcome:** `map` fields accept repeatable `--label key=value`. `recRef` fields flatten to depth
1 (`--addr-city`), with `--<field>-json` and `--<field>-file` always available as escape hatches
regardless of depth.

**Verification:** A nested-record spec fixture; assert flattened flags and both escape hatches
produce identical payloads.

**Notes:** Depth 1 covers the common case; deeper nesting is intentionally JSON-only rather than
generating unbounded flag names.

### T9 — Strict construction on unsupported constructs

**Outcome:** An unknown field type, an unresolvable `$ref`, or a flag-name collision fails
`cli.New` with a precise error. Nothing is ever silently dropped.

**Verification:** Negative table test per failure mode asserting the error message names the
offending command and field.

**Notes:** This is the single most important behavioural difference from the generated CLI,
where an unsupported type became a comment and an exit code of 0.

### T10 — Required flags from `FieldDef.Tags`

**Outcome:** A field tagged `required` produces a required flag; omitting it fails before the
executor runs.

**Verification:** Missing-required-flag test asserts a non-zero exit and no executor invocation.

**Notes:** Depends on the `FieldDef.Tags` fix in PR #24 (branch `feat/23-cli-generation-complete-reusable`).
Rebase once that merges. Also depends on `ugo` U4 if the ugo renderer is active; the stdlib
renderer enforces it directly.

### T11 — Global flags, custom subcommands, overrides

**Outcome:** `WithGlobalFlags` for persistent flags such as `--server`, `--token`, `--output`;
`WithCommand` to add hand-written subcommands beside spec-derived ones; `WithOverride` to
rename or hide a generated command or flag.

**Verification:** A test CLI adds a `version` subcommand and a `--server` global flag, and
asserts both appear in help and parse correctly on every subcommand.

**Notes:** Persistent flags need `ugo` U1; the stdlib renderer handles them by manual propagation.

### T12 — Per-command pre/post hooks

**Outcome:** `WithHook(cmdName, ...)` supplying `PreRun(ctx, cmd any) error` for extra validation
and `PostRun(ctx, cmd any, result any) error` for custom rendering.

**Verification:** A `PreRun` returning an error aborts before the executor; a `PostRun` observes
the decoded result value.

**Notes:** Hooks receive the typed struct, so implementers can type-assert to their own command
types and stay type-safe.

### T13 — Result renderers and exit codes

**Outcome:** `json` (default), `table`, and `quiet` renderers selected by `--output`. Command
errors map `mvep.ErrorInfo.Code` to a process exit code.

**Verification:** `mvep validate --in <invalid>` exits non-zero; `--output json` emits parseable
JSON on stdout with diagnostics on stderr.

**Notes:** Reuse the taxonomy behind `HTTPStatusForErrorCode` rather than inventing a second
error classification. Needs `ugo` U2 (`Run` returning an error) when the ugo renderer is active.

### T14 — `ugo/cli` U1–U6 and U8 (external)

**Outcome:** `github.com/mainvec/ugo` `cli` package gains persistent flags (U1), `Run` returning
an error (U2), `Int64`/`Float64`/`Duration`/`StringSlice`/`Bytes` var helpers (U3), required-flag
enforcement (U4), short/long flag pairing (U5), mutually-exclusive groups (U6), and help
categories (U8). Released as a minor version.

**Verification:** The ugo renderer passes the same test suite as the stdlib renderer.

**Notes:** External repository, tracked here because it gates the production renderer. U7
(completion) is out of scope. Shipped as a new minor rather than a vendored fork — the changes
are generically useful to all ugo consumers.

### T15 — Toolkit `--cli=runtime|legacy|none`

**Outcome:** `ExecuteGenerate` takes a CLI mode. `runtime` (default) emits no command tree,
`legacy` reproduces today's output, `none` replaces the old `skipCmd` boolean. The dormant
`go_cli_cobra_*.txt` templates are deleted.

**Verification:** `mvep generate --cli=legacy` output is byte-identical to the pre-change
generator on every testdata fixture. `--cli=runtime` emits no `cmd/` tree.

**Notes:** `skipCmd` is a positional boolean in the current signature; folding it into the mode
enum removes a boolean-parameter trap.

### T16 — Dogfood: rebuild `mvep`'s own CLI on the library

**Outcome:** `toolkit/mvepapi/cmd/mvep/mvep_main_cmd.go` is rewritten to embed
`gengen/toolkit_plain.jsonc` and drive it through `cli.New`, replacing the generated tree.

**Verification:** `mvep init`, `mvep generate`, and `mvep validate` behave identically to the
current binary across the full `testdata` set, including exit codes. Offline check: with the
network down, `mvep --help` issues no HTTP request.

**Notes:** The file already carries `// NOMVGEN`, so it is hand-owned and safe to rewrite. This
task is the real acceptance test for the whole plan.

### T17 — Documentation

**Outcome:** `runtime/go/README.md` documents the CLI library; `toolkit/MVEP_SKILL.md` and
`AGENT.md` describe the `--cli` modes and the runtime-builder approach; CHANGELOG entries for
both modules.

**Verification:** Documented snippets compile as an example test.

**Notes:** Follow `/memories/repo/docs-sync.md` for which docs must move together.

## Rollout

1. T1–T3 land first and are independently releasable: they are a pure refactor with the existing
   toolkit suite as the gate. Ship as `runtime/go` and `toolkit` minors.
2. T4–T13 build the library behind no flag — it is new API, unreachable until someone calls it.
3. T14 proceeds in parallel in `ugo`; the stdlib renderer means it never blocks.
4. T15 flips the toolkit default to `--cli=runtime`. Adopters who want the old output add
   `--cli=legacy`; nothing they already generated stops compiling.
5. T16 is the migration proof, done on the maintainer's own CLI before asking anyone else.

Rollback: `--cli=legacy` restores the previous generator output. The new packages are additive
and can be left unused.

## Decision Log

**Spec at runtime over a generated descriptor.** A generated descriptor would be structurally
identical to the spec model. Relocating the model makes the descriptor a ~10-line template later
if drift ever justifies it, and both feed the same `cli.New`. Building the descriptor first would
mean maintaining a parallel type system.

**Parse and semantic validation move; JSON Schema validation does not.** Layers 1 and 2 are
dependency-free and the CLI needs `$ref` resolution for nested-record flags. Layer 3 pulls
`santhosh-tekuri/jsonschema/v6`, the embedded schema tree, and an `http.Client` that fetches
unrecognised `$schema` URLs — startup latency, an offline failure mode, and data-driven network
egress in every MVEP CLI, to re-check something that could not have changed since generation.

**No reflection.** Reusing the encoder decode path that `PackageHandler.executeCmd` already
relies on gives plain/protobuf format independence for free and keeps `--<field>-json` a natural
composition rather than a special case.

**`ugo/cli` as the renderer, behind a seam.** It is already a runtime dependency and the U1–U8
work benefits all ugo consumers. The seam exists because those gaps would otherwise serialise
this plan behind an external release.

**023 stays open.** It is the requirements catalogue — *what* a usable CLI must cover. This plan
is the mechanism. Keeping them separate means the type-coverage checklist survives independently
of how it gets delivered.

**Legacy codegen preserved.** Deleting it would strand adopters mid-migration for no benefit;
the templates are inert once the default flips.
