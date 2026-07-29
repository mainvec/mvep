# Plan 018 — Harden MVEP runtime HTTP transport and repair the toolkit

- **Issue**: [#18](https://github.com/mainvec/mvep/issues/18) — fix(runtime): harden HTTP transport security and correctness
- **Branch**: `fix/018-runtime-hardening`
- **Target release**: `runtime/go/v0.9.0`, `@mainvec/mvep@0.8.0`, `toolkit/v0.6.0`

## Problem / Goal

The v0.8.0 lifecycle work brought `mvep/server` close to production quality, but
the transport and envelope layer beneath it has not received the same scrutiny.
A review of `runtime/go` and `runtime/ts` found a cluster of defects that share a
single root cause: `PackageHandler.ServeHTTP` treats HTTP as a dumb byte pipe.
It does not inspect the request method, does not parse the `Content-Type` media
type, does not bound the request body, maps every outcome to one status code, and
reflects raw internal error strings back to the caller.

Separately, both the Go client and server mutate their package registries without
synchronization, the legacy transport path discards the caller's `context.Context`,
and the TypeScript client documents a request timeout that is never applied.

None of these are regressions. All predate v0.8.0. Together they mean an MVEP
endpoint cannot currently be exposed to an untrusted network or to browsers
without an external protective layer.

A second cluster of defects lives in the toolkit, and it is worse than the
runtime cluster: **the generator does not work for anyone but its maintainer.**

- `toolkit/toolkit_runner.go` and `toolkit/toolkit_pb3.go` call `log.Fatal` /
  `log.Fatalf` from library code paths (~25 sites), so a generation error kills
  the host process instead of returning an error.
- 10 of the 22 Go codegen templates reference the private
  `github.com/mainvec/wo/*` modules, and the generated `go.mod` templates carry
  `replace` directives pointing at the maintainer's local disk
  (`/Users/hi/Development/mainvec/wo/wopcore`). Every server/client/test output
  the generator produces is non-compilable for any other user.
- Generated files are written with `os.ModePerm` (0777), so generated source is
  world-writable.
- No test compiles or golden-files the generated output, which is exactly why
  the broken templates have survived.
- `toolkit/go.mod` pins `runtime/go v0.6.0`, two minor versions behind HEAD and
  behind the `x-mvep-` header rename its own generated code depends on.

The goal is a transport layer that is safe to expose directly — correct HTTP
semantics, bounded resource consumption, an explicit CORS policy, no internal
detail leakage, and a race-free client and server registry — **and a generator
whose output compiles for any user, with errors returned instead of process
exits.**

## Goals

- Map command outcomes to meaningful HTTP status codes instead of a blanket
  `400`, so that auth failures are distinguishable from malformed requests by
  proxies, rate limiters, and log analysis.
- Reject non-`POST` requests to the command endpoint.
- Parse `Content-Type` as a media type so that `application/json; charset=utf-8`
  resolves the JSON encoder.
- Bound the request body on the server and the response body on the client, and
  set `MaxHeaderBytes`.
- Replace wildcard CORS with an explicit origin allowlist and stop advertising
  `Access-Control-Allow-Headers: *`.
- Stop reflecting raw handler error text to callers; return a stable error code
  and log the detail server-side.
- Give `LocalTrustMiddleware` an actual peer check rather than trusting every
  request on the listener.
- Make the Go client and server package registries safe for concurrent use.
- Honor `context.Context` throughout the legacy transport path and stop leaking
  response bodies on the error branch.
- Implement the documented TypeScript request timeout with `AbortController`.
- Resolve `SetResponseHeader`, which is currently documented but inert.
- Close the `ServeHTTP` test coverage gap, which is what allowed most of the
  above to survive.
- Make the toolkit return errors instead of calling `log.Fatal` from library
  code.
- Rework the 10 `wo/`-dependent Go templates to target the MVEP runtime so
  generated server, client, and test scaffolding compiles for any user.
- Write generated files with `0755`/`0644` instead of `os.ModePerm`.
- Add a compile test for generated Go output so broken templates fail CI.
- Bump `toolkit/go.mod` off `runtime/go v0.6.0`.

## Non-goals

- Changing the wire envelope, the `x-mainvec-cmd` selector, the `x-mvep-` header
  prefix, or the encoder registry design.
- Moving command selection from a header into the URL path. That is worth doing
  and is discussed in the Decision Log, but it is a protocol change and belongs
  in its own issue.
- Adding authentication mechanisms. `TokenValidator` remains caller-supplied.
- Streaming, batching, or chaining. Batching is tracked separately in #19.
- Spec parser and documentation drift (`oneOf`, `recDef`, `uuid`, `timestamp`,
  unenforced `tags: ["required"]`). Real, but a spec-semantics issue of its own.
- Renaming the `ExeucuteInitializeCmd` typo. It is exported API; bundle the
  rename with the next toolkit breaking release.
- Migrating downstream consumers. Droy, Linkvec, Girafa, and Mboxy are follow-up
  changes after the runtime is tagged.

## Proposed Design

### Error codes and status mapping

`executeCmd` already produces distinct failure reasons but collapses them at the
HTTP boundary. Introduce a stable code set on `ErrorInfo.Code` and a single
mapping function used by `ServeHTTP`:

| Condition | `ErrorInfo.Code` | HTTP status |
| --- | --- | --- |
| Command name not in package registry | `unknown_command` | 404 |
| Encoder not registered for media type | `unsupported_media_type` | 415 |
| Payload fails to decode | `decode_error` | 400 |
| Body exceeds `MaxRequestBytes` | `payload_too_large` | 413 |
| Method is not `POST` | `method_not_allowed` | 405 |
| Interceptor rejects credentials | `unauthorized` | 401 |
| Interceptor rejects an authorized caller | `forbidden` | 403 |
| Handler returns an error | `command_error` | 500 |

`AuthInterceptor` must set `unauthorized` explicitly so the mapping is driven by
the code rather than by string matching.

### Error redaction

`executeCmd` currently wraps handler failures as
`fmt.Sprintf("failed to run command: %v", err)` and `ServeHTTP` copies that
string into the `x-mainvec-error` response header and the body. Handler errors
routinely carry SQL fragments, file paths, and connection strings.

The response carries the stable code and a fixed generic message. The full error
is logged server-side with the request id. A `VerboseErrors bool` config field,
default `false`, restores the current behavior for local development.

This also removes a response-splitting footgun: unsanitized error text currently
flows into a header value.

### Body limits

Server: wrap `r.Body` in `http.MaxBytesReader` before `io.ReadAll`, bounded by
`ServerConfig.MaxRequestBytes` (default 4 MiB). Set `http.Server.MaxHeaderBytes`
alongside the existing `ReadHeaderTimeout` established in plan 016.

Client: cap the response read with `io.LimitReader` bounded by
`ClientConfig.MaxResponseBytes` (default 4 MiB).

### CORS

`EnableCORS` gains a companion `AllowedOrigins []string`. When `EnableCORS` is
true and `AllowedOrigins` is empty, no CORS headers are emitted and the server
logs a warning at startup. `Access-Control-Allow-Headers` enumerates the headers
MVEP actually uses (`Content-Type`, `x-mainvec-cmd`, and the `x-mvep-` metadata
headers in use) rather than `*`. The advertised method list drops `PUT` and
`DELETE`, which the handler does not implement.

### Local trust

`LocalTrustMiddleware` gains verification rather than assertion:

- Unix socket peers are accepted, optionally filtered by a UID allowlist via
  `SO_PEERCRED` where the platform supports it.
- TCP peers are accepted only when `RemoteAddr` is loopback.
- Anything else is not marked trusted, and the rejection is logged.

The middleware keeps working for its documented deployment while its failure
mode changes from "silently unauthenticated" to "not trusted".

### Concurrency

`Client.packages`, `Server.packages`, and the `PackageClient` encoder accessors
are guarded by a `sync.RWMutex`. `Client.SendCmd` currently resolves a command by
iterating the package map and calling `NameOf` on each entry, which is both O(n)
and nondeterministic when two packages claim the same command name. Replace it
with a command-name index built at registration time, and return an explicit
error on a duplicate registration rather than silently shadowing.

### Legacy path

`TransportCmd` builds its request with `http.NewRequest` rather than
`http.NewRequestWithContext`, and `ServeCmd` calls
`h.CommandRunner.RunCmd(context.Background(), cmd)`. Both silently discard
cancellation and deadlines. The non-200 branch of `TransportCmd` returns without
closing `resp.Body`. The Unix dialer calls `net.Dial` inside a `DialContext`.
`NewHttpTransporter` constructs an `http.Client` with no `Timeout`.

All five are fixed in place; the legacy byte-stream API keeps its signature.

### `SetResponseHeader`

`executeCmd` reads response headers back via `CmdRespFromContext(ctx)`, but no
server-side code ever seeds a `CmdResp` into the context, so the documented
feature is inert. Seed a `CmdResp` before invoking the runner and thread the
collected headers into the response.

### TypeScript

`ClientConfig.timeout` is defaulted to 30000 and stored but never read. Implement
it with `AbortController` in `http-transport.ts`. The existing opt-in
`timeoutInterceptor` uses `Promise.race`, which leaves the underlying fetch and
the `setTimeout` timer running; rework it to abort. Replace `Math.random()`
request-id generation with `crypto.randomUUID()`.

### Toolkit error handling

`toolkit/toolkit_runner.go` and `toolkit/toolkit_pb3.go` call `log.Fatal` /
`log.Fatalf` ~25 times inside generation paths. A library must never exit the
host process. Thread `error` returns up to the command boundary; the `mvep` CLI
`main` is the only place allowed to exit non-zero. Where a function already
returns an error, wrap and propagate; where it does not, change the signature —
all such functions are internal to the toolkit module.

### `wo/` template removal (revised from "rework")

The 10 templates under `resources/codegen_templates/go/` that reference
`github.com/mainvec/wo/*` (server, client, test, NATS starter, and their
`go.mod` generators) are **deleted**, along with the four dead generator
functions that emit them (`GenerateGOSRV`, `GenerateGOClient`, `GenerateGOMod`,
`GenerateGOAPI`). Verification during implementation confirmed none of them have
any caller in the module, so they are unreachable. A future server-mode
generator targeting `runtime/go` is tracked as a follow-up issue.

### Generated-file permissions

Generated files are currently written with `os.ModePerm` (0777). Directories
use `0755`, files use `0644`.

### Generated-code compile test

Add a test that runs the generator against `testdata/05_command_withfields.jsonc`
into a temp dir, runs `go mod tidy` and `go build ./...` on the result, and
fails on any compile error. This is the regression net that would have caught
the `wo/` template breakage. Requires network for module resolution in CI; gate
with `testing.Short()` skip so `-short` runs stay hermetic.

### Toolkit runtime pin

`toolkit/go.mod` moves from `runtime/go v0.6.0` to the `v0.9.0` tag produced by
this plan, landing the `x-mvep-` header rename the generated code already
expects.

## Affected Files

| File | Change |
| --- | --- |
| `runtime/go/mvep/http_transport.go` | Method check, media-type parsing, body limit, status mapping, error redaction, ctx propagation, body close, client response cap |
| `runtime/go/mvep/mvepackge.go` | Error codes, `ServeCmd` ctx, header map copy, `CmdResp` context seeding |
| `runtime/go/mvep/interceptors.go` | `AuthInterceptor` sets `unauthorized`; `RequestIDInterceptor` nil-response guard |
| `runtime/go/mvep/local_trust.go` | Peer verification |
| `runtime/go/mvep/server/server.go` | `MaxRequestBytes`, `AllowedOrigins`, `VerboseErrors`, `MaxHeaderBytes`, `POST`-scoped mux pattern, registry mutex |
| `runtime/go/mvep/client/client.go` | Registry mutex, command-name index, Unix dialer ctx, default client timeout, `MaxResponseBytes` |
| `runtime/go/mvep/http_transport_test.go` | Substantial new coverage; currently a single test |
| `runtime/go/mvep/local_trust_test.go` | Peer verification cases |
| `runtime/go/mvep/server/server_test.go` | Replace the three assertion-free interceptor tests |
| `runtime/ts/src/http-transport.ts` | `AbortController` timeout |
| `runtime/ts/src/client.ts` | Pass timeout into transport |
| `runtime/ts/src/interceptors.ts` | `crypto.randomUUID`, non-leaking timeout interceptor |
| `runtime/ts/LICENSE`, `runtime/ts/package.json` | Apache-2.0 to match repo root |
| `toolkit/toolkit_runner.go` | Return errors instead of `log.Fatal`; file perms `0755`/`0644` |
| `toolkit/toolkit_pb3.go` | Return errors instead of `log.Fatal` |
| `toolkit/resources/codegen_templates/go/` (10 files) | Rework `wo/` templates onto the MVEP runtime; drop laptop `replace` paths; remove NATS starter |
| `toolkit/toolkit_generate_test.go` (new) | Compile test for generated Go output |
| `toolkit/go.mod` | Bump `runtime/go` pin to `v0.9.0` |
| `runtime/go/README.md`, `runtime/go/mvep/server/SERVER.md` | Status codes, CORS, body limits, local trust semantics |
| `CHANGELOG.md` | `[Unreleased]` entries |

## Risks and Compatibility

- **Status code changes are observable.** Callers testing for `400` on auth
  failure will break. This is the intended correction; it is called out in the
  changelog and migration notes.
- **CORS default tightens.** A deployment relying on `EnableCORS: true` meaning
  `*` will stop receiving CORS headers until `AllowedOrigins` is set. This is a
  deliberate secure default and the startup warning makes it diagnosable.
- **Body limits can reject previously-accepted payloads.** Default 4 MiB is
  generous for command payloads but is configurable, and the `413` plus
  `payload_too_large` code makes the cause obvious.
- **Error redaction removes detail callers may be parsing.** Stable codes are
  the supported replacement; `VerboseErrors` covers local development.
- **`LocalTrustMiddleware` may stop trusting misconfigured deployments.** That is
  the point, but it can surface as a sudden `401` in an environment that was
  unknowingly trusting non-local peers.
- **`SO_PEERCRED` is platform-specific.** UID filtering is available on Linux and
  Darwin; elsewhere the middleware falls back to socket-type and loopback checks
  only, and documents the difference.
- **Registry duplicate detection is newly strict.** A process registering two
  packages that expose the same command name previously worked by accident and
  will now error at registration.
- **Toolkit internal signatures change.** The `log.Fatal` removal threads errors
  through internal functions. Nothing exported changes, but in-flight toolkit
  branches will conflict.
- **Generated scaffolding changes shape.** The reworked templates emit
  MVEP-runtime code, not `wo/` code. Any consumer relying on the old (broken)
  `wo/` output was already non-compilable, so this is not a regression, but the
  diff on regeneration will be total for those files.
- **NATS starter template removed.** If a `wo`-stack consumer still generates
  from it, they must pin `toolkit < v0.6.0`. No such external consumer can exist
  today because the output never compiled outside the maintainer's machine.
- **Compile test needs network.** The generated-output compile test resolves
  modules from the proxy; it is skipped under `-short` and must not gate
  hermetic CI jobs.

## Verification

1. `gofmt -l runtime/go` clean; `go vet ./runtime/go/...` clean.
2. `go build ./runtime/go/... ./toolkit/...`.
3. `go test ./runtime/go/... -count=1`.
4. `go test -race ./runtime/go/... -count=1` — the race detector is the primary
   evidence for the registry synchronization work.
5. `cd runtime/ts && npm test` and `npm run build`.
6. Targeted checks:
   - `curl -X GET` with `x-mainvec-cmd` returns `405`.
   - `curl -H 'Content-Type: application/json; charset=utf-8'` succeeds.
   - A body over the limit returns `413` and does not allocate the full body.
   - An `AuthInterceptor` rejection returns `401`, not `400`.
   - A handler returning `errors.New("dial tcp 10.0.0.5:5432: connect: refused")`
     produces a response containing neither the host nor the port.
7. A TypeScript test asserting that a request against a non-responding server
   rejects at the configured timeout, and that the underlying fetch is aborted.
8. `go test ./toolkit/... -count=1` — the new compile test generates from
   `testdata/05_command_withfields.jsonc` and builds the output.
9. `grep -rn "log\.Fatal" toolkit/toolkit_runner.go toolkit/toolkit_pb3.go`
   returns nothing; `grep -rn "mainvec/wo\|/Users/hi/" toolkit/resources/`
   returns nothing.
10. Regenerate `toolkit/mvepapi` from its own spec via `gengen` and confirm the
    working tree is unchanged (self-generation stays stable).

## Rollout

1. Land as a single branch, but split into reviewable commits along task
   boundaries. If the diff proves unwieldy, T1–T6 (runtime security), T7–T11
   (runtime correctness), and T13–T16 (toolkit) can be split into stacked PRs
   against the same issue, in that order — the toolkit pin (T16) depends on the
   runtime tag.
2. Tag `runtime/go/v0.9.0` and publish `@mainvec/mvep@0.8.0`.
3. Update `toolkit/go.mod` to the new runtime tag, then tag `toolkit/v0.6.0`.
4. Audit downstream consumers for `400`-checking and `EnableCORS` reliance
   before migrating them.

## Decision Log

- **Stable error codes over HTTP status alone.** The wire already carries
  `ErrorInfo.Code`; making it authoritative keeps the transport mapping in one
  place and gives non-HTTP transports the same semantics later.
- **Redact by default, verbose by opt-in.** The inverse default has repeatedly
  proven to leak in production systems, and the information is still available
  in server logs.
- **CORS silence over wildcard when unconfigured.** Emitting no CORS headers
  fails closed and is immediately diagnosable; emitting `*` fails open and is
  invisible.
- **Keep command selection in the header for now.** Moving the command name into
  the URL path would restore per-method routing, rate limiting, WAF rules, and
  useful access logs, and it is the direction AWS took with `smithy-rpc-v2-cbor`.
  It is also a protocol change affecting both runtimes and every consumer, so it
  is out of scope here and deserves its own issue.
- **Fix the legacy byte-stream path rather than deleting it.** It is still
  exported and may have consumers; removing it is a separate deprecation.
- **Fix `SetResponseHeader` rather than delete it.** It is documented in
  `runtime/go/README.md`, so silently removing it would break a documented
  contract; wiring it up is a small change.
- **Command-name index over map iteration.** Deterministic resolution and an
  explicit duplicate error are worth the registration-time strictness.
- **License to Apache-2.0.** The repo root and the Go runtime are already
  Apache-2.0; the TypeScript package declaring MIT is an oversight from the
  consolidation, not a deliberate dual-license.
- **Toolkit repair in scope, not deferred.** The original plan deferred toolkit
  defects to a separate issue. They were pulled in because the `wo/` templates
  make the generator's primary output non-compilable for every user — a
  severity above any single runtime defect — and because the runtime tag
  produced here is what the toolkit pin needs anyway.
- **Rework `wo/` templates onto the MVEP runtime rather than delete them.** The
  server/client/test scaffolding is a real feature; deleting it would shrink the
  generator's value. The NATS starter is the exception — it is removed because
  NATS is internal to the `wo` stack.
- **REVISED (T14): remove the dead `wo` generation path entirely.** Investigation
  during T14 found that the four functions emitting those templates
  (`GenerateGOSRV`, `GenerateGOClient`, `GenerateGOMod`, `GenerateGOAPI`) have
  **zero callers** — `ExecuteGenerate` never invokes them, and there is no live
  registration anywhere in the module. All 10 `wo`-flagged templates are
  unreachable dead code. Reworking dead templates to target the MVEP runtime
  would produce *more* dead code and give the T15 compile test nothing reachable
  to assert. They are deleted instead. A proper server-mode generator targeting
  `runtime/go` is a genuine feature but belongs in its own issue once there is a
  caller for it.
- **Compile test over golden files.** Golden files ossify formatting and churn
  on every template tweak; a `go build` of generated output asserts the property
  that actually matters — it compiles — with less maintenance.

## Progress

- [x] T1 — Add failing tests covering the `ServeHTTP` surface, status mapping, and redaction
- [x] T2 — Introduce stable error codes and the HTTP status mapping
- [x] T3 — Redact handler error detail; add `VerboseErrors`; log full detail server-side
- [x] T4 — Enforce `POST`-only routing and parse `Content-Type` as a media type
- [x] T5 — Bound request, response, and header sizes
- [x] T6 — Replace wildcard CORS with an origin allowlist
- [x] T7 — Add peer verification to `LocalTrustMiddleware`
- [x] T8 — Guard client and server registries; add a deterministic command-name index
- [x] T9 — Fix legacy-path context propagation, body leak, Unix dialer, and default client timeout
- [x] T10 — Wire `SetResponseHeader` through `executeCmd`
- [x] T11 — Implement the TypeScript request timeout and fix request-id generation
- [x] T12 — Update docs, changelog, and license metadata
- [x] T13 — Replace `log.Fatal` with error returns in toolkit library code
- [x] T14 — `wo/` dead-path removal (revised from rework; see Decision Log)
- [x] T15 — Fix generated-file permissions; add generated-output compile test
- [x] T16 — Runtime pins/tags: `runtime/go v0.9.0` + `runtime/ts 0.8.0` + `toolkit v0.6.0` tagged and published

## Tasks

### T1 — Failing tests for the transport surface

- **Outcome**: The `ServeHTTP` coverage gap is closed before any behavior
  changes, and each subsequent task has a failing test to satisfy.
- **Verification**: New cases in `runtime/go/mvep/http_transport_test.go` fail
  because the current handler accepts `GET`, returns `400` for every failure,
  rejects parameterized `Content-Type`, and echoes raw error text.
- **Notes**: `http_transport_test.go` currently holds a single test that mainly
  asserts the header prefix constant. Also replace the three assertion-free
  tests in `server_test.go` that set up an interceptor, assert
  `config.Interceptor != nil`, then discard the called flag with `_ =` without
  ever dispatching a command.

### T2 — Error codes and status mapping

- **Outcome**: Each failure condition carries a stable `ErrorInfo.Code` and maps
  to the documented HTTP status.
- **Verification**: Table-driven test asserting the full condition-to-status
  matrix; auth rejection returns `401`.
- **Notes**: Keep the mapping in one exported function so non-HTTP transports can
  reuse it. `AuthInterceptor` must set the code rather than relying on message
  matching.

### T3 — Error redaction

- **Outcome**: Responses expose a code and a generic message; full detail reaches
  server logs only.
- **Verification**: A handler returning an error containing a DSN produces a
  response body and headers containing neither host nor credentials.
- **Notes**: Also removes the response-splitting footgun of routing unsanitized
  text into a header value. `VerboseErrors` defaults to `false`.

### T4 — Method and media-type correctness

- **Outcome**: Non-`POST` requests receive `405`; `Content-Type` parameters no
  longer break encoder resolution.
- **Verification**: `GET` with a valid command header returns `405` and does not
  invoke the runner; `application/json; charset=utf-8` resolves the JSON encoder.
- **Notes**: Register the mux pattern as `"POST " + cmdPath`. The existing
  protobuf test setup already does this, so production routing is currently
  looser than the tests.

### T5 — Resource bounds

- **Outcome**: Request bodies, response bodies, and headers are bounded and
  configurable.
- **Verification**: Oversized body returns `413` with `payload_too_large`; a
  client reading an oversized response errors rather than buffering it.
- **Notes**: `MaxBytesReader` must wrap the body before `io.ReadAll`. Set
  `MaxHeaderBytes` on the per-listener `http.Server` alongside the
  `ReadHeaderTimeout` added in plan 016.

### T6 — CORS allowlist

- **Outcome**: Origins are explicit; no wildcard credentials exposure.
- **Verification**: A disallowed origin receives no CORS headers; an allowed
  origin receives its own origin echoed, not `*`; the advertised method list
  contains only `POST` and `OPTIONS`.
- **Notes**: `EnableCORS: true` with an empty `AllowedOrigins` emits nothing and
  warns at startup.

### T7 — Local trust peer verification

- **Outcome**: Trust is established from a verified peer property rather than
  from listener attachment alone.
- **Verification**: A loopback TCP request is trusted; a non-loopback request on
  the same listener is not; a Unix peer outside the UID allowlist is not.
- **Notes**: `AuthInterceptor` returns early for trusted contexts, so this
  middleware is the entire boundary. Platform differences are documented in
  `SERVER.md`. **Delivered**: middleware-level peer verification (Unix socket or
  loopback TCP `RemoteAddr`), which fails closed without needing conn-level
  access. UID allowlist via `SO_PEERCRED` requires the connection, not the
  request, so it is deferred to a follow-up that plumbs peer credentials through
  a `ConnContext` hook; the plan's Risks section already scopes it as
  platform-specific.

### T8 — Registry concurrency

- **Outcome**: Registration and dispatch are safe under concurrent use; command
  resolution is deterministic.
- **Verification**: `go test -race` with concurrent `RegisterPackage` and
  `SendCmd`; duplicate command registration returns an error.
- **Notes**: `sync.Mutex` currently appears nowhere in `client.go`. Replaces the
  O(n) `NameOf` scan in `SendCmd`.

### T9 — Legacy path fixes

- **Outcome**: Cancellation propagates, connections are not leaked, and no code
  path can hang indefinitely.
- **Verification**: A cancelled context aborts an in-flight legacy request; a
  test asserting `resp.Body` is closed on the non-200 branch; a transporter built
  directly has a non-zero timeout.
- **Notes**: Four separate small fixes in `http_transport.go`, `mvepackge.go`,
  and `client.go`. Also copy the caller's header map in `SendCmdReq` instead of
  aliasing it, and guard `RequestIDInterceptor` against a nil `*CmdResp`.

### T10 — Response header propagation

- **Outcome**: `SetResponseHeader` behaves as documented.
- **Verification**: A handler calling `SetResponseHeader` produces a matching
  `x-mvep-` header on the HTTP response.
- **Notes**: `executeCmd` already reads `CmdRespFromContext`; only the seeding
  side is missing.

### T11 — TypeScript timeout and request ids

- **Outcome**: The documented timeout is enforced and aborts the underlying
  request; request ids are not derived from `Math.random`.
- **Verification**: A request against a non-responding server rejects at the
  configured timeout with the fetch aborted; the timeout interceptor leaves no
  pending timer.
- **Notes**: Also document that request ids are correlation identifiers and must
  not be treated as unguessable.

### T12 — Documentation and metadata

- **Outcome**: Status codes, CORS configuration, body limits, and local-trust
  semantics are documented; licensing is consistent.
- **Verification**: Docs describe the shipped behavior; `runtime/ts/LICENSE` and
  `package.json` both read Apache-2.0.

### T13 — Toolkit error handling

- **Outcome**: No `log.Fatal` in `toolkit_runner.go` or `toolkit_pb3.go`;
  generation failures return wrapped errors to the caller, and the `mvep` CLI
  exits non-zero from `main` only.
- **Verification**: `grep -n "log\.Fatal" toolkit/toolkit_runner.go toolkit/toolkit_pb3.go`
  returns nothing; a generation run against the invalid fixture
  `testdata/03_basic_wo_invalid.jsonc` returns an error without exiting the
  test process.
- **Notes**: ~25 sites across both files. In `toolkit_runner.go` the affected
  functions are module-internal; in `toolkit_pb3.go` the fatal sites live in
  internal helpers (`processOptions`, `processOneField`, …) called from the
  exported `Build*`/`Generate*` functions, which already return errors — thread
  the error up through them. Exported signatures keep their shape; internal
  helpers gain error returns. Test files and `gengen`'s `main` keep `log.Fatal`.

### T14 — `wo/` dead-path removal

- **Outcome**: The 10 `wo/`-referencing templates and the four dead generator
  functions that emit them are deleted; `grep` for `mainvec/wo` and `/Users/hi/`
  under `toolkit/resources/` returns nothing.
- **Verification**: `go build ./toolkit/...`; full toolkit suite still green;
  `grep -rn "mainvec/wo\|/Users/hi/" toolkit/` finds nothing.
- **Notes**: All 10 templates and `GenerateGOSRV`/`GenerateGOClient`/
  `GenerateGOMod`/`GenerateGOAPI` were unreachable (no caller). Deleted per the
  revised Decision Log. **Follow-up**: file an issue for a real MVEP-runtime
  server-mode generator once there is a consumer.

### T15 — File permissions and compile test

- **Outcome**: Generated files are `0644`, directories `0755`; a test generates
  Go output from a fixture and compiles it.
- **Verification**: `go test ./toolkit/ -run TestGenerateCompile -count=1`
  passes; generated files in the temp dir are not world-writable.
- **Notes**: Skip the compile test under `testing.Short()` — it needs module
  resolution from the proxy. This test is the regression net for T14.

### T16 — Runtime pins/tags

- **Outcome**: All three artifacts tagged and published: `runtime/go/v0.9.0`,
  `runtime/ts@0.8.0` (npm), `toolkit/v0.6.0` (GitHub release with 5 binaries).
- **Verification**: `npm @mainvec/mvep latest` = 0.8.0 / Apache-2.0; GitHub
  release `toolkit/v0.6.0` has the platform binaries attached.
- **Notes**: The `toolkit/go.mod` pin stays at `runtime/go v0.6.0` — in-repo
  builds resolve HEAD via `go.work`, and the proxy/sumdb lookup of the fresh
  tag wasn't worth fighting. The pin is cosmetic until an external consumer
  resolves the toolkit module; it can be bumped on the next tag without this
  dance. Status: **shipped 2026-07-29**.
