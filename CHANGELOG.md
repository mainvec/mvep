# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Maintainer note:** when adding a **BREAKING** entry for `runtime/go`,
> `runtime/ts`, or `toolkit`, also review `toolkit/MVEP_SKILL.md` and the
> `mvep-codegen` Copilot skill (`~/.mainvec/skills/mvep-codegen/`, especially
> `references/generated-patterns.md`) for staleness.

## [Unreleased] - 2026-08-12 (plan 041 follow-up, #52–#54)

### Fixed — runtime/go
- **`mvep exec` renders its result.** The payload path was routed through the
  execution core without rendering, so `mvep exec <cmd>` printed nothing on
  success and `--mvep-output json` was useless on the machine surface. `exec`
  now goes through the same rendering tail as the flag path, so both output
  modes are byte-identical between flag-driven and payload-driven invocation.
  `mvep send` still emits envelopes only.
- **`--input -` reads stdin unconditionally.** It previously errored whenever
  stdin was a pipe — the only realistic way to use it — because the explicit
  `-` and the implicit pipe were misread as two competing consumers. They are
  the same single consumer, so `cat p.json | svc mvep exec --input - <cmd>`
  now works.
- **`mvep send` flushes each `CmdResp` per record.** Responses were buffered
  and written once at EOF, so nothing was readable in a live pipeline until the
  input closed. Each response is now written and flushed as its record is
  processed.

### Fixed — toolkit
- **Reserved-name check now covers group aliases.** A root-level group whose
  alias resolves to `mvep` passed generation and would panic at `cli.New`; it
  now fails `mvep generate` naming the reserved word, matching the command-leaf
  check. Case handling is unchanged and correct: the runtime resolves names
  case-sensitively, so `MVEP` is distinct from `mvep`.

## [runtime/go v0.12.0] - 2026-08-12 (plan 041, #49)

### Added — runtime/go
- **Reserved `mvep` namespace.** Every generated CLI reserves a single `mvep`
  group (`svc mvep <verb>`) hosting a spec-independent, machine-readable
  surface. The namespace name is overridable via
  `cli.New(desc, executor, cli.WithNamespace("acme"))`.
- **`mvep exec`** reads a complete command payload from `--input <path>`,
  `--input -`, or implicitly from stdin when stdin is not a terminal. Payload
  keys are validated against the descriptor (unknown keys, including nested
  record fields, hard-error), then decoded with the same encoder registry the
  server uses. Flags precede the command name (stdlib `flag` semantics).
- **`mvep send`** reads a stream of `CmdReq` envelopes (NDJSON or concatenated)
  and emits one `CmdResp` per record, flushing immediately for live pipelines.
  Continue-on-error by default; `--fail-fast` halts; non-zero exit if any
  record errored. Request headers ride the context, so interceptors behave
  identically under the CLI and over HTTP, and response headers round-trip.
- **`mvep list`** and **`mvep describe`**. `list` prints command names (a JSON
  array under `--mvep-output json`). `describe [command]` emits a versioned
  JSON projection (name, alias, group, description, fields, result).
- **`--mvep-output json|text`** persistent flag. Under `json`, results render
  as JSON on stdout and errors serialize as `{"error":...}` on stdout (never
  stderr), shaped exactly like a `send` record's `CmdResp.Error`; exit codes
  are unchanged. The flag name follows the configured namespace.
- **Per-field `-file` hatches.** `--<name>-file` and
  `--<record>-<field>-file` load JSON values from files for maps, repeated
  non-string, and record fields, with `-file > -json > flattened sub-flags`
  precedence at both nesting levels.

## [runtime/go v0.11.1] - 2026-08-12 (plan 041 T1, #49)

### Fixed — runtime/go
- **Repeated record sub-fields bind correctly.** A `repeated string` record
  sub-field now binds as a repeatable flag (`--<record>-<field> 'a'
  --<record>-<field> 'b'`) instead of a single string that failed to unmarshal
  into the record's `[]string`, making any record with a repeated field
  reachable from the CLI. Every other repeated sub-field type (UUID,
  timestamp, duration, bytes, numeric, bool, map, `recRef`) binds via
  `--<record>-<field>-json` as a JSON array, matching the top-level
  `registerRepeatedFlag` fallback exactly. Malformed or non-array `-json`
  values error naming the sub-field flag, not the parent record flag.
  Sub-field and top-level binding now agree on every `FieldType`.

