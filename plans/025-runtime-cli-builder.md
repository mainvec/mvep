# 025 — Package Descriptor and Runtime CLI Builder

- Issue: [#25](https://github.com/mainvec/mvep/issues/25)
- Related: [#23](https://github.com/mainvec/mvep/issues/23) (type-coverage requirements catalogue, stays open), [#26](https://github.com/mainvec/mvep/issues/26) (`SrvDef` → `SpecDef` rename, kept separate)
- Branch: `feat/25-runtime-cli-builder`

## Problem / Goal

Today a CLI is produced by emitting a whole `main.go` from `go_cli_main.txt`. That template
hardcodes `package main`, hardcodes `GetCommandRunner()`, supports only string/boolean/int32
flags, discards command results, and returns no meaningful exit codes. Anything an implementor
wants that the template does not already emit — a global `--endpoint` flag, an extra
subcommand, JSON output, an auth hook — cannot be added without either editing generated code
(lost on regen) or forking the template.

Underneath that specific complaint is a more general gap. `mvep.Package` already acts as a
partial, stringly-typed descriptor: `InstanceOf(name) (any, bool)` constructs a command by
name, `NameOf(comp)` is its inverse, and the optional `CommandLister.CommandNames()` lists
them. What it lacks is **field-level metadata** — types, tags, ordering, required-ness. Because
that metadata does not exist at runtime, every consumer that needs it has to have code
generated for it. A CLI is only the first such consumer.

So the goal is two things, in order:

1. Give every generated package a **complete runtime descriptor**, emitted into
   `mvep_package.go` and backing the `Package` methods it already implements by hand.
2. Build `mvep/cli` as a **library** on top of that descriptor, so implementors get a
   `*cli.App` they can extend, executing through the same `mvep.CommandRunner` (in-process) or
   `client.PackageClient` (remote) the runtime already has.

The original framing of this issue was "parse the spec JSON at runtime, no codegen". That is
**not** what this plan does, and the Decision Log records why.

## Goals

- A generated package exposes a full descriptor: commands, fields, results, types, tags, order.
- `InstanceOf` / `NameOf` / `CommandNames` are **derived** from that descriptor rather than
  emitted as three separate hand-shaped switch statements.
- `mvep/cli` is a library in `runtime/go`, importable and extendable.
- One executable can drive commands **locally** or **remotely** by swapping an `Executor`.
- Flag → struct binding is total for the types the spec can express, and a codegen mistake is a
  **compile error**, not a silent runtime drop.
- Command, flag and help ordering is deterministic.
- The legacy generated-`main.go` path keeps working until consumers migrate.

## Non-goals

- Async execution (`--async`, job polling). `mvep/job.go` exists but is out of scope.
- Shell completion.
- TypeScript parity — `runtime/ts` is untouched.
- `oneOf` / mutually exclusive flag groups. The spec cannot express them.

### Enabled but deliberately not built

The descriptor makes all of the following straightforward. None are in scope; they are listed
so the descriptor shape stays general enough not to block them:

- Required-field validation in `PackageHandler.executeCmd`, before dispatch.
- Field-level redaction in interceptors/middleware, driven by `FieldDesc.Tags`.
- A server-side command/field discovery endpoint.
- A generic `mvep call <spec.json>` client.

## Proposed Design

### The descriptor lives in core `mvep`

Codegen emits a `mvep.PackageDesc` Go literal **into the generated `mvep_package.go`**, from the
same `ExecuteGenerate` run that produces the command structs:

```go
// generated
var pkgDesc = mvep.PackageDesc{
    Name:        "mvep",
    SpecVersion: "0.2",
    Commands: []mvep.CommandDesc{{
        Name:  "generate",
        Alias: "gen",
        Desc:  "Generate code from a spec",
        New:   func() any { return &GenerateCmd{} },
        Fields: []mvep.FieldDesc{{
            Name: "in", Fnum: 1, Type: mvep.FieldString, Required: true,
            Desc: "input spec file",
            Ptr:  func(c any) any { return &c.(*GenerateCmd).In },
        }},
        Result: &mvep.ResultDesc{ /* ... */ },
    }},
}
```

Three properties follow from this shape, and they are the whole reason for it:

1. **Descriptor and structs are generated together**, so they cannot disagree. No digest, no
   embedded spec copy, no provenance check, no version pin between them.
2. **`Ptr` closes over a real struct field.** If codegen emits a field that does not exist or a
   type that does not match, the generated package **fails to compile**. Compare an embedded
   spec, where a stale field is silently dropped by `json.Unmarshal` — the flag parses, the
   user sees no error, and the value never arrives.
3. **No reflection in the binding path.**

### `Ptr`, not `Bind`

An earlier draft had `FlagDesc.Bind(cmd any, fs *cli.FlagSet)`. That cannot live in core
`mvep`, because it references a `cli` type and everything imports `mvep` — core must depend on
nothing.

Instead `FieldDesc` carries a typed pointer accessor, `Ptr func(cmd any) any`, returning
`*string`, `*int64`, `*[]string` and so on. `mvep/cli` type-switches on it to pick the right
`FlagSet` var helper. The same accessor serves every other consumer: validation reads through
it to test for zero values, redaction writes through it to scrub a field. One accessor, many
consumers, compile-time safety retained.

The rejected alternative was core-holds-metadata + `cli`-emits-its-own-`Bind`-table. That means
codegen emits two parallel tables that can drift from each other — reintroducing exactly the
class of bug this design exists to remove.

### Descriptor shape

Owned by `runtime/go/mvep`, runtime-shaped, **not** `toolkit.SrvDef`:

- `PackageDesc{ Name, Namespace, Title, Desc, Base, SpecVersion string; Commands []CommandDesc; Records []RecordDesc }`
- `CommandDesc{ Name, Alias, Desc string; New func() any; Fields []FieldDesc; Result *ResultDesc }`
- `ResultDesc{ Name string; New func() any; Fields []FieldDesc }`
- `FieldDesc{ Name, Alias, Desc string; Fnum int32; Type FieldType; Repeated, Required bool; Tags []string; Ptr func(any) any; Ref *RecordDesc }`
- `RecordDesc{ Name string; Fields []FieldDesc }`
- `FieldType` — a runtime-owned enum mirroring the spec's types
- `PackageDescriber interface { Describe() *PackageDesc }` — optional, mirroring the existing
  `CommandLister` idiom

The toolkit keeps full ownership of the spec model. There is no relocation of `SrvDef` into
`runtime/go`, and adding a spec field requires a runtime release only if the descriptor must
carry it. `SpecVersion` is informational, for `--version` output.

### What the descriptor deliberately excludes

`PackageDesc` describes what a package **is** at runtime, not how it was **built**. Two `SrvDef`
fields are therefore excluded:

- **`ProtocOpts`** — marked `json:"-"` and self-described as a transient holder. It is assembled
  in `toolkit_pb3.go` and spent as `protoc` argv in `toolkit_go.go`. It is a subprocess command
  line with no runtime meaning.
- **`GenOpts`** — every consumer is codegen-time: `go_package` and `go_api_package` select output
  packages, `format` selects the encoding of the generated structs. The runtime does not need
  `format`, because encoding is resolved per-request through `oenc.LookupEncoding` from the
  envelope, not from the spec.

The decisive argument against `GenOpts` is disclosure rather than tidiness: `go_package` and
`go_api_package` are internal module and filesystem paths. A discovery endpoint that serialises
`PackageDesc` would publish internal repository layout to every client.

If a generation option is ever genuinely needed at runtime, it gets **promoted to a typed
descriptor field on an allowlist basis**. A passthrough `map[string]string` would make every
future genopt part of core runtime API by default.

`ResultDesc` is not optional garnish: `InstanceOf` today returns **both** command types and
`*XxxCmdResult` types, so the descriptor cannot back it without describing results.

All collections are **ordered slices**, never maps. This is load-bearing. `omap.OMap[K,V]` is a
plain `map[K]V`; ordering exists only when you go through `omap.IteratorByKey` or
`IterateByValue`. Command ordering in help output is nondeterministic today. Slices fix it by
construction, and codegen must emit through those iterators to get a defined order.

### Deriving the `Package` methods

The runtime gains a single constructor:

```go
// also satisfies CommandLister and PackageDescriber
func NewPackageFromDesc(desc *PackageDesc) Package
```

Generated code collapses to
`func NewPackage() mvep.Package { return mvep.NewPackageFromDesc(&pkgDesc) }`. The generated
`mvepPackage` struct is **deleted outright** rather than kept as a delegator — it is an empty
struct today and carries no state. The derivation logic, including the type map below, is
written and tested once in the runtime instead of re-emitted per package.

`NewPackageFromDesc` is **additive, never mandatory**: `mvep.Package` stays implementable by
hand, as `runtime/go/test/api/iunet_package.go` does.

`NameOf(comp any) string` needs type identity, which today is a generated type switch. Derived,
it becomes a `map[reflect.Type]string` built **once** from the `New()` closures. That is a
single one-time use of `reflect.TypeOf` and O(1) per call — faster than the O(n) switch it
replaces. Worth stating plainly: "no reflection" is a property of the **binding path**, not an
absolute ban across the runtime.

Three constraints govern this change. Each is a real hazard, not a style preference:

1. **`GetName()` must return the same string as today**, because it feeds HTTP routing in
   `server/server.go` and `client/client.go`. The template emits `return "{{$PKG}}"`, where
   `$PKG := print .NAME "Package"` and `.NAME` is `srvDef.Name`. `GetName()` is therefore
   mechanically `Name + "Package"` — spec name `mvep` yields `"mvepPackage"`, `iunet` yields
   `"iunetPackage"`. `NewPackageFromDesc` reapplies that suffix rather than returning bare
   `desc.Name`, which would move every route from `/api/mvepPackage/cmd` to `/api/mvep/cmd` and
   404 existing clients. The suffix is a legacy compatibility shim; dropping it is wire-breaking
   and needs its own issue.
2. **Exported package-level `InstanceOf` and `NameOf` are kept.** Consumers call them directly —
   `iunetApi.InstanceOf("NewIUHubCmd")` appears in the `util/protobuf` and `util/protojson`
   tests. Only the switch *bodies* are replaced by delegation; deleting the symbols would break
   the generated package's API.
3. **Implementing `CommandLister` is a behaviour change.** No generated package implements it
   today, and `client.go` uses it for cross-package duplicate-command detection. Once every
   generated package has it, a multi-package client whose command names collide begins erroring
   at registration where it previously succeeded silently.

### Execution

`client.PackageClient.SendCmd(ctx, cmd any) (any, error)` and
`mvep.CommandRunner.RunCmd(ctx, cmd any) (any, error)` already have identical signatures, so
one interface covers both:

```go
type Executor interface {
    Run(ctx context.Context, cmd any) (any, error)
}
```

- `cli.LocalExecutor(runner mvep.CommandRunner)` — in-process.
- `cliclient.RemoteExecutor(pc *client.PackageClient)` — in subpackage `mvep/cli/cliclient`, so
  `mvep/cli` never takes a dependency on `mvep/client`.

### Rendering

Built directly on `ugo/cli` v0.7.0. An earlier draft put a renderer interface behind the builder
so a stdlib-`flag` implementation could ship while ugo caught up. v0.7.0 already provides
persistent flags, `RunE`-style error returns, and — via an embedded `flag.FlagSet` — the scalar
var helpers, so that abstraction no longer pays for itself. It can be reintroduced if a second
frontend is ever genuinely needed.

### Strictness at generate time

If a spec uses a construct the descriptor or CLI builder cannot represent, `mvep generate`
**fails**. This is stronger than validating at `cli.New` time: the error lands on the developer
running codegen, with the spec in hand, rather than on an end user at startup.

### Stability marker

`PackageDesc` and friends are core public API from day one, which makes them harder to change
than a `cli`-internal type would have been. They ship marked `// EXPERIMENTAL: shape may change`
for one release cycle, and the marker is removed once T16 has dogfooded the design.

## Affected Modules

| Module | Change |
|---|---|
| `runtime/go/mvep` | New: descriptor types, `PackageDescriber`, derivation helpers |
| `runtime/go/mvep/cli` | New package: `Executor`, `cli.New`, `App.Run`, flag binding, renderers |
| `runtime/go/mvep/cli/cliclient` | New subpackage: `RemoteExecutor` |
| `runtime/go/go.mod` | `ugo` v0.6.0 → v0.7.0. No other new dependency. |
| `toolkit/toolkit_go.go` | `SpecDef` → `PackageDesc` literal mapping |
| `toolkit/resources/codegen_templates/go` | Descriptor emission; `go_cli_main.txt` retained for `gen_options.cli: legacy` |
| `toolkit/toolkit_runner.go` | `--cli` mode flag |
| `toolkit/mvepapi/cmd/mvep` | Dogfood: mvep's own CLI rebuilt on the library |
| `runtime/ts` | Untouched |

Import direction is preserved: `toolkit → runtime/go`, never the reverse.

## Risks and Compatibility

| Risk | Mitigation |
|---|---|
| Descriptor is core API and hard to change later | `// EXPERIMENTAL` marker for one release cycle; removed after T16 |
| Deriving `Package` methods changes behaviour for existing consumers | T2 is behaviour-preserving by construction; existing `mvep_package.go` tests must pass unchanged |
| `GetName()` derivation silently rewrites HTTP routes | `NewPackageFromDesc` reapplies the `"Package"` suffix; T2 asserts routes are unchanged |
| Newly satisfying `CommandLister` activates duplicate-command detection | Real behaviour change: multi-package clients with colliding command names now fail at registration. Called out in the changelog |
| Removing exported `InstanceOf` / `NameOf` breaks consumers | Symbols retained as delegators; only the switch bodies change |
| Descriptor bloats generated output for large specs | Data-only static initialisers; measure at T16 |
| Regenerating overwrites a hand-edited `main.go` | `gen_options.cli: legacy` stays the default until T15; `// NOMVEP` still honoured |
| ugo v0.7.0 bump affects other consumers | Bump is additive; `Run` is deprecated but retained upstream |
| Required-flag semantics diverge from ugo's | Enforced in `cli`, not ugo, so the behaviour is ours to define |
| Spec grows a construct the descriptor cannot express | T5 makes it a generate-time failure, not a silent gap |
| **Generated code references descriptor types absent from the published runtime** | Descriptor emission is unconditional, so output from a toolkit release only compiles against a `runtime/go` that already ships the types. Strict release ordering — see Rollout step 0 |

Wire compatibility is unaffected: no envelope, header, or encoding change.

## Verification

- `cd toolkit && go test ./...`
- `cd runtime/go && go test ./...`
- Golden test: a fixture spec generates a descriptor that builds an `App` whose `--help` output
  is byte-stable across repeated runs in one process.
- Negative test: a descriptor referencing a non-existent struct field must **fail to compile** —
  asserted by running `go build` on a testdata package expected to error.
- Round-trip: every field type in `testdata/05_command_withfields.jsonc` and `08_maps.jsonc`
  binds from a flag and arrives in the command struct.
- Derivation parity: `InstanceOf` / `NameOf` / `CommandNames` return identical results before
  and after T2 for the existing `mvepapi` package.
- Leak guard: generated descriptor output for a spec whose `gen_options` set `go_package` and
  `go_api_package` must contain neither value. **Currently verified by hand against fixture 06
  only; still needs an automated regression test.**

## Rollout

0. **Release ordering is a hard constraint.** `toolkit/go.mod` requires
   `github.com/mainvec/mvep/runtime/go v0.9.0`, which predates the descriptor types, and
   emission is unconditional — so generated output does not compile against the published
   runtime. `runtime/go` must therefore be tagged with the descriptor types, and
   `toolkit/go.mod` bumped to it, **before any toolkit release** ships descriptor emission.
   Until that bump lands, two things follow: `TestGenerateCompilePlain` builds against the
   local checkout via a `replace` directive, and `toolkit/mvepapi/api/mvep_package.go` must
   **not** be regenerated — `go.work` would make it compile locally while leaving the published
   toolkit uninstallable via `go install`.
1. T1–T5 land the descriptor. Generated packages gain a descriptor and lose three switch
   statements; no CLI change, no consumer change.
2. T6–T11 land the CLI library against the descriptor, behind `--cli=runtime`, default still
   `legacy`.
3. T12–T14 add the extension surface.
4. T15 flips the default to `runtime`; `legacy` and `none` remain selectable.
5. T16 dogfoods mvep's own CLI — the real acceptance test — and drops the `EXPERIMENTAL` marker.
6. `go_cli_main.txt` is removed no earlier than the release after the default flip.

## Decision Log

- **Descriptor in core `mvep`, not `mvep/cli`.** `mvep.Package` is already a degenerate
  descriptor; putting a second one in `cli` would duplicate `InstanceOf`, `NameOf` and
  `CommandNames`. Placing it in core also unlocks validation, redaction and discovery, none of
  which are CLI concerns.
- **`FieldDesc.Ptr` over `FlagDesc.Bind`.** `Bind` references a `cli` type and cannot live in a
  package that everything imports. `Ptr` keeps compile-time safety, adds no reflection to the
  binding path, and serves non-CLI consumers.
- **Derive the `Package` methods** rather than emitting them alongside the descriptor, through a
  single `mvep.NewPackageFromDesc`. One source of truth; accepts one one-time `reflect.TypeOf`
  map for `NameOf`. Additive — hand-written `Package` implementations keep working.
- **No `RouteName` field.** `GetName()` is mechanically `Name + "Package"`, so the descriptor
  carries `Name` alone and the runtime reapplies the suffix. A second field was considered and
  rejected as redundant data that could drift from `Name`.
- **Always emit the descriptor**, never opt-in. Consumers can rely on it unconditionally.
- **`GenOpts` and `ProtocOpts` are excluded.** Both are build-time inputs. `go_package` and
  `go_api_package` are internal paths that a discovery endpoint would otherwise disclose. Any
  genuinely runtime-relevant option is promoted to a typed field, never passed through as a map.
- **`Namespace`, `Base`, `Title` and `Desc` are included**, having passed the same
  is-it-runtime-shaped test that excluded the build options.
- **Generated descriptor over runtime spec parsing.** The issue asked for runtime parsing. It
  was rejected because shipping a spec next to generated structs creates several drift paths and
  all of them fail *silently* — `json.Unmarshal` ignores unknown fields, so a stale spec yields
  a flag that parses and is then dropped. Codegen drift fails *visibly*: the flag is simply
  absent. The library goal is fully preserved; only the metadata source changed.
- **Go literal over gob.** Gob ignores stream fields absent from the target struct — precisely
  the fail-silent behaviour being designed out. A Go literal fails to compile instead, costs
  nothing at startup, needs no `encoding/gob` import or type registration, and is diff-reviewable.
- **Runtime-shaped descriptor over `spec.SrvDef`.** Avoids relocating the spec model into
  `runtime/go` and decouples release cadence.
- **No JSON binding hop.** Marshalling flags to JSON and unmarshalling into the command struct
  would drag encoder wire conventions into the CLI (protojson requires `int64` as a *string*,
  plain JSON as a number) and degrade errors from `invalid value "abc" for --count` to
  `cannot unmarshal string into Go struct field .count of type int64`.
- **Renderer seam dropped.** Justified only by ugo gaps that v0.7.0 has already closed.
- **Ordered slices, not maps.** `omap.OMap` is a plain map; help ordering is nondeterministic today.
- **Strictness at generate time**, not construction time.
- **Legacy codegen preserved** behind `gen_options.cli: legacy`.
- **CLI mode is a spec gen_option, not a CLI flag.** It is read like the existing `format`
  genopt fallback, leaving `ExecuteGenerate`'s signature unchanged and the mode declarable
  next to the spec it describes. `skipCmd=true` still forces `none`.
- **ugo v0.7.0 as-is; no upstream change.** v0.7.0 already ships persistent flags, `RunE`,
  `Int32Var`, `StringSliceVar` and `BytesVar`. `Uint32Var` / `Float32Var` are absent, and
  rather than make this issue depend on a ugo release, `mvep/cli` carries two small custom
  `flag.Value` types. They can be swapped for upstream helpers if ugo ever ships them.
- **`RemoteExecutor` uses `SendCmdReq`, not `SendCmd`.** `SendCmd` drops the `*CmdResp` that
  carries `Error.Code`; exit-code classification needs it. No core API change required.
- **Exit codes key on error-code classes, not HTTP statuses.** `HTTPStatusForErrorCode`
  semantics truncate lossily into 0–255; a class mapping (2 usage / 3 not-found / 4 auth /
  1 other) is scriptable and honest about `executeCmd` collapsing runner errors to
  `command_error`.
- **`NewPackageFromDesc` is the derivation entry point.** One runtime helper builds the
  `Package` (plus `CommandLister` / `PackageDescriber`) from the descriptor; generated
  `NewPackage()` delegates to it and the three switch statements disappear.
- **The derived package is a shared singleton.** The generated `var pkg =
  mvep.NewPackageFromDesc(&pkgDesc)` is built once at package init; `NewPackage()` and the
  exported `InstanceOf` / `NameOf` delegators all return/use that one instance rather than
  rebuilding the lookup maps on every call. `*descPackage` is immutable after construction, so
  sharing is safe; the only behavioural delta is that `NewPackage() == NewPackage()` is now
  true, which no consumer should depend on.
- **#23 remains the requirements catalogue** for type coverage and is not closed by this work.
- **#26 (`SrvDef` rename) is not bundled here.**

## Progress

- [x] T1 — Core descriptor types and `PackageDescriber`
- [x] T2 — Derive `InstanceOf` / `NameOf` / `CommandNames`
- [x] T3 — Codegen emits the descriptor into `mvep_package.go`
- [x] T4 — Field type coverage in the descriptor
- [x] T5 — Generate-time hard error on unsupported constructs
- [x] T6 — ugo v0.7.0 bump; local `uint32` / `float32` flag values
- [x] T7 — `Executor` interface, local and remote adapters
- [x] T8 — `cli.New` and `App.Run`
- [x] T9 — Flag binding via `FieldDesc.Ptr`
- [x] T10 — Required flags
- [x] T11 — Deterministic command and flag ordering
- [ ] T12 — Global flags, custom subcommands, overrides
- [ ] T13 — Pre/post execution hooks
- [ ] T14 — Result renderers and exit codes
- [ ] T15 — `gen_options.cli: runtime|legacy|none`
- [ ] T16 — Dogfood mvep's own CLI
- [ ] T17 — Documentation

## Tasks

### T1 — Core descriptor types and `PackageDescriber`

Define `PackageDesc`, `CommandDesc`, `ResultDesc`, `FieldDesc`, `RecordDesc`, `FieldType` and
the optional `PackageDescriber` interface in `runtime/go/mvep`. Ordered slices only.

- **Outcome:** Runtime carries a complete, dependency-free description of a package.
- **Verification:** A hand-written descriptor in a test compiles and iterates in stable order.
- **Notes:** Mark `// EXPERIMENTAL: shape may change`. No new dependency. `SpecVersion` is
  informational only. Carries no build-time options — see "What the descriptor deliberately
  excludes".
  **Done (2026-08-07):** descriptor.go added with `PackageDesc`, `CommandDesc`, `ResultDesc`,
  `FieldDesc`, `RecordDesc`, `FieldType` (14 variants), `PackageDescriber`. New descriptor_test.go
  asserts stable iteration order over 50 passes and full FieldType coverage. `go test ./mvep/` and
  `go vet ./mvep/` green. Required-ness is tag-derived (`tags: ["required"]`); T3 added
  `FieldIsRequired` for the Go descriptor, but the JS/TS `IsRequiredField` still hardcodes false —
  reconciled in T4.

### T2 — Derive `InstanceOf` / `NameOf` / `CommandNames`

Add `mvep.NewPackageFromDesc(desc *PackageDesc) Package`, satisfying `Package`, `CommandLister`
and `PackageDescriber`. Build the `map[reflect.Type]string` for `NameOf` **eagerly at
construction** from the `New()` closures — no `sync.Once`, no mutex, race-free by construction.
Generated `NewPackage()` returns it, the empty `mvepPackage` struct is deleted, and exported
`InstanceOf` / `NameOf` become one-line delegators.

- **Outcome:** Generated packages stop emitting three switch bodies and gain `CommandLister`.
- **Verification:** Parity test — identical results to the current generated `mvepapi` methods,
  including for `*XxxCmdResult` types. `GetName()` must still return `"mvepPackage"`, and
  registered server routes must be unchanged. `iunetApi.InstanceOf("NewIUHubCmd")` must still
  compile and pass.
- **Notes:** Depends on T1. Additive — `mvep.Package` stays hand-implementable, as
  `test/api/iunet_package.go` does. Newly satisfying `CommandLister` activates the duplicate
  command check in `client.go`; note it in the changelog.
  **Done (2026-08-07):** `NewPackageFromDesc` implemented in descriptor.go as `*descPackage`
  (byName + nameByType maps built once from `New()` closures; `GetName` reapplies the
  `"Package"` suffix). descriptor_test.go asserts GetName/InstanceOf/NameOf parity plus
  `CommandLister` + `PackageDescriber`. Full `go test ./...` and `go vet ./...` green across
  the runtime module. Full parity against the real generated `mvepapi` lands with T3, once
  codegen emits a descriptor to compare against.

### T3 — Codegen emits the descriptor

Emit the `PackageDesc` literal into `mvep_package.go` for every generated package.

- **Outcome:** `mvep generate` produces a descriptor with no opt-in required.
- **Verification:** Generated output for a testdata spec compiles and is byte-stable across runs.
- **Notes:** Depends on T1. Emit via `omap.IteratorByKey` / `IterateByValue` so order is defined.
  **Done (2026-08-07):** `go_package_code.txt` rewritten — emits `pkgDesc` (commands via
  `SortedCommandNames`, fields via `SortFieldsByFnum`, records via `SortedRecordNames`), derives
  `NewPackage`/`InstanceOf`/`NameOf` via `NewPackageFromDesc`, adds `Describe()`, deletes the
  `mvepPackage` struct and the three switch bodies. New helpers in toolkit.go: `SortedCommandNames`,
  `SortedRecordNames`, `GoFieldTypeEnum`, `FieldIsRequired` (tag `"required"`), `GoStringLit`,
  `GoStringSliceLit`. toolkit_descriptor_test.go asserts emission, byte-stability (5 passes), field
  order + enums. Leak guard verified clean on 06 (`go_package` not in output) — by hand, not yet a
  test. recRef emits `Type: FieldRecord` + `Ref`, but name-only: `Ref.Fields` is empty and nothing
  links it to `PackageDesc.Records`, which T9's depth-1 flattening needs. **Required fix:**
  `TestGenerateCompilePlain` now writes a `replace` to the local runtime/go — the proxy's v0.9.0
  predates the descriptor types, so the compile test must build against the local module until a
  runtime release ships them. Toolkit suite + runtime suite + vet all green. `mvepapi`/`iunet`
  regeneration is **blocked**, not merely deferred: until `toolkit/go.mod` bumps to a runtime
  release carrying the descriptor types, regenerating `mvepapi/api/mvep_package.go` would compile
  locally through `go.work` while breaking `go install` of the published toolkit. It lands with the
  runtime bump; T16 then dogfoods it.

### T4 — Field type coverage in the descriptor

string, bool, int32, int64, uint32, sint32, float, double, bytes, timestamp, duration, uuid,
plus `repeated`, maps, and `$ref` records.

- **Outcome:** Every construct the spec can express is described, with a correct `Ptr`.
- **Verification:** Table test over `05_command_withfields.jsonc`, `06_command_with_ref.jsonc`
  and `08_maps.jsonc`.
- **Notes:** Depends on T3. `FieldDesc.Tags` landed in `93dfda1`. See #23 for the full catalogue.
  Resolve `FieldDesc.Ref` here — populate `Fields` (or point at `PackageDesc.Records`) so T9 can
  flatten. Two near-identical template funcs now disagree and must be reconciled:
  `IsRequiredField` (`toolkit_javascript.go`) is hardcoded `return false`, while
  `FieldIsRequired` (`toolkit.go`) is tag-derived. Go and TS output therefore disagree on
  required-ness, and the names differ only by word order — easy to pick wrong, and it fails
  silently.
  **Done (2026-08-07):** Full-type-coverage fixture `testdata/11_descriptor_type_coverage.jsonc`
  added (one field per spec type: string, bool, int32, int64, uint32, sint32, float, double, bytes,
  timestamp, duration, uuid, repeated, map, recRef). `TestGenerateDescriptorFullTypeCoverage`
  asserts every type emits its `FieldType` enum in fnum order. `FieldDesc.Ref` stays name-only in
  codegen output (`Ref: &RecordDesc{Name: "Address"}`) — the record's full `Fields` live once in
  `PackageDesc.Records`, and a new runtime helper `PackageDesc.Record(name) (*RecordDesc, bool)`
  resolves them, so T9 can flatten without codegen duplicating field data into `Ref` (drift
  hazard). `TestGenerateDescriptorRefIsNameOnlyResolvable` covers fixture 06; the runtime
  `TestPackageDescRecordResolvesByName` covers the helper. Reconciled the two required-ness
  helpers: `toolkit_javascript.go:isRequiredField` now delegates to `fieldIsRequired` (tag-
  derived), so Go and JS/TS output agree — `TestIsRequiredFieldAgreesWithFieldIsRequired` is the
  parity guard. Toolkit suite + runtime suite + vet all green. T5's hard-error on unknown types
  remains a `panic` in `goFieldTypeEnum`; T5 converts it to a returned error naming command+field.

### T5 — Generate-time hard error on unsupported constructs

- **Outcome:** Unrepresentable specs fail at `mvep generate`, not at runtime.
- **Verification:** A negative testdata spec makes `ExecuteGenerate` return an error naming the
  offending command and field.
- **Notes:** Depends on T4. The message must identify command and field, not just "unsupported".
  **Done (2026-08-10):** Negative fixture `testdata/12_unsupported_construct.jsonc` added (an
  inline `record` field type — schema-valid but unrepresentable in the descriptor). New
  `validateDescriptorRepresentable(srvDef)` in toolkit.go walks every command and record field
  (via `omap.IteratorByKey` so the owner name is in scope) against a single
  `descriptorSupportedFieldTypes` set, kept in sync with `goFieldTypeEnum`, and returns an error
  naming the offending command/record and field. Called from `executeGenerateGo` before any
  template executes, replacing the deep `goZeroValue`/`goFieldTypeEnum` panics with a clear,
  actionable error. `TestExecuteGenerateUnsupportedConstructReturnsError` asserts the error
  names `BadCmd` and `inlineRec`. Toolkit suite + runtime suite + vet all green. The check is
  Go-only for now (JS codegen has no descriptor emission yet); JS can adopt it when it gains a
  descriptor.

### T6 — ugo v0.7.0 bump; local `uint32` / `float32` flag values

v0.7.0 is already released and ships persistent flags, `RunE`, an embedded `flag.FlagSet`
(`Int64Var` / `Float64Var` / `DurationVar`), `Int32Var`, `StringSliceVar`, and `BytesVar`
(base64 or `@file`). It does **not** ship `Uint32Var` or `Float32Var`, so `mvep/cli` defines
two small custom `flag.Value` types for those; no upstream ugo change is in scope. If ugo
later ships the helpers, swap them in.

- **Outcome:** Both modules on v0.7.0; uint32/float32 binding available in `mvep/cli`.
- **Verification:** `go test ./...` green in both modules after the bump.
- **Notes:** Blocks T9 and T10. Parallelisable with Phase 1.
  **Done (2026-08-10):** `runtime/go/go.mod` and `toolkit/go.mod` bumped `ugo` v0.6.0 → v0.7.0;
  both suites green after the bump with no other change. New `runtime/go/mvep/cli/flag_value.go`
  adds `Uint32Var` and `Float32Var` as small custom `flag.Value` types (`uint32Value` / `float32Value`)
  registered via `cli.FlagSet.Var`, since ugo v0.7.0 and the stdlib `flag.FlagSet` it embeds ship
  neither. `flag_value_test.go` is a table test covering default/zero/value/max/overflow/negative/
  non-numeric for uint32 and default/zero/value/negative/non-numeric for float32. Full runtime +
  toolkit suites and vet green. The helpers are ready for T9's flag binding switch; if ugo later
  ships upstream helpers they can be swapped in directly.

### T7 — `Executor` interface, local and remote adapters

`Executor.Run(ctx, cmd any) (any, error)`; `LocalExecutor` over `mvep.CommandRunner`;
`RemoteExecutor` over `client.PackageClient` in `mvep/cli/cliclient`.

`RemoteExecutor` must call `PackageClient.SendCmdReq` (which returns `(any, *mvep.CmdResp,
error)`), not `SendCmd`: `SendCmd` flattens the server error into a string and drops the
`*CmdResp` carrying `Error.Code`, which T14's exit-code classification needs. The executor
returns a typed error wrapping the code so the CLI can classify without string parsing.

- **Outcome:** One CLI binary can target in-process or remote execution.
- **Verification:** Both adapters satisfy `Executor`; a fake runner records the received command.
- **Notes:** The subpackage exists solely to keep `cli` free of a `client` edge.
  **Done (2026-08-10):** `cli.Executor` interface (`Run(ctx, cmd) (any, error)`) and `cli.ErrorCode`
  typed error (wraps `Code`/`Message`/`Err`) added in `runtime/go/mvep/cli/executor.go`. `LocalExecutor`
  is a struct wrapping `mvep.CommandRunner`; `RemoteExecutor` lives in the new
  `runtime/go/mvep/cli/cliclient` subpackage (preserving the import boundary — `cli` never imports
  `mvep/client`) and calls `PackageClient.SendCmdReq` (not `SendCmd`), wrapping `CmdResp.Error.Code`
  in `*cli.ErrorCode` on failure so T14 can classify exit codes without string parsing. Tests cover:
  LocalExecutor forwards result + propagates error; RemoteExecutor forwards a command end-to-end
  over httptest, propagates a server error as `*cli.ErrorCode{Code: "http_500"}`, and surfaces an
  unknown command as `*cli.ErrorCode{Code: "http_404"}`.
  **Discovery for T14:** the HTTP transport (`http_transport.go:151`) sets `CmdResp.Error.Code` to
  `"http_<status>"` (e.g. `http_500`), NOT the semantic code (`command_error`/`unknown_command`)
  the server sets in the `x-mainvec-error-code` response header. `PackageHandler.executeCmd`
  builds `CmdResp` with semantic codes; `WriteCmdResp` writes the semantic code to the header and
  maps it to an HTTP status; the client transport discards the header and synthesises `http_<status>`.
  T14 must recover the semantic code from the `x-mainvec-error-code` header (or the transport must
  be fixed to read it) to key exit codes on error-code classes as the plan intends. Runtime suite +
  vet green. Unblocks T8 (`cli.New` and `App.Run`).

### T8 — `cli.New` and `App.Run`

Walk a `*mvep.PackageDesc`, build the ugo command tree, execute via `Executor`. Fills the
currently-empty `cli.go`.

- **Outcome:** A descriptor plus an executor yields a working CLI.
- **Verification:** End-to-end test parses argv and asserts the populated command struct.
- **Notes:** Depends on T1 and T7. Use `RunE`, not the deprecated `Run`.
  **Done (2026-08-10):** `cli.New(desc, executor)` in `runtime/go/mvep/cli/app.go` walks a
  `*mvep.PackageDesc` and builds a ugo `*cli.Command` tree: one subcommand per `CommandDesc`,
  with snake_case command names (`EchoCmd` -> `echo_cmd`), spec aliases preserved, and the
  root rejecting unknown positional args (ugo otherwise silently prints help for an
  unrecognized command name). Each subcommand's `RunE` constructs the command struct via
  `New()`, applies parsed flag values through the `FieldDesc.Ptr` accessors, dispatches via
  `Executor.Run`, and prints the result (T14 replaces this with real rendering). `App.Run`
  wraps ugo's `Framework.Run` (os.Args); `App.RunWithIO` is the testable entry point.
  Minimal flag binding in `flags.go` handles string + int32 (the test-fixture types); T9
  replaces it with the full Ptr-driven type-switch over every `FieldType`. Tests cover:
  command tree built (verified via --help); EchoCmd end-to-end with --in/--count binding into the
  struct and dispatch through a recording Executor; PingCmd no-field dispatch; Executor error
  propagation; unknown command error; --help lists commands. Runtime suite + vet green.
  Unblocks T9 (full flag binding), T10 (required flags), T11 (ordering), T12 (global flags/
  custom subcommands — `App.Root()` exposes the root for extension).

### T9 — Flag binding via `FieldDesc.Ptr`

Type-switch on the accessor's dynamic type to select the `FlagSet` var helper.

- **Outcome:** Every described field is reachable from a flag.
- **Verification:** Round-trip test per type; an unhandled type fails the switch loudly.
- **Notes:** Depends on T4 and T6. Flatten nested records to depth 1 (`--record-field`); maps and
  anything deeper get `--x-json` / `--x-file`. Depth 1 is deliberate — deeper nesting is a
  generate-time error per T5.
  **Done (2026-08-10):** `flags.go` rewritten with the full Ptr-driven type-switch covering every
  spec FieldType: string, bool, int32/sint32 (both *int32), int64, uint32 (T6's custom Uint32Var),
  float32 (T6's custom Float32Var), float64, bytes (ugo BytesVar, base64), timestamp (custom
  timeValue flag.Value, RFC3339), duration (ugo DurationVar), uuid (string flag +
  encoding.TextUnmarshaler, no google/uuid dependency), map (JSON object via --<name>-json), and
  $ref record (depth-1 flattening: each record field becomes --<name>-<subField>, constructed via
  JSON unmarshal which handles nil pointer construction). Repeated string uses ugo StringSliceVar;
  repeated non-string gets --<name>-json (JSON array). The binding does NOT mutate the shared
  descriptor's Ptr — parsed values are held in per-execution closures and written back via Ptr
  after parsing. Tests: one subtest per scalar type (all 11), repeated string accumulation, map
  JSON binding, record depth-1 flattening. Runtime suite + vet green. Unblocks T10 (required
  flags — the binding now knows which fields are Required), T11 (ordering).

### T10 — Required flags

Honour `FieldDesc.Required`; enforce in `cli`, not ugo.

- **Outcome:** A missing required flag produces a usage error and exit code 2.
- **Verification:** Test asserts both message and exit code.
- **Notes:** Depends on T6.
  **Done (2026-08-10):** `checkRequired` in app.go runs after `applyBindings` and before
  dispatch: it walks `CommandDesc.Fields` and returns `"required flag --<name> is missing"` for
  the first `Required` field whose value is still the zero value of its type. Enforcement
  lives in `cli`, not ugo, so the behaviour is ours to define. `isZeroValue` type-switches on the
  Ptr return type for scalars/slices/maps and falls back to `reflect.IsZero` for pointer-to-struct
  records and non-probeable types (time.Time, uuid.UUID). A required field set to its zero value
  intentionally (--count 0) is indistinguishable from missing — the standard CLI convention. The
  executor is NOT called when a required flag is missing. Tests: missing required string flag,
  provided required string flag, missing required int32 flag, provided required int32 flag.
  Runtime suite + vet green. Exit-code 2 mapping is T14's job (this returns the error; the caller
  sets the exit code).

### T11 — Deterministic command and flag ordering

- **Outcome:** `--help` is byte-identical across runs.
- **Verification:** Golden-file test run repeatedly within one process.
- **Notes:** Guards against the `omap` nondeterminism described above.
  **Done (2026-08-10):** No production change needed — ordering is deterministic by construction:
  the descriptor uses ordered slices (T1), `cli.New` walks `desc.Commands` in slice order, and
  ugo's help iterates commands in `AddCommand` order (declaration order). ugo's flag help uses the
  stdlib `flag.FlagSet.VisitAll`, which sorts flags lexically — a deterministic, stable order
  (not fnum order; fnum order is preserved in codegen emission via `SortFieldsByFnum` in T3 and in
  the binding via `bindFlags` walking Fields in slice order in T9). Tests: `TestHelpIsByteStable`
  runs root --help and subcommand --help 10 times each within one process and asserts byte-
  identical output; `TestCommandsListedInDeclarationOrder` asserts EchoCmd appears before PingCmd
  in root help (declaration order); `TestFlagsListedInHelp` asserts both flags appear in
  subcommand help. Runtime suite + vet green.

### T12 — Global flags, custom subcommands, overrides

- **Outcome:** Implementors extend the app without touching generated code.
- **Verification:** Test adds `--endpoint`, adds a custom subcommand, and overrides a generated one.

### T13 — Pre/post execution hooks

- **Outcome:** Auth, logging and metrics can wrap execution.
- **Verification:** Hook order asserted; a pre-hook error aborts execution.

### T14 — Result renderers and exit codes

Default human renderer plus `--output=json`, driven by `ResultDesc`. Exit codes key on
`ErrorInfo.Code` **classes**, not HTTP statuses: 0 success; 2 usage (flag parse errors,
missing required flags); 3 not-found (`unknown_command`); 4 auth (`unauthorized`,
`forbidden`); 1 all other execution errors.

One wrinkle to document: `PackageHandler.executeCmd` collapses every runner error to
`command_error`, so classes 3 and 4 only surface for pre-dispatch failures (interceptor
rejections, unknown command names); the common in-command failure lands in class 1.

- **Outcome:** Results are printed instead of discarded; failures are scriptable.
- **Verification:** Renderer and exit-code table tests.
- **Notes:** Fixes two concrete `go_cli_main.txt` defects.

### T15 — `gen_options.cli: runtime|legacy|none`

The mode is a **spec gen_option only**, read in `toolkit_runner.go` exactly like the existing
`format` genopt fallback — no new `mvep` CLI flag, and `ExecuteGenerate`'s signature is
unchanged. `skipCmd=true` still forces `none` regardless of the genopt. Absent genopt →
`legacy` until the flip, `runtime` after.

- **Outcome:** Mode is selectable; default flips to `runtime`.
- **Verification:** All three modes generate and compile.

### T16 — Dogfood mvep's own CLI

Rebuild `toolkit/mvepapi/cmd/mvep` on the library; drop the `EXPERIMENTAL` marker.

- **Outcome:** mvep's CLI is its own first consumer.
- **Verification:** `mvep generate` / `init` / `validate` behave identically to today.
- **Notes:** `mvep_main_cmd.go` carries `// NOMVGEN` and is safe to rewrite.

### T17 — Documentation

- **Outcome:** `runtime/go/mvep/cli/README.md`, descriptor notes in `runtime/go/README.md`, and
  a migration note in `docs/`.
- **Verification:** Doc examples compile as tests.
