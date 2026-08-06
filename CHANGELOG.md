# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Maintainer note:** when adding a **BREAKING** entry for `runtime/go`,
> `runtime/ts`, or `toolkit`, also review `toolkit/MVEP_SKILL.md` and the
> `mvep-codegen` Copilot skill (`~/.mainvec/skills/mvep-codegen/`, especially
> `references/generated-patterns.md`) for staleness.

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