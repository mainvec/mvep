# Plan 018 — Harden MVEP runtime HTTP transport security and correctness

- **Issue**: [#18](https://github.com/mainvec/mvep/issues/18) — fix(runtime): harden HTTP transport security and correctness
- **Branch**: `fix/018-runtime-hardening`
- **Target release**: `runtime/go/v0.9.0`, `@mainvec/mvep@0.8.0`

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

The goal is a transport layer that is safe to expose directly: correct HTTP
semantics, bounded resource consumption, an explicit CORS policy, no internal
detail leakage, and a race-free client and server registry.

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

## Non-goals

- Changing the wire envelope, the `x-mainvec-cmd` selector, the `x-mvep-` header
  prefix, or the encoder registry design.
- Moving command selection from a header into the URL path. That is worth doing
  and is discussed in the Decision Log, but it is a protocol change and belongs
  in its own issue.
- Adding authentication mechanisms. `TokenValidator` remains caller-supplied.
- Streaming, batching, or chaining. Batching is tracked separately in #19.
- Toolkit defects (`log.Fatalf` in library code, laptop paths in templates,
  `os.ModePerm` on generated files, runtime version pin drift). Those are real
  but belong to a separate toolkit issue.
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

## Rollout

1. Land as a single branch, but split into reviewable commits along task
   boundaries. If the diff proves unwieldy, T1–T6 (security) and T7–T11
   (correctness) can be split into two PRs against the same issue.
2. Tag `runtime/go/v0.9.0` and publish `@mainvec/mvep@0.8.0`.
3. Update `toolkit/go.mod`, which currently pins `runtime/go v0.6.0` — two minor
   versions behind, and behind the `x-mvep-` header rename its generated code
   depends on.
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

## Progress

- [ ] T1 — Add failing tests covering the `ServeHTTP` surface, status mapping, and redaction
- [ ] T2 — Introduce stable error codes and the HTTP status mapping
- [ ] T3 — Redact handler error detail; add `VerboseErrors`; log full detail server-side
- [ ] T4 — Enforce `POST`-only routing and parse `Content-Type` as a media type
- [ ] T5 — Bound request, response, and header sizes
- [ ] T6 — Replace wildcard CORS with an origin allowlist
- [ ] T7 — Add peer verification to `LocalTrustMiddleware`
- [ ] T8 — Guard client and server registries; add a deterministic command-name index
- [ ] T9 — Fix legacy-path context propagation, body leak, Unix dialer, and default client timeout
- [ ] T10 — Wire `SetResponseHeader` through `executeCmd`
- [ ] T11 — Implement the TypeScript request timeout and fix request-id generation
- [ ] T12 — Update docs, changelog, and license metadata

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
  middleware is the entire boundary. Document the platform differences in
  `SERVER.md`, which already warns about this in bold.

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
- **Notes**: Include a short migration note covering the status-code and CORS
  changes for downstream consumers.
