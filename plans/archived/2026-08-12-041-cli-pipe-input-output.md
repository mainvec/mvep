# 041: CLI payload reachability — `mvep` namespace, pipes, and nested-field parity

**GitHub Issue**: [#49](https://github.com/mainvec/mvep/issues/49)
  (`feat(cli): mvep namespace, payload pipes, and nested-field flag parity`)

- Branch: `feat/049-cli-pipe-input-output`
- Supersedes: plan 042 (`fix(cli): bind repeated record sub-fields as repeatable flags`) — folded in as **T1**
- Related: [plan 023](023-cli-generation-complete-reusable.md) items **B2**, **B3**, **B3c**, **D2**, **E5**;
  [plan 025](archived/2026-08-10-025-runtime-cli-builder.md) T9 (introduced depth-1 flattening);
  [plan 040](archived/2026-08-11-040-cli-command-groups.md) (group machinery this reuses)
- Target: `runtime/go` patch for T1, then `runtime/go` minor for T2–T10 (+ `toolkit/go.mod` bumps per release discipline)

## Progress

- [x] T1: Repeated record sub-fields bind correctly (patch release, unblocks zirafa)
- [x] T2: Reserved `mvep` namespace and command index
- [x] T3: Input sources — file, explicit stdin, implicit pipe
- [x] T4: `mvep exec` payload dispatch
- [x] T5: Built-in JSON renderer and `--mvep-output`
- [x] T6: `mvep send` streaming envelope pipe
- [x] T7: `mvep list` and `mvep describe`
- [x] T8: Per-field `-file` hatch at both nesting levels
- [x] T9: Toolkit reserved-name validation
- [x] T10: Documentation and release

## Problem / Goal

A command payload that is nested, repeated, or large is not reachable from the
CLI. That has two faces, and they are the same defect seen from opposite sides.

**Flag path.** Depth-1 `recRef` flattening drops `Repeated`.
`registerSubFieldFlag` (`runtime/go/mvep/cli/flags.go:304`) type-switches on
`rf.Type` alone, so a `repeated string` sub-field falls into the `FieldString`
case, binds with `StringVar`, and is encoded as a JSON **string**. Unmarshalling
that into the record struct fails:

```
Error: --runtimeConfig: json: cannot unmarshal string into Go struct field
RuntimeConfig.argsTemplate of type []string
```

The top-level path is correct — `registerFlag` checks `f.Repeated` first and
delegates to `registerRepeatedFlag`. Only the sub-field path lacks the branch,
so the two disagree for the same field type. The effect is that **any record
containing a repeated field is unreachable from the CLI**, with no partial
workaround: the flag exists, accepts a value, and fails at apply time.

**Payload path.** There is no way to supply a whole command payload at all — no
file, no pipe — and no machine-readable output, so nothing round-trips. Complex
requests are addressable only through `--<field>-json` inline JSON: painful to
type, impossible for large payloads, hostile to scripts.

**Unifying invariant — parity.** Every (`FieldType` × `Repeated` × depth)
combination must be reachable, and top-level and sub-field binding must agree.
That invariant is why these are one plan: shipped separately, the payload path
adds `--<field>-file` at top level only and re-opens the exact top-level /
sub-field drift the flag fix just closed.

### Downstream report

Found while adding a runtime to `zirafa` (`zirafa runtime register`), whose
`RuntimeConfig` record carries `argsTemplate []string` and
`paramsSchema []*RuntimeParamSchema`. Both are unusable; registering a runtime
that takes more than one argument requires hand-writing
`<state_dir>/runtimes/<id>.json` and restarting the daemon. Tracked on the
zirafa side in `plans/028-improvements-from-field-report.md`, blocked on the
`runtime/go` release carrying **T1**.

## Goals

- Repeated **string** sub-fields bind as repeatable flags:
  `--runtimeConfig-argsTemplate '--model' --runtimeConfig-argsTemplate '{{model_path}}'`.
- Repeated **non-string** sub-fields (including `recRef`) bind as a JSON array
  via `--<record>-<field>-json`, matching the top-level fallback.
- Sub-field and top-level binding agree on every `FieldType`.
- A reserved `mvep` namespace on every generated CLI provides a uniform,
  spec-independent machine surface: `exec`, `send`, `list`, `describe`.
- `mvep exec <command>` reads a complete payload from a file, from `-`, or
  **implicitly from a pipe** when stdin is not a terminal.
- `mvep send` reads a stream of `CmdReq` envelopes and emits a stream of
  `CmdResp` envelopes, flushing per record so it works in a live pipeline.
- `--mvep-output json|text` (default `text`) renders results machine-readably,
  namespaced so plain `--output` stays free for the implementor.
- Payload decoding uses the same encoder registry as the server, so CLI payload
  semantics are byte-identical to the wire.
- A spec that uses none of this generates today's CLI, with today's behaviour.

## Non-goals

- **YAML input.** JSON only; no new dependency.
- **Codegen-template or schema changes.** `FieldDesc` already carries `Repeated`
  for nested record fields and command structs already carry `json` tags. The
  only non-runtime change is toolkit *validation* (T9).
- **Flattening beyond depth 1** (plan 023 D2) — unchanged.
- **Repeatable flags for non-string top-level scalars** (`--count 1 --count 2`).
  Orthogonal flag ergonomics; deferred to its own plan.
- **Widening the repeatable set to `FieldUUID`/`FieldTimestamp`/`FieldDuration`.**
  Better ergonomics than `-json` for these string-shaped types, but it must
  change top level and sub-field in one commit or it re-creates the drift T1
  closes. Deferred so the T1 patch stays minimal.
- Whole-payload `--input-file` on individual generated subcommands — explicitly
  rejected, see Decision Log.
- Interactive/TUI prompting, colour output, `--mvep-output yaml|table|template`
  (plan 023 D3/D4), env-var or config-file binding (plan 023 I1/I2).

## Proposed Design

### Surface

```
# machine path
cat p.json | svc mvep exec generate            # implicit stdin
svc mvep exec --input p.json generate          # flags precede the command name
svc mvep exec --input - generate
cat reqs.ndjson | svc mvep send                # CmdReq stream -> CmdResp stream
svc mvep list
svc mvep describe [command]

# human path
svc generate --record-sub a --record-sub b     # repeated string sub-field
svc generate --record-sub-json '[1,2]'         # repeated non-string sub-field
svc generate --record-file nested.json         # per-field file hatch
--mvep-output json|text                        # root persistent flag
```

### Namespacing rule for flags

A flag visible on **generated** commands claims a name from the implementor's
space and must be namespaced; a flag scoped **inside** the `mvep` namespace
cannot collide and stays short. Only one flag is global, so only one is
prefixed:

| Flag | Scope | Namespaced |
|---|---|---|
| `--mvep-output` | root persistent — visible on every command | yes |
| `--input`, `--fail-fast` | `mvep exec` / `mvep send` only | no |
| `--<field>-file`, `--<field>-json` | generated commands, derived from spec field names | n/a — already spec-owned |

The prefix follows the configured namespace, so overriding it to `acme` yields
`--acme-output`. This is the same reasoning that produced the group namespace: a
persistent root flag is as much a claim on the implementor's name space as a
top-level command is.

### Reserved namespace, not reserved verbs

Framework verbs live under a single reserved group, `svc mvep <verb>`, rather
than as top-level names or a `mvep-` prefix. This reserves **one** identifier
instead of N, makes future verbs free to add with no new reservation and no
breaking change, reuses plan 040's `groupFor()` machinery, and gives
`svc mvep --help` as a self-documenting home for the framework surface.
Ergonomics barely matter here: this path is typed by scripts, not humans.

There is precedent for namespacing framework-level wire concerns — the HTTP
transport already uses `x-mainvec-cmd`
(`runtime/go/mvep/cli/cliclient/remote_executor.go:26`), and the protocol
already reserves `SubmitJob`/`GetJobStatus` (`runtime/go/mvep/job.go:33`).

The namespace name defaults to `mvep` and is overridable via a new variadic
`opts ...Option` parameter on `cli.New` (non-breaking).

### Dispatch reuses the server seam

`PackageHandler.executeCmd()` (`runtime/go/mvep/mvepackge.go:138-195`) already
does `InstanceOf(name)` → `enc.Decode(payload, cmd)` → `RunCmd`. The CLI
dispatcher mirrors it rather than inventing a second way to build the same
struct.

`runCommand` (`runtime/go/mvep/cli/app.go:229-275`) splits at its natural seam:

```
flag path:     cmdDesc.New() -> applyBindings
payload path:  cmdDesc.New() -> decode
shared tail:   checkRequired -> preHooks -> executor.Run -> postHooks -> renderer
```

Both paths **must** use the identical tail, so hooks, required-checking, and
rendering cannot drift between them. The payload half is extracted as
`dispatch(ctx, name string, payload []byte) (any, error)`; `send` (T6) is that
function in a loop.

Decoding goes through `oenc.LookupEncoding("application/json")` — the registry
the server uses — not hardcoded `encoding/json`. This keeps CLI and wire
semantics identical and makes pb3 correct: protojson handles enums, `oneof`, and
well-known types that `encoding/json` silently mangles.

### Payload key validation

`oenc` cannot reject unknown fields, so strictness is enforced by a separate
pre-validation pass rather than by the decoder. Silently dropping a typo'd key
is the worst failure mode for the scripting audience this serves.

Before decoding, the payload is unmarshalled into `map[string]json.RawMessage`
and its keys are walked against the descriptor:

- top-level keys against `CommandDesc.Fields[].Name`;
- a field carrying a `Ref` recurses into `desc.Record(ref).Fields`, including
  each element of a repeated record;
- **map values** are entered but map **keys** are not validated — they are
  arbitrary by definition;
- an unresolved `Ref` stops the walk for that subtree rather than rejecting it.

Validating only top-level keys was rejected: it would miss sub-field typos,
which is precisely the nested case this plan exists to serve.

Key comparison **normalises case and underscores** on both sides. This is
required for correctness, not leniency: protojson accepts both the proto
`snake_case` name and its `lowerCamelCase` form, and `encoding/json` already
falls back to case-insensitive field matching. Comparing against
`FieldDesc.Name` verbatim would reject payloads the decoder itself accepts,
breaking pb3 packages. Normalising still catches every genuine typo.

The cost is a second parse of each payload. That is negligible for `exec` and
acceptable per record for `send`; if streaming throughput ever matters,
validation is the part to make opt-out.

### Pipes

`exec` resolves its payload from, in order: `--input <path>`, `--input -`, or —
when `--input` is absent and `os.Stdin.Stat()` reports stdin is not a
`ModeCharDevice` — stdin implicitly. Stdlib only, no dependency. Empty input is
an empty payload (`{}`), not a parse error; `checkRequired` then applies
normally. Two flags claiming `-` in one invocation is an error.

`send` decodes with a `json.Decoder` loop, which handles NDJSON **and**
concatenated objects in one code path with no format sniffing. Each record is
dispatched and its `CmdResp` written and flushed immediately. Default is
continue-on-error, emitting `CmdResp.Error` — mirroring the wire — with
`--fail-fast` to stop at the first failure; the process exits non-zero if any
record errored.

`Executor.Run(ctx, cmd)` (`runtime/go/mvep/cli/executor.go:15`) takes no
headers, but no new context key is needed: `envelope.go:135` already exports
`ContextWithCmdReq` / `CmdReqFromContext`, with `GetRequestHeader` and
`SetResponseHeader` convenience helpers on top. `send` puts the whole `*CmdReq`
on the context exactly as the server does, so an interceptor or hook that reads
headers behaves identically under the CLI and over HTTP.

The reverse direction comes free: `send` reads `CmdRespFromContext` after
dispatch to populate the emitted envelope's `Headers`, mirroring
`mvepackge.go:203`. Response headers therefore round-trip through the CLI.

### Error output

Under `--mvep-output json`, errors serialize as JSON on **stdout**, using
`mvep.ErrorInfo` (`code`, `message`) wrapped as `{"error": {...}}`. Reusing the
envelope's own error type and field name means a failed `exec` is shaped exactly
like the `CmdResp.Error` that `send` emits per record, so one consumer parses
both. Nothing is duplicated on stderr in JSON mode; `--mvep-output text` keeps
today's behaviour unchanged.

Two invariants this must not break:

- **`RunWithIO` still returns the error.** Rendering is additive, not a
  replacement. `json_error_test.go` asserts on the returned error, and the exit
  code derives from it — a script running under `set -e` must still fail.
- **Exit codes are unchanged.** Usage errors stay `2`, others non-zero. Writing
  the error to stdout changes where it is *reported*, never whether it *failed*.

`RunWithIO` (`app.go:218`) is the single funnel for every error — parse, usage,
required-flag, dispatch, and execution — so rendering happens there and covers
all of them uniformly. One wrinkle: when flag parsing itself fails,
`--mvep-output` may never have been parsed, so the output mode falls back to a
pre-scan of the raw args. Without that, exactly the errors a script is least able
to anticipate would be the ones it cannot parse.

### Sub-field binding (T1)

`registerSubFieldFlag` gains an `rf.Repeated` branch **before** its type switch,
mirroring `registerFlag`:

```go
if rf.Repeated {
    return registerRepeatedSubFieldFlag(fs, parentName, rf)
}
```

`registerRepeatedSubFieldFlag` returns the same `{field, rawVal, isSet}` triple
the caller already assembles into its JSON object, so `registerRecordFlag`
(`flags.go:215`) needs no change:

- **Repeated `FieldString`** → `StringSliceVar`; `isSet` is `len(*p) > 0`;
  `rawVal` is `json.Marshal(*p)`, yielding a JSON array.
- **Everything else repeated** (`FieldUUID`, `FieldTimestamp`, `FieldDuration`,
  `FieldBytes`, numeric, bool, map, `recRef`) → `--<record>-<field>-json`;
  `isSet` is `*p != ""`; `rawVal` passes the `json.RawMessage` through.

The split point is **`registerRepeatedFlag`, not the scalar switch**. Those are
different sets, and only one of them matters: the invariant this task exists to
restore is that top-level and sub-field binding agree for the same field type.
The scalar switch string-binds `FieldString`, `FieldUUID`, `FieldTimestamp`,
`FieldBytes` and `FieldDuration`; `registerRepeatedFlag` string-binds
**`FieldString` alone**. Mirroring the scalar switch here would leave repeated
UUID as `--ids-json '["a","b"]'` at top level but `--rec-ids a --rec-ids b` one
level down — a new drift in place of the old one.

| Repeated `FieldType` | Top level | Sub-field |
|---|---|---|
| `FieldString` | `StringSliceVar` | `StringSliceVar` |
| `FieldUUID`, `FieldTimestamp`, `FieldDuration`, `FieldBytes` | `-json` | `-json` |
| numeric, bool, map, `recRef` | `-json` | `-json` |

Malformed `-json` is validated **inside the `rawVal` closure** so the error names
the sub-field flag rather than surfacing as the parent's opaque
`--<record>: json: cannot unmarshal ...`. Validation must check the value is a
JSON **array**, not merely valid JSON: unmarshalling into `json.RawMessage`
accepts `{"a":1}`, which then fails at the parent with exactly the opaque error
this branch is meant to eliminate.

### Rejected: whole-payload `--input-file` per subcommand

The original 041 draft bound `--input-file` on every generated subcommand and
merged it with explicitly-set flags. That requires knowing which flags were set,
which is **unreachable** in ugo v0.7.0: `Run` parses a throwaway merged FlagSet
(`mergedFlags`, `ugo/cli/framework.go`) and passes only positional args to
`RunE`, so `cmd.Flags()` never has `Parse` called on it and its `actual` map is
always empty. The existing `isSet` closures (`flags.go:310-369`) are zero-value
heuristics, so `--count 0` could never override a payload value.

A namespace dispatcher has no flags to merge — the payload is the whole truth.
Set-detection, `flag.Value` wrapper machinery, reset-between-runs state leakage,
and the flag-vs-payload precedence table all disappear.

## Affected Modules

| File | Change |
|---|---|
| `runtime/go/mvep/cli/flags.go` | T1 `registerRepeatedSubFieldFlag`; T8 `-file` hatches. `applyBindings` untouched; no `isSet` gating |
| `runtime/go/mvep/cli/app.go` | Split `runCommand`; extract `dispatch`; namespace + collision guard in `New`; `--mvep-output` persistent flag. `checkRequired` reused unchanged |
| `runtime/go/mvep/cli/exec.go` | **new** — namespace group, `exec` |
| `runtime/go/mvep/cli/send.go` | **new** — streaming envelope pipe |
| `runtime/go/mvep/cli/input.go` | **new** — file / `-` / implicit-pipe resolution |
| `runtime/go/mvep/cli/describe.go` | **new** — versioned schema projection |
| `runtime/go/mvep/cli/renderer.go` | built-in JSON renderer |
| `runtime/go/mvep/cli/executor.go` | unchanged interface; headers ride the context |
| `runtime/go/mvep/envelope.go` | reference only — `CmdReq`/`CmdResp` shapes and the existing `ContextWithCmdReq` helpers |
| `runtime/go/mvep/mvepackge.go` | reference only — `executeCmd()` pattern |
| `toolkit/toolkit.go` | T9 reserved-name validation in `validateCommandGroups()` |
| `toolkit/testdata/21_reserved_namespace_collision.jsonc` | **new** fixture |
| `runtime/go/mvep/cli/README.md` | flag-form table, input sources, `--mvep-output` |
| `plans/023-cli-generation-complete-reusable.md` | reconcile B2/B3/B3c/D2/E5 |
| `toolkit/go.mod`, `CHANGELOG.md` | release discipline |

## Tasks

### T1: Repeated record sub-fields bind correctly

**Outcome**: `registerSubFieldFlag` checks `rf.Repeated` before its type switch.
Repeated `FieldString` sub-fields bind via `StringSliceVar` with `rawVal`
emitting a JSON array; **every other** repeated type registers
`--<record>-<field>-json` and passes the array through as `json.RawMessage`,
matching `registerRepeatedFlag` exactly. Malformed or non-array `-json` errors
name the sub-field flag. Shipped as its own PR and `runtime/go` **patch** tag
ahead of the rest of the plan.

**Verification**: a record with a `[]string` sub-field set from two repeated
flags unmarshals to a two-element slice (the zirafa case); repeated `int32` and
repeated `recRef` sub-fields round-trip via `-json`; malformed input names
`--<record>-<field>-json`, not the parent record flag; a **non-array** but valid
JSON value (`{"a":1}`) also errors naming the sub-field flag; a repeated
`FieldUUID` sub-field binds via `-json`, proving it agrees with top level; a
mixed record with repeated and scalar sub-fields set together assembles into one
correct struct; a record with no repeated sub-fields is behaviourally identical
to before.

**Notes**: the split must mirror `registerRepeatedFlag`, **not** the scalar
switch — see the parity table in Proposed Design. This is the only task with a
downstream blocker; do not let it queue behind T2–T10.

Review of the first implementation (commit `fix(runtime/go): bind repeated record
sub-fields correctly (#49)`) found the core branch, error wrapping, and
`registerRecordFlag` stability all correct, with three items outstanding before
the patch tag:

1. The repeated string-like set was `FieldString, FieldUUID, FieldTimestamp,
   FieldDuration`, mirroring the scalar switch. Narrow it to `FieldString` for
   exact top-level parity. Nothing regresses — repeated UUID sub-fields were
   totally broken before, so there is no working usage on either side.
2. `rawVal` validates JSON validity but not array-ness; add the array check.
3. No repeated `recRef` sub-field test — the most complex path through the new
   branch. The `t9Address` fixture carries only `tags` (`[]string`) and `scores`
   (`[]int32`).

The `CHANGELOG.md` entry and the `registerRepeatedSubFieldFlag` doc comment both
assert that sub-field and top-level binding agree on every `FieldType`; both
need correcting alongside item 1.

### T2: Reserved `mvep` namespace and command index

**Outcome**: `cli.New` builds a `map[string]*mvep.CommandDesc` index while it
already iterates commands, registers the reserved namespace group, and accepts
`opts ...Option` for overriding the namespace name. A descriptor command or
group colliding with the namespace panics at construction.

**Verification**: the namespace group appears in root help; a descriptor
declaring a `mvep` command or group panics with a message naming the collision;
the override option relocates the namespace; a spec using none of this produces
an otherwise unchanged tree (existing `groups_test` passes).

**Notes**: `New(desc, executor) *App` (`app.go:63`) returns no error, so a panic
is the only option without a breaking signature change. It matches ugo's
`AddCommand` self-parent panic and is unreachable in generated code once T9
hard-errors at generation time. Do not add `PackageDesc.Command()` — the local
index is enough.

### T3: Input sources — file, explicit stdin, implicit pipe

**Outcome**: `input.go` resolves a payload from `--input <path>`, `--input -`,
or implicitly from stdin when `--input` is absent and stdin is not a character
device. Empty input yields `{}`. A second consumer of `-` in one invocation is
a clear error.

**Verification**: each of the three sources produces the same payload for the
same bytes; empty stdin yields `{}` and then fails `checkRequired` only when a
field is required; a TTY stdin with no `--input` does not block; requesting `-`
twice errors before any read.

**Notes**: `os.Stdin.Stat()` + `os.ModeCharDevice` is stdlib — no new dependency.
Guard the not-a-char-device check against `Stat` errors, which occur on some CI
runners.

### T4: `mvep exec` payload dispatch

**Outcome**: `runCommand` splits into flag path, payload path, and a shared tail;
`dispatch(ctx, name, payload)` is extracted. `mvep exec <command>` looks up the
command, validates payload keys against the descriptor, decodes via
`oenc.LookupEncoding("application/json")`, and runs the shared tail.

**Verification**: flag path and payload path produce an **identical** command
struct and identical hook/executor invocation order for the same logical input;
`echo '{"in":"spec.json"}' | svc mvep exec generate` succeeds; an unknown
top-level key errors naming the key; an unknown key **nested inside a record**
also errors; arbitrary map keys are accepted; a `snake_case` payload against a
`lowerCamelCase` descriptor name is accepted; an unknown command name errors
listing valid names; a payload satisfying a required field passes
`checkRequired`.

**Notes**: depends on T2 and T3. The shared tail is the correctness anchor of
this plan — assert on hook ordering, not just the final result. `oenc` cannot
reject unknown fields, so strictness is a separate pre-validation pass over
`map[string]json.RawMessage`; see Payload key validation in Proposed Design. Get
the key normalisation right before writing the recursive walk — comparing
`FieldDesc.Name` verbatim would reject valid protojson payloads.

### T5: Built-in JSON renderer and `--mvep-output`

**Outcome**: a JSON renderer ships in `renderer.go`; `--mvep-output json|text`
(default `text`) registers on `Root().PersistentFlags()` and selects the renderer
in the shared tail. The flag prefix follows the configured namespace name. Under
`--mvep-output json`, errors serialize to stdout as
`{"error":{"code":...,"message":...}}` using `mvep.ErrorInfo`, rendered at
`RunWithIO` so parse, usage, required-flag, and execution errors are covered
alike. Under `--mvep-output text`, output and error behaviour are unchanged.

**Verification**: `--mvep-output json` emits parseable JSON on stdout and nothing
else; `--mvep-output text` matches today's output byte for byte; a failing
command emits a parseable `{"error":...}` on stdout, writes nothing to stderr,
**and still returns a non-nil error** from `RunWithIO` with an unchanged exit
code; a flag-parse failure combined with `--mvep-output json` still produces JSON
(exercises the arg pre-scan); an `exec` error and a `send` per-record
`CmdResp.Error` share the same shape; an implementor registering its own
`--output` coexists with no collision and keeps its own semantics —
`renderer_test.go` and `json_error_test.go` pass **unmodified**; renaming the
namespace renames the flag.

**Notes**: namespacing removes what was previously this task's main hazard.
`renderer.go:18` says the `--output` flag "is the implementor's to add" and
`renderer_test.go:116` registers one — under `--mvep-output` that statement stays
true, so no contract is reversed, no collision policy is needed, and no panic is
introduced. `json_error_test.go` asserts `RunWithIO` returns an error, which is
exactly the invariant JSON error rendering must preserve: render *and* return.

### T6: `mvep send` streaming envelope pipe

**Outcome**: `send` reads `CmdReq` records with a `json.Decoder` loop, dispatches
each via `dispatch`, and writes and flushes a `CmdResp` per record. Continue-on-
error by default with `CmdResp.Error` populated; `--fail-fast` stops at the first
failure; non-zero exit if any record errored. Each request is placed on the
context with the existing `ContextWithCmdReq`, and response headers set by a
handler are read back via `CmdRespFromContext` into the emitted envelope.

**Verification**: NDJSON and concatenated-object inputs give identical results;
a response is readable **before** the input stream closes (proves incremental
flushing, not buffer-to-EOF); one malformed record yields a `CmdResp.Error` and
the stream continues; `--fail-fast` halts; exit code is non-zero when any record
errored; a request header is readable via `mvep.GetRequestHeader` from a test
executor; a header set via `mvep.SetResponseHeader` appears on the emitted
`CmdResp`.

**Notes**: depends on T4's `dispatch`. No new context key is needed —
`envelope.go:135` already exports `ContextWithCmdReq` / `CmdReqFromContext`, and
reusing them means header-reading interceptors behave identically under the CLI
and over HTTP. `send` always emits envelopes and ignores `--mvep-output text` —
its output *is* the wire format; document that asymmetry with `exec`.

Implemented `send` line-by-line with a fresh decoder per line rather than one
long-lived `json.Decoder`. A decoder cannot advance past a `json.SyntaxError`
— it returns the same error forever — and its read-ahead makes resync via
`Buffered()` unreliable. Reading a line and decoding it fresh handles NDJSON,
concatenated objects on a line, and malformed lines with one code path and no
infinite loop. `CmdReq.Payload` is `[]byte`, so send inputs carry base64
payloads (the wire format), not raw JSON objects.

### T7: `mvep list` and `mvep describe`

**Outcome**: `mvep list` prints command names (a JSON array under
`--mvep-output json`). `mvep describe [command]` emits a minimal, versioned,
hand-written JSON projection — name, alias, group, description, fields
(name, type, repeated, required, ref), result type. No argument describes all.

**Verification**: `list` output matches the command tree; `describe` output is
stable across a synthetic `FieldDesc` field addition, proving the projection is
decoupled; `describe` of an unknown command errors.

**Notes**: do **not** marshal `*PackageDesc` directly. `Describe()` returns the
internal descriptor; emitting it verbatim pins an internal Go type to a public
wire contract and turns every future `FieldDesc` addition into a compatibility
question.

### T8: Per-field `-file` hatch at both nesting levels

**Outcome**: `--<name>-file` for maps, repeated non-string, and record fields,
**and** `--<record>-<field>-file` for the matching sub-field cases.
Within-field precedence is `-file > -json > flattened sub-flags`.

**Verification**: a nested record loads from `--<name>-file`; a repeated
sub-field loads from `--<record>-<field>-file`; `-file` supersedes `-json` when
both are given; the README table shows top-level and sub-field forms agreeing for
every `FieldType` × `Repeated` combination.

**Notes**: registering `-file` at top level only would re-open the drift T1
closes — this parity requirement is the concrete reason 041 and 042 are one plan.
These are all string flags, so the existing `!= ""` check (`flags.go:310-369`) is
sufficient; no set-detection machinery is needed.

### T9: Toolkit reserved-name validation

**Outcome**: `validateCommandGroups()` (`toolkit/toolkit.go:1061`) hard-errors
when a spec declares a top-level command or group named `mvep`.

**Verification**: `toolkit/testdata/21_reserved_namespace_collision.jsonc` fails
generation with a message naming the reserved word; existing collision fixtures
13–20 still behave as before.

**Notes**: the toolkit has **no** reserved-word validation today — it will not
even reject `SubmitJob`/`GetJobStatus`, reserved at the protocol level in
`runtime/go/mvep/job.go:33`. Reserving those too is optional scope for this task.

### T10: Documentation and release

**Outcome**: `runtime/go/mvep/cli/README.md` documents input sources, the
namespace verbs, `--mvep-output`, and a flag-form table listing top-level and
sub-field columns side by side. `CHANGELOG.md` entries. `runtime/go` minor
tagged, `toolkit/go.mod` bumped, plan 023 reconciled.

**Verification**: the README table matches the implemented forms for every
`FieldType` × `Repeated` × depth combination; `GOWORK=off go build ./...` in
toolkit succeeds against the published tag.

**Notes**: the flag-form table is the artefact that keeps parity visible — it is
how the next contributor notices if top-level and sub-field diverge again.

## Risks and Compatibility

- **Flag-name and arity changes for repeated sub-fields (T1).** A repeated
  numeric sub-field moves from `--<record>-<field>` to
  `--<record>-<field>-json`, and repeated strings become repeatable. Nothing can
  depend on the old forms: passing them always errored. The bug is total, not
  partial, so there is no working usage to break.
- **Strict key validation could reject a payload the decoder would accept (T4).**
  Mitigated by normalising case and underscores, which matches both protojson's
  dual naming and `encoding/json`'s case-insensitive fallback. Any tightening
  beyond that risks breaking pb3 packages and must be tested against a
  pb3-generated command.
- **Panic on namespace collision (T2)** is a behaviour change for a
  hand-constructed descriptor. Mitigated by T9 catching it at generation time and
  by the namespace override option.
- **Reserved namespace claims one identifier** from every spec's top-level name
  space. `mvep` is implausible as a real command name, which is precisely why the
  namespace was chosen over reserving `exec`/`send`/`describe`.
- **Implicit stdin could surprise** a caller who pipes something unrelated into a
  flag-driven invocation. Contained: implicit reading only happens under
  `mvep exec`, never on generated subcommands.
- **Payload pre-validation is a second parse per record.** Negligible for `exec`;
  for `send` it is per-record overhead and is the first thing to make opt-out if
  streaming throughput ever matters.
- **JSON error rendering depends on an arg pre-scan** when flag parsing fails
  before `--mvep-output` is read. The fallback must agree with the parsed value
  in every other case, or the output stream silently changes shape depending on
  where the failure occurred.
- Records with no repeated sub-fields, and specs that never touch the namespace,
  are untouched — the new branches are simply not taken.

## Verification

1. Test-first per `test-first.instructions.md`. `go test ./...` in `runtime/go`
   and `toolkit`; `go vet` clean.
2. Existing `flags_test`, `required_test`, `record_nonstring_test`,
   `groups_test`, `renderer_test`, `json_error_test`, `extension_test` pass
   unchanged — no regression on the flag path.
3. Flag path and payload path produce an identical command struct and identical
   hook/executor ordering for the same logical input.
4. `zirafa runtime register` accepts a two-element `argsTemplate` via a `replace`
   directive onto the T1 branch, and the persisted JSON round-trips.
5. `cat p.json | svc mvep exec generate` and `svc mvep exec generate --input p.json`
   agree; unknown keys hard-error.
6. Under `--mvep-output json` a failing command emits `{"error":...}` on stdout,
   leaves stderr empty, and still exits non-zero.
7. `send` streams incrementally, continues past a bad record, honours
   `--fail-fast`, and exits non-zero when any record errored.
8. `svc mvep describe <cmd>` output is unchanged by a synthetic `FieldDesc`
   addition.
9. A pb3-generated package round-trips a payload containing an enum and a
   well-known type through `mvep exec` — validates the `oenc` decision.
10. `GOWORK=off go build ./...` in toolkit after each runtime bump.

## Rollout

Two releases from one plan:

1. **T1 first**, in its own PR, merged and tagged as a `runtime/go` **patch**.
   Bump `toolkit/go.mod`, verify standalone `GOWORK=off`, notify zirafa. This
   unblocks the field report without waiting for T2–T10.
2. **T2–T10** as a `runtime/go` **minor**, then bump `toolkit/go.mod` again and
   tag `toolkit` for the T9 validation change.

CI runs with `GOWORK=off`, so each `go.mod` bump is mandatory — a missed bump
silently builds against the stale published runtime. Follow
`/memories/repo/plan-040-release.md`.

## Decision Log

- **2026-08-12 — Reserved group namespace over reserved verbs or a `mvep-`
  prefix.** A namespace reserves one identifier instead of N, makes future verbs
  addable with no new reservation and no breaking change, reuses plan 040's group
  machinery, and gives one self-documenting help page. Rejected: top-level
  `exec`/`send`/`describe` (all plausible real command names in user specs) and
  `mvep-` hyphen prefixes (same isolation, N names, uglier).
- **2026-08-12 — Whole-payload `--input-file` per subcommand rejected.** It
  requires flag set-detection, which ugo v0.7.0 cannot provide: `Run` parses a
  throwaway merged FlagSet and `RunE` receives only positional args. Rather than
  wrap every `flag.Value` to track `Set()` — with reset-between-runs leakage — the
  dispatcher removes the need entirely.
- **2026-08-12 — `exec` takes a bare payload; `send` takes full envelopes.**
  Two shapes, two commands. `send`'s self-describing records are what make
  streaming natural, and mixing both into one flag would have needed format
  sniffing.
- **2026-08-12 — Decode via `oenc.LookupEncoding`, not `encoding/json`.**
  Matches the server's `executeCmd()` exactly, so CLI and wire semantics cannot
  diverge, and pb3 packages get protojson for enums, `oneof`, and well-known
  types.
- **2026-08-12 — Unknown payload keys are a hard error**, consistent with the
  reserved-name enforcement decision. Silent key-dropping is the worst failure
  mode for a scripting surface.
- **2026-08-12 — Strictness is a pre-validation pass, not a decoder option.**
  `oenc` cannot reject unknown fields. The walk recurses through record `Ref`s
  rather than checking top-level keys only, because sub-field typos are the case
  this plan exists to serve. Key comparison normalises case and underscores —
  required for correctness, since protojson accepts both `snake_case` and
  `lowerCamelCase` and `encoding/json` already matches case-insensitively.
- **2026-08-12 — The global output flag is `--mvep-output`.** A persistent root
  flag claims a name on every generated command, so it is as much a namespace
  claim as a top-level command is. Prefixing it keeps `renderer.go:18`'s
  statement that `--output` "is the implementor's to add" true, which deletes the
  contract reversal, the collision policy, and the panic this plan previously
  needed. Flags scoped inside the namespace (`--input`, `--fail-fast`) cannot
  collide and stay unprefixed.
- **2026-08-12 — `send` reuses `ContextWithCmdReq`, adding no context key.**
  `envelope.go:135` already provides it, plus `GetRequestHeader` /
  `SetResponseHeader`. Reusing the protocol's own mechanism means header-reading
  interceptors behave identically under the CLI and over HTTP, and response
  headers round-trip for free. Rejected: a CLI-local header key, which would have
  pointed the dependency from protocol toward CLI.
- **2026-08-12 — Errors serialize as JSON on stdout under `--mvep-output json`.**
  A consumer that parses stdout should not have to parse stderr too, and reusing
  `mvep.ErrorInfo` under an `error` key makes a failed `exec` shaped identically
  to a `send` record's `CmdResp.Error`. Rendering is additive: `RunWithIO` still
  returns the error and exit codes are untouched, so `set -e` scripts and
  `json_error_test.go` both keep working. Rejected: errors on stderr in JSON
  mode, which splits one logical outcome across two streams.
- **2026-08-12 — Continue-on-error is the `send` default.** It mirrors
  `CmdResp.Error` on the wire; `--fail-fast` is opt-in.
- **2026-08-12 — `describe` emits a hand-written versioned projection**, not a
  marshalled `*PackageDesc`, to avoid pinning an internal type to a public
  contract.
- **2026-08-12 — 041 and 042 unified.** The parity invariant must be enforced
  once: T8's `-file` hatch would otherwise re-introduce the top-level /
  sub-field drift T1 closes. Sequencing preserves 042's fast patch release.
- **2026-08-12 — Sub-field repeated binding mirrors `registerRepeatedFlag`, not
  the scalar switch.** The two sets differ: the scalar switch string-binds
  `FieldString`, `FieldUUID`, `FieldTimestamp`, `FieldBytes` and `FieldDuration`,
  while `registerRepeatedFlag` string-binds `FieldString` alone. Only the
  top-level comparison expresses the invariant T1 exists to restore, so repeated
  sub-fields string-bind `FieldString` only. Rejected: widening top level to
  match a larger sub-field set — better UX, but it enlarges a patch whose purpose
  is to unblock zirafa quickly. Logged as a Non-goal for a later minor.
- **2026-08-12 — `-json` sub-field validation checks array-ness, not just
  validity.** Unmarshalling into `json.RawMessage` accepts any valid JSON, so a
  JSON object would pass the closure and fail at the parent with the opaque
  `--<record>: json: cannot unmarshal ...` error the closure exists to replace.
- **2026-08-12 — Fix in the runtime binder, not in codegen.**
  `FieldDesc.Repeated` is already present and correct for nested record fields;
  codegen emits the right descriptor and the binder ignores it. Teaching codegen
  to special-case this would duplicate field data into `Ref` — the exact drift
  plan 025 removed.
- **2026-08-12 — `-json` suffix for repeated non-string sub-fields**, consistent
  with the top-level and map fallbacks, rather than a comma-separated list —
  comma-splitting is ambiguous for strings and does not generalise to `recRef`.
- **2026-08-12 — Flags precede positionals inside `mvep exec`/`send`.** ugo
  inherits stdlib `flag` parsing, which stops at the first non-flag argument, so
  `mvep exec echo_cmd --input p.json` leaves `--input` unbound; the flags must
  come first: `mvep exec --input p.json echo_cmd`. A Cobra-style pre-scan that
  pulls `--input`/`--fail-fast` out of the stream wherever they appear is
  feasible and low-risk (it mirrors the `--mvep-output` pre-scan in T5, and ugo
  even ships a `Preprocessor` hook for exactly this), but is deferred: `exec` is
  script-facing where the flag-first form is easy to standardize, and adding it
  is an ergonomic nicety rather than a correctness fix. Revisit if a human-facing
  consumer objects.

## Known Limitations

- A required field whose legitimate value is the zero value cannot be satisfied
  by a payload alone — `checkRequired` (`app.go:277`) uses `isZeroValue`.
- True round-tripping (`svc a --mvep-output json | svc mvep exec b`) only works
  when the result shape matches the next command's request shape, which is
  usually false. Do not advertise it as a general capability.
- Payload keys are validated against the descriptor, so a spec whose `Ref` cannot
  be resolved has that subtree accepted unchecked rather than rejected.

## Status

Implemented on branch `feat/049-cli-pipe-input-output` (issue #49). T1 shipped as
`runtime/go v0.11.1` (patch, PR #50); T2–T10 ship as a `runtime/go` minor, then a
`toolkit` release for the T9 validation change (see Rollout).