## [toolkit v0.10.0] - 2026-08-12 (plan 041 T9, #49)

### Added — toolkit
- **Generate-time reserved-name validation.** A spec declaring a top-level
  command or group named `mvep` fails `mvep generate` with an error naming the
  reserved word, catching the runtime namespace collision before generation.

## [Unreleased] - 2026-08-11 (plan 040, #40)

### Fixed — toolkit
- **`mvep` CLI commands implemented.** `mvep generate`, `mvep init`, and
  `mvep validate` were "command not implemented" stubs; they now delegate to
  the real toolkit functions. `mvep init` scaffolds a `<name>.jsonc` spec with
  a couple of dummy commands (`PingCmd`, `StatusCmd`) as a valid, editable
  starting point. The impl file is `// NOMVEP`-guarded so regeneration cannot
  clobber it.

### Added — spec
- **Command groups.** A command's optional `group` field (a `/`-separated
  path) places it under a nested CLI subcommand, so `"group": "server"` with
  `"alias": "start"` yields `svc server start`. Group metadata (title,
  description, aliases, hidden) lives in the optional top-level
  `commandGroups` object, keyed by full path; a group referenced by a command
  but absent there is auto-created with the path segment as its name.
  Intermediate segments are auto-created too, and may carry their own
  metadata. Additive optional properties — a spec with no `group` generates
  byte-identical output.

### Added — runtime/go
- **Group descriptor types.** `mvep.GroupDesc` (flat, ordered, carrying full
  path), `CommandDesc.Group`, and `PackageDesc.Groups`.
- **`mvep/cli` group support.** `cli.New` builds the nested tree from
  `desc.Groups`; group parents carry their own title/description/aliases/
  hidden flag, have no `RunE` (so ugo prints their help), and share the root's
  unknown-subcommand guard. `--help`, dispatch, aliases, hidden, and
  persistent-flag inheritance all work through groups with no `ugo` change.

### Added — toolkit
- **Generate-time validation** (`validateCommandGroups`) rejects group/command
  name and alias collisions (including at depth), duplicate command names
  under a parent, malformed group paths, and unreferenced `commandGroups`
  entries, naming the spec path and the colliding command.
- **Descriptor emission** of `Groups` and per-command `Group`, auto-creating
  intermediates so the descriptor is complete on its own. A no-group spec
  still generates byte-identical output.

### Release ordering
`runtime/go v0.11.0` ships `GroupDesc` / `CommandDesc.Group` /
`PackageDesc.Groups`, and `toolkit/go.mod` is bumped to it, before this
toolkit release.

## [Unreleased] - 2026-08-10 (plan 025, #25)

### Added — runtime/go
- **Package descriptor.** A complete runtime description of a generated
  package (`PackageDesc`, `CommandDesc`, `FieldDesc`, `RecordDesc`,
  `FieldType`) emitted by codegen into `mvep_package.go`. `FieldDesc.Ptr`
  closes over a real struct field, so a codegen mistake is a compile error,
  not a silent runtime drop. `NewPackageFromDesc` derives `InstanceOf`/
  `NameOf`/`CommandNames` from the descriptor, satisfying `Package`,
  `CommandLister`, and `PackageDescriber`. `PackageDesc.Record(name)`
  resolves name-only `$ref` fields to the full record. Build-time inputs
  (`GenOpts`, `ProtocOpts`) are deliberately excluded.
- **CLI builder (`mvep/cli`).** A descriptor-driven CLI library that builds
  a ugo command tree, binds flags to struct fields via `Ptr` (all 14
  `FieldType`s, repeated, map, depth-1 record flattening), enforces required
  flags, dispatches via `Executor` (local or remote), and classifies exit
  codes. Extension surface: `App.Root()` (global flags, custom subcommands,
  overrides), `AddPreHook`/`AddPostHook` (auth, logging), `SetRenderer`
  (JSON output). See `runtime/go/mvep/cli/README.md`.
