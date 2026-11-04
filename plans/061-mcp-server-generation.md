# 061 — Generate MCP server for any MVEP package

**GitHub Issue**: [#65](https://github.com/mainvec/mvep/issues/65)

- **Status:** Draft (plan only — no implementation started)
- **Date:** 2026-08-13 (revised 2026-08-13 after code fact-check)
- **Scope:** Add an MCP server surface that exposes any MVEP package's commands as MCP tools.

## Summary

The MVEP framework is spec-driven: a JSON spec → `mvep generate` → typed command structs,
a runtime `mvep.PackageDesc` descriptor, and a `CommandRunner` dispatcher. Every command,
field, type, `Required`-ness, and description is already machine-readable in the descriptor.
This means an MCP server — which is essentially a `tools/list` (schema) + `tools/call`
(dispatch) server — maps cleanly onto existing abstractions.

No MCP code, templates, `gen_options` values, or dependencies exist anywhere in the
workspace yet (verified).

## Effort estimate

| Scope | Estimate |
| --- | --- |
| v1: runtime `mcp` package only, hand-written main (stdio, nested records) | **1–1.5 days** |
| v2: codegen target + end-to-end spawn tests | **+1–2 days** |

Roughly ~70% of the work is already provided by the framework (descriptor + decoder +
dispatcher). The genuinely new work is (a) an MCP JSON-RPC transport, (b) a `PackageDesc`
→ tool-schema mapper, and (c) `tools/call` glue.

## Decisions

- **Ship the runtime library first; defer codegen.** v1 is `runtime/go/mvep/mcp` plus a
  ~15-line hand-written main that builds a `mvep.PackageHandler` around the generated
  package and passes it to `mcp.NewServer(api.Describe(), handler)`. This validates the API
  before committing to a generation target and sidesteps the `gen_options` / `skipCmd` /
  output-path issues listed in Phase C. Codegen lands only after the API has settled.
- **Transport:** stdio-first behind a clean `Transport` interface. HTTP/SSE can be added
  later behind the same core without reworking the server.
- **Language:** Go first. TS surface deferred; the TS client is already thin.
- **Dispatch: route through `PackageHandler.ServeCmdReq`, not `CommandRunner.RunCmd`
  directly.** Calling `RunCmd` directly bypasses the interceptor chain — including
  `RecoveryInterceptor`, so a panicking handler would kill the stdio process — plus auth
  and structured `ErrorInfo` error codes.
- **Arg binding: reuse the existing JSON decode path, do not re-implement it.**
  `executeCmd` (`InstanceOf` → `oenc.Decode`) and `(*cli.App).decode` (`cmdDesc.New()` +
  `validatePayloadKeys` + `oenc.Decode`) already handle nested records and maps. Do **not**
  hand-roll a type switch over `FieldDesc.Ptr` — `bindFlags` is the wrong analogue, since
  MCP delivers a JSON object, not flag strings.
- **Streaming:** no streaming job results in v1 — see Open questions.

## Open questions (resolve before implementation)

1. **Hand-roll JSON-RPC vs `github.com/modelcontextprotocol/go-sdk`?** This plan assumes a
   minimal hand-rolled JSON-RPC 2.0 framer. The SDK buys spec compliance and future
   transports at the cost of a dependency in `runtime/go`. **Decide first — it materially
   changes Phase A.**
2. **Tool naming.** `Package.InstanceOf` and the `mvep exec/send/list/describe` machine
   surface key commands by the descriptor **`Name`** (e.g. `"GenerateCmd"`) —
   `app.commands[cmdDesc.Name] = cmdDesc` in `cli/app.go`. `commandName()`
   (alias-or-snake_case, and it lives in `cli/app.go`, not `describe.go`) is only the
   human-facing CLI leaf. MCP tool names must map back to `Name` for `InstanceOf`; pick one
   scheme deliberately and keep a name→`Name` map if it is not `Name`.
3. **Group flattening.** `PackageDesc.Groups` exists and commands carry `Group` paths, but
   MCP tool names are flat. Needs an explicit flattening rule (e.g. `group_sub_command`)
   plus a collision check.
4. **Async jobs are header-driven.** `SubmitJob` / `GetJobStatus` communicate entirely via
   request/response headers (`job-id`, `job-cmd`, `job-status`, `job-progress-*`) — see
   `runtime/go/mvep/job.go`. MCP has no header channel, so excluding jobs from v1 is a
   structural constraint, not a preference. A future version needs an explicit substitute
   (e.g. dedicated `submit_job` / `get_job_status` tools).
5. **Schema fidelity** for `Timestamp` / `Duration` / `UUID` — recommend pragmatic `string`
   with `format` hints over strict object types in v1.

## Existing abstractions to reuse (do NOT reinvent)

| Abstraction | Location | MCP role |
| --- | --- | --- |
| `mvep.PackageDesc` / `CommandDesc` / `FieldDesc` / `ResultDesc` / `RecordDesc` | `runtime/go/mvep/descriptor.go` | Source for `tools/list` schema |
| `PackageDescriber` (`Describe() *PackageDesc`) | `runtime/go/mvep/descriptor.go` | Descriptor access — **not** a method on `mvep.Package` |
| `FieldType` enum (14 values: `FieldString/Bool/Int32/Int64/Uint32/Sint32/Float/Double/Bytes/Timestamp/Duration/UUID/Map/Record`) | `runtime/go/mvep/descriptor.go` | Source for JSON Schema types |
| `PackageDesc.Record(name) (*RecordDesc, bool)` | `runtime/go/mvep/descriptor.go` | Resolves name-only `FieldDesc.Ref` |
| `PackageHandler.ServeCmdReq` → `executeCmd` (`InstanceOf` → `oenc.Decode` → `RunCmd` → encode) | `runtime/go/mvep/mvepackge.go` | `tools/call` dispatcher |
| `CmdInterceptor` / `Chain`; `RecoveryInterceptor`, `AuthInterceptor` | `runtime/go/mvep/middleware.go`, `interceptors.go` | Panic recovery + auth for free |
| `(*cli.App).decode` + `validatePayloadKeys` | `runtime/go/mvep/cli/app.go`, `cli/exec.go` | Pattern ref for strict JSON→cmd binding |
| `fieldTypeName(FieldType) string` | `runtime/go/mvep/cli/describe.go` | Type→spec-name mapping to adapt |
| `cli.New(desc, executor)` | `runtime/go/mvep/cli/app.go` | Pattern ref for a descriptor-driven surface |
| `toolkit/mvepapi/cmd/mvep/mvep_main_cmd.go` | toolkit | **Correct** reference shape for a generated main |

### Descriptor facts that constrain the schema mapper

- `FieldDesc` fields are `Name, Alias, Desc, Fnum, Type, Repeated, Required, Tags, Ptr, Ref`.
  There is **no** `RecordRef` and **no** `Description` (it is `Desc`).
- `FieldDesc.Ref` is emitted **name-only** by codegen (`&mvep.RecordDesc{Name: "..."}` with
  empty `Fields`) — always resolve via `desc.Record(f.Ref.Name)`. Recursion needs a
  visited-set for self-referential records.
- **`FieldMap` carries no key/value type information.** Map value types are not derivable
  from the descriptor, so v1 must emit a permissive `object` schema — matching the CLI,
  which treats maps as raw JSON via `--<name>-json`.
- `CommandDesc.New` and `FieldDesc.Ptr` are **struct fields, not methods, and may be nil**
  (`NewPackageFromDesc` explicitly guards `newFn == nil`). Nil-check before use.
- `describeProjection` is a **type**, not a function; the builder is `projectCommand`. It
  drops field `Desc` and does not expand nested records, so it is **not** reusable as a JSON
  Schema source — only `fieldTypeName` is worth adapting (note it rebuilds its map on every
  call; hoist it if used per-field in `tools/list`).

## Steps

### Phase A — Core Go MCP package (`runtime/go/mvep/mcp/`, new)

1. **`tool.go`** — `Tool` type: name, description, `inputSchema`.
2. **`schema.go`** — `SchemasFromPackage(desc *mvep.PackageDesc) ([]Tool, error)`:
   - Iterate `desc.Commands[]` → one tool per command; description from `CommandDesc.Desc`.
   - Map `FieldDesc.Type` → JSON Schema type; field description from `FieldDesc.Desc`.
   - Handle `Required` (→ `required[]`), `Repeated` (→ `array`), `FieldMap` (→ permissive
     `object`), `FieldRecord` (→ nested object resolved via `desc.Record(f.Ref.Name)`, with
     a visited-set guard).
   - Return an error for unmapped `FieldType` values rather than emitting a broken schema.
3. **`server.go`** — `NewServer(desc *mvep.PackageDesc, handler *mvep.PackageHandler) *Server`
   implementing `initialize`, `tools/list`, `tools/call`, `ping`. The signature takes
   `*PackageDesc` explicitly because `mvep.Package` does **not** expose `Describe()`.
4. **`transport.go`** — `Transport` interface (send/read messages) + `stdioTransport`
   reading/writing JSON-RPC 2.0 frames on `os.Stdin` / `os.Stdout`.
   **Constraint:** nothing else may write to stdout; route all logging to stderr.
5. **`tools/call` glue** — tool name → descriptor `Name` → build a `mvep.CmdReq` carrying the
   raw args JSON as payload → `handler.ServeCmdReq(ctx, req, "application/json")` → map the
   `CmdResp` payload / `ErrorInfo` onto an MCP tool result (`isError` on failure).

### Phase B — Tests + docs for the runtime package (depends on A)

6. **Unit tests:** schema mapper against `toolkit/testdata` specs — scalar types, `Required`,
   `Repeated`, nested records, maps, and the unsupported-type error path.
7. **In-process round-trip:** build a `PackageDesc` + `PackageHandler` (mirroring the
   hand-written-descriptor pattern in `cli_integration_test.go`) and drive `initialize` /
   `tools/list` / `tools/call` through the `Transport` interface over in-memory pipes — no
   subprocess needed.
8. **Docs:** `runtime/go/mvep/mcp/README.md` showing the hand-written main;
   `CHANGELOG.md` entry.

### Phase C — Toolkit codegen target (deferred; only after A/B settle)

9. **New gen_option `mcp`, not a `cli` value.** `gen_options.cli` is a single-valued mode
   switch (`none` / `runtime` / default→legacy) in `toolkit_runner.go`; reusing it would make
   MCP **mutually exclusive with the CLI**. Add an independent option (e.g. `mcp: "stdio"`).
   No JSON-schema change is needed — `gen_options` is an open `string→string` map
   (`patternProperties`, `additionalProperties: false`) and `GenOptsDef` is
   `omap.OMap[string, string]`.
10. **Emit outside the `if !skipCmd` block.** The existing CLI-main switch is gated on
    `!skipCmd`; placing MCP there means `--skip-cmd` silently disables it with no diagnostic.
11. **Separate output path.** Both existing CLI modes write
    `<outdir>/cmd/<name>/<name>_main_cmd.go`. The MCP main needs its own directory (e.g.
    `cmd/<name>-mcp/<name>_mcp_main.go`) — two `func main` cannot share a package — and
    either its own version file or no `{{.VERSIONCONST}}` reference.
12. **New template `go_mcp_main.txt`.** Model it on the hand-written
    `toolkit/mvepapi/cmd/mvep/mvep_main_cmd.go`, **not** on `go_cli_runtime_main.txt`, which
    is unverified and appears broken (duplicate `cli` import name; `debug.ReadBuildInfo()`
    with no `runtime/debug` import; calls `NewPackage()`/`NewCommandRunner()` against
    `.GOIMPORT` when they are emitted into the `api` package). It is never compile-tested,
    because `TestGenerateCompilePlain` uses `skipCmd=true` and `formatGoSource` only warns.
    - The main needs **both** imports (`.GOIMPORT` and `.GOAPIIMPORT`, as `go_cli_main.txt`
      does): `Describe()` comes from the `api` package, while the wired `GetCommandRunner()`
      comes from the **root** package (`go_commands_runner_code.txt`).
      `api.NewCommandRunner()` returns an **empty** struct with nil handler funcs and would
      panic on the first `tools/call`.
    - Add a `// NOMVEP` protection hint, consistent with `go_cli_version.txt`.
13. **Extend the pb3 guard.** The check in `toolkit_runner.go` currently rejects only
    `cliMode == "runtime"`; MCP consumes the same plain-JSON descriptor path and must be
    rejected under `format == "pb3"` too.
14. **Reject unknown option values.** The `cli` switch's `default:`→legacy behaviour means
    typos pass silently today — do not repeat that for the new option.
15. **Codegen tests:** follow the `cli_mode_test.go` pattern (inline JSONC spec → tempdir →
    `ExecuteGenerate` → `strings.Contains` assertions; negative cases assert `os.IsNotExist`).
    An end-to-end **spawn** test is new infrastructure: it needs the `go.mod` +
    `replace github.com/mainvec/mvep/runtime/go => ../runtime/go` + `go mod tidy` +
    `go build` scaffolding from `TestGenerateCompilePlain`, and must be `-short`/network
    gated the same way. No existing test spawns a generated binary, and there are no golden
    files.

### Not needed (verified)

- **No toolkit funcMap or `goFieldTypeEnum` changes.** Type → JSON Schema mapping belongs in
  the runtime `mcp` package, resolved from `PackageDesc` at runtime — exactly like
  `cli.New`. (For reference: funcMap is at `toolkit.go:536`, `goFieldTypeEnum` at `:910`.)
- **No new generate-time field-type gate.** `validateDescriptorRepresentable`
  (`descriptorSupportedFieldTypes`, `toolkit.go:949`) already runs unconditionally as the
  first step of `executeGenerateGo` for all modes. The real coverage gap is in the runtime
  schema mapper (step 2), not in codegen.

## Key files

- **Reuse (no edits):** `runtime/go/mvep/descriptor.go`, `mvepackge.go`, `middleware.go`,
  `interceptors.go`, `job.go` (constraint ref), `cli/describe.go`, `cli/app.go`,
  `cli/exec.go` (pattern refs).
- **New (Phase A/B):** `runtime/go/mvep/mcp/{tool,schema,server,transport}.go` + tests +
  `README.md`.
- **Edit (Phase C only):** `toolkit/toolkit_runner.go` (new option branch, pb3 guard,
  unknown-value error); new `toolkit/resources/codegen_templates/go/go_mcp_main.txt`.

## Verification

1. `go build ./...` and `go vet ./...` in `runtime/go`.
2. Unit test: `SchemasFromPackage` output matches expected JSON Schema for a testdata
   package (scalars, required, repeated, nested records, maps) and errors on unsupported
   types.
3. In-process `initialize` / `tools/list` / `tools/call` round-trip over piped transports;
   assert the results **and** that a panicking handler surfaces as an MCP error rather than
   killing the process (proves the `PackageHandler` routing).
4. Manual: register the hand-written main with an MCP client (Claude Desktop / VS Code) and
   invoke a command.
5. Phase C only: generated-main string assertions, plus a network-gated build+spawn test.