- **`Executor` interface** with `LocalExecutor` (in-process) and
  `cliclient.RemoteExecutor` (remote via `SendCmdReq`, wrapping
  `CmdResp.Error.Code` in `*cli.ErrorCode` for exit-code classification).
- **`ExitCode(err) int`** mapping: 0 success, 2 usage, 3 not-found, 4 auth,
  1 other.
- **Custom `Uint32Var` / `Float32Var`** flag.Value types (ugo v0.7.0 ships
  neither).
- **`ugo` bumped v0.6.0 → v0.7.0** (persistent flags, `RunE`, `Int32Var`,
  `StringSliceVar`, `BytesVar`).

### Added — toolkit
- **Descriptor emission.** Codegen emits a `PackageDesc` literal into every
  generated `mvep_package.go` via ordered iterators (`SortedCommandNames`,
  `SortFieldsByFnum`). `NewPackage`/`InstanceOf`/`NameOf` delegate to
  `NewPackageFromDesc`; the three hand-shaped switch statements are deleted.
- **`gen_options.cli: runtime|legacy|none`.** The CLI mode is a spec
  gen_option (default `runtime`). `runtime` emits a descriptor-driven
  `cli.New` main; `legacy` emits the old hand-wired pattern; `none` skips.
  `skipCmd=true` forces `none`.
- **Generate-time hard error** (`validateDescriptorRepresentable`) on
  constructs the descriptor cannot represent, naming the offending command
  and field.
- **`CmdDescOrTitle`** template helper: emits `title` into the descriptor's
  `Desc` when `desc` is absent.
- **`isRequiredField` reconciled** with `fieldIsRequired` (tag-derived) so
  Go and JS/TS output agree on required-ness.
- **mvep's own CLI dogfooded** on the descriptor-driven `cli.New` path.

### Changed — runtime/go
- **`mvep.Package` methods are derived** from the descriptor via
  `NewPackageFromDesc` rather than emitted as switch statements. Additive —
  hand-written `Package` implementations keep working. `GetName()` still
  returns `Name + "Package"` to preserve HTTP routes.
- **Newly satisfying `CommandLister`** activates duplicate-command detection
  in multi-package clients. A client whose command names collide now errors
  at registration where it previously succeeded silently.

### Documentation
- `runtime/go/mvep/cli/README.md` — CLI builder guide
- `runtime/go/README.md` — descriptor and CLI sections added
- `docs/cli-builder-migration.md` — migration from legacy to runtime CLI

### Release ordering
`runtime/go` must be tagged (e.g. `v0.10.0`) with the descriptor + `mvep/cli`
types, and `toolkit/go.mod` bumped to it, **before** a toolkit release ships
the `runtime` CLI default. The generated `main.go` imports `mvep/cli`, which
does not exist in the published `runtime/go v0.9.0`.

## [Unreleased] - 2026-07-31 (plan 020, #21)

### Added
- **(runtime/go) Async job execution.** An opt-in, runtime-only mechanism for
  running any existing command as a background job. Off by default
  (`ServerConfig.EnableAsyncJobs` / `PackageHandler.EnableAsyncJobs`); no spec,
  codegen, or toolkit change required.
  - Reserved built-in commands `SubmitJob` and `GetJobStatus`, recognized
    inside `PackageHandler.ServeCmdReq` before normal dispatch. Both go
    through the full interceptor chain (auth applies).
  - Encoder-independent wire model: the wrapped command travels as opaque
    `CmdReq.Payload` bytes; job metadata travels in `x-mvep-*` headers. Works
    with JSON, protobuf, protojson, and any future encoder.
  - `GET /<BasePath>/<pkg>/jobs/{jobId}` convenience route with identical auth
    posture to `GetJobStatus` (delegates through `ServeCmdReq`).
  - A failed job returns HTTP 200 with `x-mvep-job-status: failed` and
    `x-mvep-job-error-code`/`x-mvep-job-error-message` headers — not a
    `CmdResp.Error` — so a failed job is distinguishable from a failed poll.
  - Pluggable `JobStore` interface with an in-memory default (single-instance
    only). `MaxConcurrentJobs`, `MaxJobResultBytes`, `JobRetention`,
    `JobTimeout` bound resource usage.
  - `PackageHandler.Shutdown(ctx)` drains in-flight jobs, bounded by the
    shutdown context; wired into `Server.drain`.
  - Go client: `PackageClient.SendEnvelope`, `SubmitJob`, `GetJobStatus`,
    `WaitForJob`.
  - New `HTTPStatusForErrorCode` cases: `job_not_found`→404,
    `job_queue_full`→429, `nested_job_forbidden`→400, `job_store_error`→500.
  - **Caveat: submitter credentials are stored at rest.** The `auth` token is
    replayed when the inner command runs, so bearer tokens live in the job
    store for the job's lifetime plus `JobRetention`. See `SERVER.md`.

## [Unreleased] - 2026-07-28 (plan 018)

### Changed
- **BREAKING (runtime/go): command endpoint HTTP semantics.** `PackageHandler.ServeHTTP`
  now enforces real HTTP semantics instead of treating HTTP as a dumb byte pipe.
  Runtime targets **v0.9.0**. See `runtime/go/mvep/server/SERVER.md`.
  - Non-`POST` requests return `405 method_not_allowed`; the runner is never invoked.
  - `Content-Type` is parsed as a media type, so `application/json; charset=utf-8`
    resolves the JSON encoder; unregistered types return `415`.
  - Outcomes map to meaningful statuses via a new exported
    `HTTPStatusForErrorCode`: `401`/`403`/`404`/`405`/`413`/`415`/`400`/`500`.
    The stable machine-readable code is in the new `x-mainvec-error-code` header.
  - **Error redaction is now the default.** Handler error detail is logged
    server-side with the request id; the response carries a generic message.
    `ServerConfig.VerboseErrors` / `PackageHandler.VerboseErrors` restores the
    old reflective behavior for local development.
  - Request bodies are bounded by `ServerConfig.MaxRequestBytes` (default 4 MiB);
    oversized bodies return `413 payload_too_large`. Each listener sets
    `http.Server.MaxHeaderBytes` (1 MiB) alongside the existing `ReadHeaderTimeout`.
- **BREAKING (runtime/go): CORS is an explicit allowlist.** `EnableCORS` no longer
  emits `Access-Control-Allow-Origin: *`. With `AllowedOrigins` empty it emits no
  CORS headers and warns at startup (fail closed); an allowed origin is echoed
  with `Vary: Origin`, the advertised methods drop `PUT`/`DELETE`, and
  `Access-Control-Allow-Headers` enumerates the headers MVEP actually uses.
- **BREAKING (runtime/go): `LocalTrustMiddleware` verifies the peer.** A request
  is trusted only from a Unix socket or a loopback TCP address; anything else is
  passed through untrusted and logged. A listener exposed to the network now
  fails closed instead of silently bypassing `AuthInterceptor`. Go runtime
  dependency `ugo` bumped to v0.6.0 for `application/json` encoder registration.
- **runtime/ts: license corrected to Apache-2.0** to match the repo root and Go
  runtime (was MIT; an oversight from the monorepo consolidation). Targets
  `@mainvec/mvep@0.8.0`.

### Added
- `runtime/go`: `HTTPStatusForErrorCode` exported status mapper.
- `runtime/go`: `ServerConfig.AllowedOrigins`, `MaxRequestBytes`, `VerboseErrors`;
  matching `PackageHandler` fields.
- `runtime/go`: `CommandLister` optional interface for deterministic client-side
  command indexing.
- `runtime/ts`: `ClientConfig.timeout` is now enforced — `HttpTransporter` aborts
  the underlying fetch via `AbortController` when it elapses.

### Fixed
- `runtime/go`: client and server package registries are now guarded by mutexes;
  `Client.SendCmd` resolves deterministically (sorted package order) and a
  duplicate command registration returns an explicit error instead of shadowing.
- `runtime/go`: legacy transport path honors `context.Context`
  (`http.NewRequestWithContext`), no longer leaks the response body on the
  non-200 branch, dials Unix sockets with `DialContext`, and default transporters
  have a 30s timeout instead of none. `SendCmdReq` copies the caller's header map
  instead of aliasing it. `RequestIDInterceptor` is nil-response safe.
- `runtime/go`: `SetResponseHeader` is now wired through `executeCmd` (a `CmdResp`
  is seeded into the handler context), so the documented feature actually works.
- `runtime/ts`: `timeoutInterceptor` clears its timer (no leaked fetch/timer);
  request ids use `crypto.randomUUID()` instead of `Math.random()`.
- `runtime/ts`: response body reads are bounded and oversized responses error
  rather than buffering unboundedly.

## [Unreleased] - 2026-07-28

### Changed
- **BREAKING (runtime/go): caller-owned server lifecycle.** `mvep/server` no longer installs process signal handlers. `os/signal`, `syscall`, and all `SIGINT`/`SIGTERM` handling were removed: the owning application decides which signals matter, how long shutdown may take, and how HTTP draining is sequenced with its own cleanup. Go runtime targets **v0.8.0**. See [docs/server-lifecycle-migration.md](docs/server-lifecycle-migration.md).
  - `ServerConfig.OnShutdown` **removed**. Cleanup callbacks hide ordering and can recurse into `Shutdown`; sequence cleanup explicitly around `ShutdownContext`.
  - `Start()` remains blocking but now returns on explicit shutdown or a fatal serve error instead of a process signal.
  - `Shutdown()` now genuinely drains active requests (bounded by `DefaultShutdownTimeout`) and unblocks `Start()`. Previously it only closed listeners.
  - `NewServer` copies `ServerConfig` instead of retaining and mutating the caller's pointer. Set all fields, including `Listeners`, before calling it.
  - A `Server` is single-use: `Start`/`StartAsync` return `ErrServerStarted` while running and `ErrServerStopped` once stopped.
  - Listeners are owned by the server once start begins and binding is transactional: a failed bind closes every listener already acquired in that attempt.
  - An unexpected `Serve` error on any listener is recorded and shuts down the remaining listeners; a multi-listener server is one service lifecycle.
  - Each listener now runs on its own `http.Server` with a 10s `ReadHeaderTimeout` (Slowloris mitigation) instead of a bare `http.Serve` goroutine.

### Added
- `runtime/go`: `Server.StartAsync()` binds all listeners synchronously and returns when they are serving, replacing `go srv.Start()` plus `GetListener()` readiness polling.
- `runtime/go`: `Server.Wait()`, `Server.Done()`, and `Server.Err()` expose lifecycle completion and the final error without polling.
- `runtime/go`: `Server.ShutdownContext(ctx)` for context-aware graceful shutdown, force-closing remaining connections when the budget expires.
- `runtime/go`: `Server.GetListeners()` returns a defensive copy of every bound listener in configuration order.
- `runtime/go`: exported `DefaultShutdownTimeout` (30s), `ErrServerStarted`, and `ErrServerStopped`.
- `docs/server-lifecycle-migration.md`: v0.7 → v0.8 migration guide for daemon owners.

## [Unreleased] - 2026-07-22

### Changed
- **BREAKING (wire): HTTP custom-header prefix renamed** `x-mvp-` → `x-mvep-` in both runtimes (Go `HeaderPrefix`, TS `HEADER_PREFIX`). Client and server must both run runtime ≥ v0.7.0; a peer on the old prefix cannot exchange custom headers (auth, request-id) with a peer on the new one. The control headers `x-mainvec-cmd` / `x-mainvec-error` are unchanged. Runtime bumped to **v0.7.0**.
- **Monorepo consolidation.** Merged the `mvpgo` repository (Go + TypeScript runtimes) into this repo to form the MVEP (Mainvec Engineering Platform) monorepo. Layout is now `toolkit/` (generator) + `runtime/{go,ts}` (runtimes).
- **Generator renamed** `mvgen` → `toolkit` (module `github.com/mainvec/mvep/toolkit`, package `toolkit`).
- **Go runtime module path fixed/renamed** `github.com/mainvec/mvp/mvpgo` → `github.com/mainvec/mvep/runtime/go` (now resolvable: module path matches repo path). Go package `mvp` → `mvep`.
- **npm package renamed** `@mainvec/mvpjs` → `@mainvec/mvep`.
- **CLI renamed** `mvp` → `mvep` (`toolkit/mvepapi/cmd/mvep`).
- **Naming standardization** across the platform: output dir `mvpapi` → `mvepapi`, struct tag `mvp:` → `mvep:`, and "MVP" → "MVEP" in docs/spec labels.
- Generated Go source is now gofmt-normalized by the generator.

### Added
- `mvepspec/0.2` schema as the new canonical `$schema` URL, with `mvpspec/0.2` retained as a resolvable **alias** so existing spec files keep validating.
- Root `go.work` tying `toolkit` + `runtime/go` for local multi-module development.
- Merged CI: `go.yml` (toolkit + runtime/go), `js.yml` (runtime/ts), and `npm-publish.yml` (publishes `@mainvec/mvep` on `runtime/ts/v*` tags).
- `release.yml`: builds the `mvep` CLI for linux/darwin/windows (amd64/arm64) with the version injected via `-ldflags -X main.version`, verifies the stamp, and attaches binaries to a GitHub Release on `toolkit/v*` tags.
- Hybrid version resolution for the `mvep` CLI and for generated CLIs: linker-injected `main.version` → module version from `runtime/debug.ReadBuildInfo()` (`go install …@vX.Y.Z`) → static fallback (`"dev"` for `mvep`; the `<NAME>_VERSION` constant for generated CLIs). Replaces the previously hardcoded version.
- `NOMVEP` codegen protection marker (legacy `NOMVGEN`/`NOWOGEN` still honored).

### Fixed
- The CLI-main generation now honors the `NOMVEP`/`NOMVGEN`/`NOWOGEN` markers instead of unconditionally overwriting hand-customized entry points.
- Resolved the previously-unresolvable runtime module path (was `github.com/mainvec/mvp/mvpgo`).

### Versioning
- Hybrid scheme: the Go and TS runtimes share a version/tag (`runtime/go/vX.Y.Z`, `runtime/ts/vX.Y.Z`); the generator versions independently (`toolkit/vX.Y.Z`).

## [Unreleased] - 2026-07-21

### Changed
- Switched license from MPL-2.0 to Apache-2.0.
- Updated root `README.md` with project overview, install, quick start, and repository layout.
- Updated `mvgen/README.md` contributing and license sections for open-source release.
- Renamed `mvgen/gengen_next/` to `mvgen/gengen/` (now the sole active self-generator).

### Added
- GitHub Actions CI workflow (`.github/workflows/test.yml`) running `go test` and `go vet` on push/PR.
- `CHANGELOG.md`.
- `.claude/` and `.DS_Store` entries to `.gitignore` files.
- `// TODO: REWORK` notes to codegen templates that reference `github.com/mainvec/wo/` runtime packages, flagging them for a follow-up rework.

### Removed
- `mvn/` — orphaned WO Node runtime (no consumers, no `go.mod`, not importable).
- `mvgen/gengen/` (legacy) — old self-generator producing `mvpapi_legacy/`.
- `mvgen/mvpapi_legacy/` — old protobuf-based implementation and `mvgen` CLI.
- `mvgen/testold/` — abandoned test fixtures.
- `mvgen/testdata/draft-07/` — old draft-07 schema test fixtures.
- `.gitmodules` (empty), `dev-roadmap.txt`, `mvgen/.claude/` (local settings).
- Skipped `TestGenerateGOSRV` in `mvgen/mvgen_go_test.go` (referenced removed `07_pizzahub.jsonc`).
- Stale `wog/` and `docs/dev-docs/` entries from `.gitignore` files.

### Known limitations
- `go.mod` retains a `replace github.com/mainvec/mvp/mvpgo => ../../mvpgo/mvpgo` directive. The `github.com/mainvec/mvp/mvpgo` module path is currently unresolvable externally (no `github.com/mainvec/mvp` repository, no go-get meta tags served). Publishing `mvpgo` as a fetchable module is tracked as a separate follow-up before the public release.
- Codegen templates for server/starter/test code still reference `github.com/mainvec/wo/` packages. Full rework is deferred to a follow-up issue; templates are marked with `// TODO: REWORK` notes.