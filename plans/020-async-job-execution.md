# Plan 020 — Async job execution for MVEP commands

> **STATUS: APPROVED FOR IMPLEMENTATION.**
>
> **Revision 2 (2026-07-31)** — reviewed against the runtime source. Changes:
> a failed job is now an HTTP 200 status response rather than a
> `CmdResp.Error`; corrected the false claim that `x-mvep-error-code`/
> `x-mvep-error-message` exist; removed references to a non-existent
> `OnlyCommands` auth policy; added `Job.Encoder` so `GET /jobs/{id}` serves a
> truthful `Content-Type`; fixed the convenience route dropping the caller's
> `auth` header; added a required `PackageClient.SendEnvelope` prerequisite;
> added header sanitization, `MaxJobResultBytes`, and shutdown/submit
> ordering; dropped `ResultHeaders`; split delivery into three PRs.

- **Issue**: [#21](https://github.com/mainvec/mvep/issues/21)
- **Branch**: `feat/021-async-job-execution`
- **Target release**: undecided; not before `runtime/go` next minor
- **Depends on**: none

## Problem / Goal

Every MVEP command executes synchronously: the HTTP request that carries it
blocks until the handler finishes, is encoded, and is written back. There is
no way to run a long-lived command (a report, an export, a batch operation)
without holding a connection, a goroutine, and a client open for its full
duration, and no standard way to check on such work after the fact.

This plan adds an opt-in, runtime-only mechanism for running any existing
command as a background job: the client wraps a command in a reserved
built-in `SubmitJob` command, the server creates a `Job`, runs the wrapped
command in a background goroutine, and returns a job ID immediately. The
client polls for status and, eventually, the result — via a reserved
`GetJobStatus` command or a convenience URL.

## Goals

- Let a client submit any existing, already-registered command to run as a
  background job instead of inline, with no spec or codegen change required.
- Return a job ID immediately, before the wrapped command has necessarily
  started running.
- Let a client poll job status/result via a standard reserved command
  (`GetJobStatus`) that goes through the normal envelope and interceptor
  chain (so auth applies), and via a convenience `GET /jobs/{id}` URL with
  identical auth posture.
- Let a running command optionally report incremental progress
  (percent/message) via a context-based helper, with no effect when the
  command is not running as a job.
- Keep the feature off by default (`EnableAsyncJobs`) and fully backward
  compatible with existing `PackageHandler`/`Server`/`Client` construction.
- Respect the server's existing single-use, caller-owned shutdown lifecycle
  (plans/016): in-flight jobs must be drained, bounded by the shutdown
  context, the same way in-flight HTTP requests already are.

## Non-goals

- Command cancellation (`CancelJob`). Left for a follow-up issue.
- A persistent or distributed job store. v1 ships an in-memory store behind
  a pluggable interface; multi-instance deployments must supply their own.
- Any change to `toolkit/` (spec format or codegen). This is a runtime-only
  feature that works for every existing generated package unmodified.
- TypeScript runtime (`runtime/ts/`) support. The wire protocol is plain
  JSON and language-agnostic, so a TS caller can still interoperate by hand.
- Any relationship to `plans/019-cmd-chain.md`. That plan is draft and
  unimplemented; this plan does not depend on it, share code with it, or
  impose nesting rules referencing it. Any structural resemblance (recursing
  through `ServeCmdReq` rather than `executeCmd`, reserved command names, an
  encoder allowlist) is independently derived here, not reused.
- Queueing/backpressure when the concurrent-job limit is reached; v1 rejects
  immediately instead.
- Per-caller authorization on `GetJobStatus`: any caller passing the
  interceptor chain can read any job given its ID. IDs are 128-bit
  `crypto/rand` values so enumeration is infeasible, but the design does not
  tie job visibility to the submitter's identity. If that is needed, a
  follow-up can check the caller's auth against the job's stored submitter
  headers.

## Proposed Design

### Reserved built-in commands

Two new reserved, hand-written command names, recognized directly inside
`PackageHandler.ServeCmdReq`'s core-handler closure, before it falls through
to the existing `executeCmd` method:

- `SubmitJobName = "SubmitJob"`
- `GetJobStatusName = "GetJobStatus"`

Placement mirrors the existing shape of `ServeCmdReq` (interceptor wraps a
core handler that decides what to do with `req.Cmd`) and requires no spec
change, no codegen change, and no toolkit release — identical to how the
(also unimplemented) CmdChain plan reasoned about this same placement,
independently re-derived here.

Because the switch lives inside `coreHandler`, which `Interceptor` wraps,
**both reserved commands are themselves subject to the interceptor chain**
— `SubmitJob` requires auth exactly like any other command, not just the
command it wraps. This is intentional: job submission needs the same
auth/logging coverage as everything else.

### Wire model: encoder-independent envelopes (in `runtime/go/mvep/job.go`)

The feature is **fully encoder-independent** — it works with JSON, protobuf,
protojson, and any future encoder, with no `RawPayload` trick, no JSON
allowlist, and no `job_unsupported_encoder` error. This follows directly
from how the existing envelope already works.

`CmdReq`/`CmdResp` (`envelope.go`) are plain structs with no struct tags and
are **never passed to any encoder**. `ServeHTTP` reads the request body into
`CmdReq.Payload []byte` and writes `CmdResp.Payload []byte` straight to the
wire; the encoder only ever touches the **inner** command/result types
inside `executeCmd` (`enc.Decode(req.Payload, cmd)` / `enc.Encode(cmdResult)`).
The envelope is a transport-only, encoder-agnostic opaque-byte container.
The job feature reuses this exact property instead of fighting it.

Concretely, the wrapped command travels as **opaque bytes** and the job
metadata travels in **headers** — the same channel `CmdResp.Headers` already
uses for `x-mvep-*` response headers via `SetResponseHeader`. No typed status
struct crosses the encoder boundary. (Verified: the protobuf and protojson
encoders in `util/protobuf`/`util/protojson` both do a hard `v.(proto.Message)`
type assertion and reject any hand-written struct, so a typed `SubmitJob`/
`GetJobStatusResult` would be JSON-only by construction. The envelope-and-
headers model sidesteps that entirely.)

#### Submit (request → `SubmitJob` command)

Request body is the client's already-encoded inner command (whatever
encoder the client chose — protobuf, JSON, …), carried verbatim. The
wrapped command name and the submitter's headers ride in `CmdReq.Headers`:

- `CmdReq.Cmd = "SubmitJob"`
- `CmdReq.Headers["job-cmd"]` = inner command name (e.g. `"ExportReport"`)
- `CmdReq.Headers` also carries the submitter's `auth` and any other
  headers, exactly as a normal request would
- `CmdReq.Payload []byte` = the client's already-encoded inner command
  bytes — never re-encoded, never inspected by the job machinery

The request's `Content-Type` (the encoder mime type) is recorded on the
`Job` as `Encoder`. It is needed twice later: `runJob` passes it to the
recursive `ServeCmdReq` call, and the `GET /jobs/{id}` convenience route
must echo it as the response `Content-Type` (see below) — the stored result
bytes are in the submitter's encoding, not necessarily JSON.

Response: a normal `CmdResp` carrying the job ID and echoed command name in
**response headers**, with an empty body:

- `CmdResp.Headers["job-id"]` = generated job ID
- `CmdResp.Headers["job-cmd"]` = inner command name
- `CmdResp.Payload` = empty

Because the response body is empty and the metadata is in headers, there is
no typed `SubmitJobResult` struct to encode — the response is encodable by
every encoder trivially (an empty body needs no encoder at all).

#### Status (`GetJobStatus` command → response)

Request body is empty; the job ID rides in a header:

- `CmdReq.Cmd = "GetJobStatus"`
- `CmdReq.Headers["job-id"]` = job ID
- `CmdReq.Payload` = empty

Response: structured status metadata in **response headers**, the inner
command's verbatim natively-encoded result in the body:

- `CmdResp.Headers["job-status"]` = `"pending" | "running" | "succeeded" | "failed"`
- `CmdResp.Headers["job-progress-percent"]` = `strconv.Itoa(percent)` (only
  when a `JobProgress` has been set; omitted otherwise)
- `CmdResp.Headers["job-progress-message"]` = sanitized message (only when set)
- `CmdResp.Headers["job-error-code"]` / `["job-error-message"]` = the inner
  command's failure, set **only** when `job-status == "failed"`
- `CmdResp.Payload []byte` = the inner command's verbatim natively-encoded
  result, present only when `Status == "succeeded"`. A protobuf client gets
  protobuf-encoded result bytes back; a JSON client gets JSON — both work
  without the job machinery ever invoking an encoder on the status response.

**Encoder-mismatch rejection on `GetJobStatus`.** When a job has succeeded,
  the result bytes are in `job.Encoder`. `ServeHTTP` labels the response
  `Content-Type` with the poll request's media type, so polling a
  protobuf-encoded job with `application/json` would serve protobuf bytes
  labelled `application/json` — a lie. `handleGetJobStatus` therefore rejects
  a poll whose `encoder` argument differs from `job.Encoder` when a result
  payload is present, returning `unsupported_media_type`. A status-only poll
  (pending/running/failed, no payload) is not affected.

There is **no typed `GetJobStatusResult` struct** on the wire. The client
reads `job-status`/`job-progress-*`/`job-error-*` from response headers and
decodes `Payload` with the same encoder it used for the inner command. This
is the same flat-string trade `CmdResp.Headers` already makes.

#### A failed job is a successful status query (HTTP 200)

`CmdResp.Error` on the status response is reserved for failures of the
**query itself** — `job_not_found`, `job_store_error`, feature disabled. A
job that ran and failed reports `job-status: failed` plus
`job-error-code`/`job-error-message` headers on an **HTTP 200**.

This is not a stylistic choice; setting `CmdResp.Error` for a failed job
breaks three ways against the existing transport:

1. `ServeHTTP` (`http_transport.go`) treats any `CmdResp.Error` as a failed
   request: it calls `http.Error(w, msg, status)`, which **discards
   `cmdResp.Payload`** and overwrites `Content-Type` with `text/plain`.
2. The status is computed by `HTTPStatusForErrorCode` from the **inner**
   command's code, so a job that failed with `unauthorized` would make the
   *poll* return HTTP 401 — the caller cannot tell "your poll was rejected"
   from "the job you asked about failed".
3. `PackageClient`'s send path returns a non-nil Go `error` whenever
   `resp.HasError()`, so every poll of a failed job surfaces as a transport
   error rather than a terminal job state.

An unknown job ID *is* a query failure, so it keeps `CmdResp.Error =
&ErrorInfo{Code: "job_not_found", ...}` → HTTP 404.

#### Correction: `ErrorInfo.Code` does not survive the HTTP round trip today

There are no `x-mvep-error-code`/`x-mvep-error-message` headers. `ServeHTTP`
writes `x-mainvec-error-code`, and writes `x-mainvec-error` **only when
`VerboseErrors` is set**. On the receiving side `HttpTransporter.
TransportCmdReq` ignores both and synthesizes `ErrorInfo{Code:
fmt.Sprintf("http_%d", resp.StatusCode)}`. A Go client therefore sees
`http_404`, never `job_not_found`.

Adding the new `HTTPStatusForErrorCode` cases makes the **status code**
correct and nothing more. This is a pre-existing gap, not one this feature
introduces, and this plan does **not** fix it — which is precisely why the
failed-job path above goes through `job-error-code` headers (which do
round-trip, via the existing `x-mvep-*` extraction loop) instead of through
`CmdResp.Error`. If stable error codes over HTTP are wanted generally, that
is a separate issue against the transport.

#### Header value constraints

`job-progress-message` and `job-error-message` carry handler-supplied text
into HTTP response headers, so the job machinery must, before setting them:

- strip CR/LF and other control bytes (Go's `net/http` silently rewrites
  newlines to spaces, which would otherwise corrupt the value invisibly);
- drop or percent-escape non-ASCII, which is not representable in an HTTP
  header value;
- truncate to `MaxJobMessageBytes` (proposed 512) — these values are
  otherwise unbounded and count against the peer's `MaxHeaderBytes`.

This sanitization belongs in one helper in `job.go`, applied at the point
headers are written, not left to callers of `SetJobProgress`.

### Internal (non-wire) types

```
JobStatus   string   // "pending" | "running" | "succeeded" | "failed"

JobProgress {
    Percent int
    Message string
}

Job {
    ID, Cmd         string
    Encoder         string          // submit-time mime type; replayed on execute and on GET /jobs/{id}
    Status          JobStatus
    Progress        *JobProgress
    Headers         map[string]string // submitter's headers, replayed on the inner request
    Payload         []byte            // submitter's verbatim encoded inner command
    CreatedAt       time.Time
    StartedAt       time.Time
    CompletedAt     time.Time
    ResultPayload   []byte            // inner command's verbatim encoded result
    Error           *ErrorInfo
}
```

`Job` and `JobProgress` are internal-only; they never cross the wire as
typed structs, so they need no encoder tags and no `proto.Message`
implementation.

`Headers` and `Payload` are listed explicitly because `runJob` reconstructs
the inner `*CmdReq` from them; an earlier draft referenced `job.Headers`/
`job.Payload` without declaring either field.

**No `ResultHeaders`.** An earlier draft stored the inner command's response
headers and never read them back, and merging them into the status response
would let an inner command's `SetResponseHeader("job-status", …)` silently
clobber the envelope's own keys. Inner response headers are dropped in v1;
if a use case appears, they need a distinct prefix (e.g. `job-result-*`),
decided then rather than smuggled in now.

**`job-cmd` is stripped from the replayed headers.** The submitter's header
map contains the `job-cmd` key that selected the wrapped command. Copying it
verbatim onto the inner `CmdReq` leaks job-machinery metadata into an
ordinary command's header namespace, so `handleSubmitJob` removes every
`job-*` key when it copies headers into `Job.Headers`. Only non-reserved
headers (`auth`, `request-id`, tracing, …) are replayed.

`JobStore` is a pluggable interface with explicit state-transition methods —
deliberately not a mutate-closure — so a future remote-backed implementation
(Redis, a DB) stays straightforward to write:

```
type JobStore interface {
    Create(ctx context.Context, job *Job) error
    Get(ctx context.Context, jobID string) (*Job, bool, error)
    MarkRunning(ctx context.Context, jobID string, startedAt time.Time) error
    MarkSucceeded(ctx context.Context, jobID string, payload []byte, completedAt time.Time) error
    MarkFailed(ctx context.Context, jobID string, errInfo *ErrorInfo, completedAt time.Time) error
    SetProgress(ctx context.Context, jobID string, progress *JobProgress) error
}
```

`InMemoryJobStore` is the default implementation: `map[string]*Job` guarded
by a `sync.RWMutex`. It sweeps entries whose `CompletedAt` is older than
`JobRetention` opportunistically on writes (no dedicated GC goroutine or
lifecycle to manage), and never sweeps pending/running jobs.

**Stored-result size cap** (resolves Open Question 3, previously unanswered).
`MaxRequestBytes` bounds what a client can submit, but nothing bounds what a
completed job holds: a result sits in memory from completion until the
retention sweep, and `MaxConcurrentJobs` bounds concurrency, not the number
of *retained completed* jobs. `PackageHandler` therefore gains
`MaxJobResultBytes int64` (`<=0` uses `DefaultMaxJobResultBytes`, proposed
4 MiB to match `DefaultMaxResponseBytes`). `runJob` checks the encoded
result length before `MarkSucceeded`; over the cap, the job transitions to
`failed` with `job_result_too_large` rather than being retained. This is a
size check in `runJob`, not a `JobStore` responsibility, so third-party
stores inherit it for free.

`GenerateJobID() string` is a small `crypto/rand`-backed random ID generator
(e.g. 16 bytes, hex-encoded). No new third-party dependency: the workspace
has no existing UUID library in either `runtime/go/go.mod` or
`toolkit/go.mod`.

### HTTP status codes for the new error codes

`HTTPStatusForErrorCode` (`runtime/go/mvep/envelope.go`) is a `switch` with
no knowledge of this feature's error codes; left alone, all would fall
through to its `default: return 500`. The switch gains these cases
(`job_unsupported_encoder` is gone — see the encoder-independent wire model
above):

- `job_not_found` → 404 (same class as the existing `unknown_command` → 404)
- `job_queue_full` → 429 (capacity exhausted, retryable — matches standard
  "too many requests" semantics better than 503, since the server itself is
  healthy)
- `nested_job_forbidden` → 400 (same class as `invalid_request`)
- `job_store_error` → 500 (operational failure of the `JobStore` itself —
  e.g. a future persistent store's disk/Redis error; surfaces as a generic
  server error, not one of the client-retryable 4xx codes)

`job_result_too_large` deliberately gets **no** case here: it is never a
`CmdResp.Error`. It is a terminal *job* state, surfaced as
`job-error-code: job_result_too_large` on an HTTP 200 status response, per
the failed-job model above.

This is a direct edit to `envelope.go`, not just `job.go` — it must land in
the same change or every new error response ships as a misleading 500.
Note the scope limit established above: these cases fix the **HTTP status
code only**. Because `HttpTransporter.TransportCmdReq` rewrites
`ErrorInfo.Code` to `http_<status>`, a Go client still cannot switch on
`job_not_found` — it must switch on the 404. Non-HTTP transports and
in-process callers of `ServeCmdReq` do see the real code.

### `PackageHandler` changes (`runtime/go/mvep/mvepackge.go`)

New exported fields, all zero-value safe so existing `&mvep.PackageHandler{...}`
struct-literal construction (used today in `server.go` and `client.go`)
keeps compiling unchanged, and the feature is off by default:

- `EnableAsyncJobs bool`
- `JobStore JobStore` — nil lazily defaults to a fresh `InMemoryJobStore`
- `MaxConcurrentJobs int` — `<=0` uses `DefaultMaxConcurrentJobs`, mirroring
  the existing `MaxRequestBytes`/`DefaultMaxRequestBytes` zero-value pattern
- `MaxJobResultBytes int64` — `<=0` uses `DefaultMaxJobResultBytes`
- `JobRetention time.Duration` — `<=0` uses a sane default (proposed 10 min)
- `JobTimeout time.Duration` — `0` means no forced timeout; unlike the
  (unimplemented) CmdChain plan's mandatory `ChainTimeout`, this is optional
  because a job is expected to potentially run long — that is the point

New unexported state, lazily initialized behind `sync.Once`:
`jobCtx context.Context` + `jobCancel context.CancelFunc` (base context for
every job goroutine, canceled only by `Shutdown`, never tied to a single
HTTP request), `jobSem chan struct{}` sized `MaxConcurrentJobs`, and
`jobWG sync.WaitGroup`.

**Lazy init and `Shutdown` must not race.** `sync.Once` alone is not enough:
if `Shutdown` runs before any job was ever submitted, `jobCancel` is nil; and
if a `SubmitJob` arrives *after* `Shutdown`, the `Once` has already fired, so
the submit would create a job against an already-canceled `jobCtx` that dies
instantly with a confusing error. The handler therefore also carries a
`jobsClosed bool` guarded by the same mutex as the `Once`:

- `Shutdown` runs the same init path (so `jobCancel` is always non-nil), sets
  `jobsClosed = true`, then cancels and waits.
- `handleSubmitJob` checks `jobsClosed` immediately after init and returns
  `CmdResp.Error{Code: "job_store_error", Message: "server is shutting down"}`
  — no job created, no semaphore slot taken.
- `GetJobStatus` keeps working after shutdown so a caller can read the
  terminal state of a job that already finished.

`ServeCmdReq`'s `coreHandler` closure gains a switch on `req.Cmd`:

```
switch req.Cmd {
case SubmitJobName:
    return h.handleSubmitJob(ctx, req, encoder)
case GetJobStatusName:
    return h.handleGetJobStatus(ctx, req, encoder)
default:
    return h.executeCmd(ctx, req, encoder)
}
```

**`handleSubmitJob`**, in order (no typed `SubmitJob` decode — metadata is
in `req.Headers` and the body is opaque bytes):
1. `EnableAsyncJobs` false → `unknown_command` (indistinguishable from an
   unregistered command to the caller).
2. Read `jobCmd := req.Headers["job-cmd"]`. Empty, or itself
   `SubmitJobName`/`GetJobStatusName` → `nested_job_forbidden`.
3. `jobCmd` unknown per `h.Package.InstanceOf(jobCmd)` → `unknown_command`,
   no job created.
4. Non-blocking acquire on `jobSem`; full → `job_queue_full` (reject
   immediately, no internal queueing in v1).
5. `GenerateJobID()`. `JobStore.Create` with `Cmd: jobCmd`, `Encoder: encoder`
   (the request's mime type, needed to replay the command and to serve
   `GET /jobs/{id}` with a truthful `Content-Type`), `Status: pending`,
   `Payload: req.Payload` verbatim, and `Headers` = a fresh copy of
   `req.Headers` **with every `job-*` key removed** (never alias caller-owned
   maps; never leak job metadata into the inner command's header namespace).
   `InMemoryJobStore.Create` stores a shallow copy, not the caller's pointer,
   so struct-level mutations by the caller don't leak into the store.
   **If `Create` returns an error, release the acquired semaphore slot
   immediately** and return `CmdResp.Error{Code: "job_store_error"}` — the
   sem-release `defer` lives in `runJob` (step 1 below), which only starts
   after a successful `Create`, so a `Create` failure would otherwise leak
   the slot and permanently shrink the concurrency budget.
6. On `Create` success: hold `jobMu` across the `jobsClosed` check **and**
   `jobWG.Add(1)`, then release. This prevents `Shutdown`'s `jobWG.Wait()`
   from returning before the new job is counted — a race that
   `sync.WaitGroup` explicitly forbids — and prevents a job from running
   after shutdown claims to have drained. If `jobsClosed`, release the
   semaphore slot and return `job_store_error`.
7. `go h.runJob(jobID, job)` — the encoder is read from `job.Encoder`, not
   passed separately.
8. Return a `CmdResp` with `Headers["job-id"] = jobID`,
   `Headers["job-cmd"] = jobCmd`, and an empty `Payload` — this is the
   "returns immediately" contract.

**`runJob(jobID, job)`** (background goroutine):
1. `defer h.jobWG.Done()`; `defer func() { <-h.jobSem }()`.
2. `JobStore.MarkRunning` (error → `MarkFailed` with `job_store_error` and
   return — a job that can't transition state is dead).
3. Build the execution context from `h.jobCtx` — never the original HTTP
   request's context, which is canceled the instant `ServeHTTP` returns.
   Wrap with `context.WithTimeout` if `JobTimeout > 0`. Attach a
   progress-reporter sink (job ID + store) so `SetJobProgress` resolves.
4. Copy `job.Headers` into a fresh map, build an inner
   `*CmdReq{Cmd: job.Cmd, Headers, Payload: job.Payload}`. `job.Payload` is
   the submitter's already-encoded inner command bytes, carried verbatim —
   no re-encoding. `job.Headers` typically includes the submitter's `auth`
   token, so the inner command authenticates as the submitter once
   `AuthInterceptor` runs on the recursive call in step 5 below — this
   propagation is load-bearing for the auth-not-bypassable guarantee, not an
   incidental side effect. See the credential-at-rest risk below: this is
   also the reason the job store holds a live bearer token.
5. Call `h.ServeCmdReq(jobCtx, innerReq, job.Encoder)` recursively — **not**
   `executeCmd` — so the inner command still passes through `Interceptor`
   (auth, logging) exactly as if it had been called directly, and decodes/
   encodes with the submitter's encoder. Calling `executeCmd` instead would
   let a caller run a privileged command in the background while bypassing
   `AuthInterceptor` and any application-supplied per-command authorization
   interceptor — a privilege-escalation hole.
6. On response: error → `JobStore.MarkFailed(resp.Error, now)`; success →
   check `len(resp.Payload)` against `MaxJobResultBytes` and `MarkFailed`
   with `job_result_too_large` if it exceeds, otherwise
   `JobStore.MarkSucceeded(resp.Payload, now)` — `resp.Payload`
   is the inner command's natively-encoded result bytes, stored verbatim.

**`handleGetJobStatus`**: read `req.Headers["job-id"]`; `JobStore.Get`; not
found → `CmdResp.Error{Code: "job_not_found"}`; store failure →
`CmdResp.Error{Code: "job_store_error"}`. Otherwise build a **success**
`CmdResp` (never `CmdResp.Error`, per the failed-job model above) with:

- `Headers["job-status"]` always;
- `Headers["job-progress-percent"]` / `["job-progress-message"]` only when a
  `JobProgress` is set, message sanitized and truncated;
- `Headers["job-error-code"]` / `["job-error-message"]` only when
  `Status == "failed"`, message sanitized and truncated;
- `Payload = job.ResultPayload` only when `Status == "succeeded"`, empty
  otherwise.

No typed `GetJobStatusResult` struct is built or encoded.

**`SetJobProgress(ctx, percent, message)`**: resolves a job-progress sink
from `ctx` and, if present, calls `JobStore.SetProgress`. If absent (i.e.
called from a normal synchronous command), it is a **true no-op** — it must
not fabricate anything. This is deliberately unlike `SetResponseHeader`
(`envelope.go`), which *does* create a `CmdResp` and store it in context
when none is present; there is no equivalent need to fabricate a progress
sink here, since there's nothing downstream that expects one. Safe for a
`runXxxCmd` implementation to call unconditionally regardless of sync/async
execution.

### Server lifecycle integration (`runtime/go/mvep/server/server.go`)

- `ServerConfig` gains `EnableAsyncJobs bool`, `JobStore mvep.JobStore`,
  `MaxConcurrentJobs int`, `MaxJobResultBytes int64`,
  `JobRetention time.Duration`, `JobTimeout time.Duration`, plumbed into the
  `PackageHandler` literal built in `RegisterPackage` alongside the existing
  `Interceptor`, `MaxRequestBytes`, `VerboseErrors` fields.
- `PackageRegistration` gains `Handler *mvep.PackageHandler` — today the
  handler is a local variable in `RegisterPackage` and is never retained, so
  the server has no way to reach it again after registration.
- New `PackageHandler.Shutdown(ctx context.Context) error`: cancels
  `jobCtx` (stopping every running job's context — subject to the same
  "handlers that ignore context cannot be forced to stop" caveat already
  documented for `ShutdownContext`), then waits on `jobWG` up to `ctx`'s
  deadline; returns a context-deadline error if jobs are still running when
  `ctx` expires.
- `Server.drain` (or `ShutdownContext` directly) calls
  `pkgHandler.Shutdown(ctx)` for every registered package and joins any
  resulting error into the existing `errors.Join` accumulation
  (`recordError`), preserving the single-shutdown-budget, idempotent,
  no-signal-handling contract established in plans/016.
- When `EnableAsyncJobs` is true, `RegisterPackage` also mounts a
  **per-package literal route** — `GET {BasePath}/{pkgName}/jobs/{jobId}` —
  the same registration model as the existing command endpoint (`cmdPath :=
  s.config.BasePath + "/" + pkg.GetName() + "/cmd"`, `server.go`). Only the
  trailing `{jobId}` segment is a Go 1.24 `net/http.ServeMux` wildcard; the
  package name is a literal per registration, exactly like `cmdPath` today.
  This was explicitly **not** designed as a single catch-all route with a
  `{pkg}` wildcard, which would require the handler to validate the package
  name itself and would depart from the existing per-package registration
  model.
  The handler extracts `r.PathValue("jobId")` and the `x-mvep-*` request
  headers exactly as `ServeHTTP` does, then builds a synthetic `CmdReq{Cmd:
  GetJobStatusName, Headers: <extracted headers, with "job-id" set to the
  path value>, Payload: nil}` **directly as a Go value** — it does not read
  or set the `x-mainvec-cmd` header at all, since it calls
  `pkgHandler.ServeCmdReq` rather than going through `ServeHTTP`'s
  header-parsing path; the command is implicit.

  The extracted headers must be **carried through**, not discarded: an
  earlier draft passed `map[string]string{"job-id": jobID}` alone, which
  drops `auth` and therefore makes every request through this route fail
  `AuthInterceptor` with `unauthorized`. The path value overwrites any
  client-supplied `job-id` header so the URL remains authoritative.

  **`Content-Type` is not immaterial.** `GetJobStatus` never encodes a typed
  body, so the `encoder` argument to `ServeCmdReq` is unused by the job
  machinery — but the *response body* is the inner command's result in the
  submitter's encoding. Serving protobuf bytes under `application/json` is
  wrong. The handler therefore reads the completed job's `Encoder` and writes
  it as the response `Content-Type`; for pending/running/failed jobs the body
  is empty and the header is omitted. Obtaining the encoder without reading
  the store directly (which would bypass auth) means echoing it back through
  the envelope: `handleGetJobStatus` sets
  `CmdResp.Headers["job-encoder"] = job.Encoder` when a result payload is
  present, and the route handler uses that. `ServeCmdReq` still needs a
  non-empty `encoder` argument to pass its own validation; `"application/json"`
  is a fine placeholder there.

  The handler must still write the `CmdResp` it gets back to the HTTP
  response the same way `ServeHTTP` does (status code via
  `HTTPStatusForErrorCode`, `Content-Type`, `x-mvep-*` response headers, body)
  — that response-writing logic lives in the HTTP transport today, not in
  `ServeCmdReq`, so the convenience handler must replicate it (or call a
  shared helper extracted from `ServeHTTP`) rather than stopping at the
  `ServeCmdReq` call. Extracting that helper is preferred: two copies of the
  error-mapping and header-prefixing logic will drift.

  This reuses the full interceptor chain so the convenience URL has
  **identical auth posture** to calling `GetJobStatus` as a command. It must
  never read the job store directly, or it would silently bypass
  `AuthInterceptor`.

### Go client changes (`runtime/go/mvep/client/client.go`)

#### Prerequisite: a raw envelope send path

**No existing client API can send these commands.** Every send path today
resolves the command name and result type from a typed Go value:
`PackageHandler.SendCmdReq` calls `h.Package.NameOf(cmd)` and then
`h.Package.InstanceOf(cmdName + "Result")`, returning an error when the
lookup fails. `SubmitJob`/`GetJobStatus` are reserved runtime commands, not
registered package commands, so both lookups fail. The one raw-ish helper,
`PackageClient.SendRawCmd`, goes through the legacy `Transporter.TransportCmd`
interface, which has no header parameter at all — it can neither send
`job-cmd` nor read back `job-id`.

So T8 must first add a raw envelope path, e.g.:

```
func (p *PackageClient) SendEnvelope(ctx context.Context, req *mvep.CmdReq) (*mvep.CmdResp, error)
```

which builds nothing from the package registry, calls
`EnvelopeTransporter.TransportCmdReq` directly with the client's encoder mime
type, and — importantly — runs the **client interceptor chain** the way
`sendCmdReqInternal` does today. Skipping that would make job submissions the
only client calls that silently bypass client-side interceptors (auth
injection, logging, retries).

This helper is generally useful beyond jobs, but it is a real API addition
that the earlier draft assumed already existed.

#### Job helpers

All three helpers encode/decode the inner command with the client's
configured encoder (the same one `SendCmdReq` uses), so they work over JSON,
protobuf, protojson, or any registered encoder.

- `PackageClient.SubmitJob(ctx, cmd any, headers map[string]string) (jobID string, err error)`
  — encodes `cmd` via the client's encoder (as `SendCmdReq` already does),
  sends `CmdReq{Cmd: "SubmitJob", Headers: merge(headers, {"job-cmd": p.pkg.NameOf(cmd)}), Payload: encodedBytes}`
  through `SendEnvelope`, reads `jobID` from `resp.Headers["job-id"]`. No
  `SubmitJob`/`SubmitJobResult` wire struct is constructed or decoded.
- `PackageClient.GetJobStatus(ctx, jobID string) (*JobStatusResult, error)`
  — sends `CmdReq{Cmd: "GetJobStatus", Headers: {"job-id": jobID}, Payload: nil}`;
  returns a small **client-side** `JobStatusResult` struct (not a wire type)
  assembled entirely from response headers: `Status`, `Progress *JobProgress`
  (nil if the headers are absent), `Error *ErrorInfo` (from the
  `job-error-code`/`job-error-message` headers, **not** from `resp.Error`),
  and the opaque `Payload []byte`. A `failed` job returns a non-nil
  `JobStatusResult` and a **nil** Go error — the query succeeded. A nil
  `JobStatusResult` plus an error means the query itself failed
  (`job_not_found`, transport failure, …).
- `PackageClient.WaitForJob(ctx, jobID string, pollInterval time.Duration) (any, error)`
  — convenience polling loop; on `succeeded`, decodes `resp.Payload` into
  `p.pkg.InstanceOf(innerCmd + "Result")` using the client's encoder (the
  same `<CmdName>Result` convention `SendCmdReq` already uses, where
  `innerCmd` comes from `resp.Headers["job-cmd"]`); on `failed`, returns the
  structured `*ErrorInfo` as an error; respects `ctx` cancellation between
  polls.

Note that `GetJobStatus` returns the full result body on **every** poll once
the job succeeds, so a `WaitForJob` loop over a large result re-transfers it
only once (it stops at the first `succeeded`), but a caller polling manually
pays for it repeatedly. A `status-only` variant is deferred, not designed
here.

## Affected Files

| File | Change |
| --- | --- |
| `runtime/go/mvep/job.go` | New — all types, `JobStore`, `InMemoryJobStore`, `GenerateJobID`, header sanitization, `handleSubmitJob`/`runJob`/`handleGetJobStatus`/`SetJobProgress`/`Shutdown` |
| `runtime/go/mvep/job_test.go` | New — store transitions, retention sweep, sanitization, dispatch, security |
| `runtime/go/mvep/envelope.go` | `HTTPStatusForErrorCode` gains `job_not_found`/`job_queue_full`/`nested_job_forbidden`/`job_store_error` cases |
| `runtime/go/mvep/mvepackge.go` | New `PackageHandler` fields; `ServeCmdReq` core-handler switch |
| `runtime/go/mvep/http_transport.go` | Extract the `CmdResp`-to-`ResponseWriter` logic from `ServeHTTP` into a helper the `/jobs/{id}` route can reuse |
| `runtime/go/mvep/server/server.go` | `ServerConfig` fields; `PackageRegistration.Handler`; `PackageHandler.Shutdown` wired into `drain`; `/jobs/{id}` route |
| `runtime/go/mvep/server/lifecycle_test.go` | New — shutdown drains an in-flight job; `/jobs/{id}` auth parity; `Content-Type` parity |
| `runtime/go/mvep/client/client.go` | New `PackageClient.SendEnvelope` raw envelope path; `SubmitJob`/`GetJobStatus`/`WaitForJob` |
| `runtime/go/mvep/client/client_test.go` | New — submit/poll/wait round-trip |
| `runtime/go/mvep/server/SERVER.md` | New config fields, reserved command names, `/jobs/{id}` endpoint, in-memory-only and stored-credential caveats |
| `runtime/go/README.md` | Feature mention, if it enumerates features |
| `CHANGELOG.md` | `[Unreleased]` entries for `runtime/go` |

## Risks and Compatibility

- **Reserved-name shadowing.** A package that already defines a command
  literally named `SubmitJob` or `GetJobStatus` is shadowed once
  `EnableAsyncJobs` is on. Same class of risk as any reserved built-in name.
- **In-memory store is single-instance only.** A job submitted on one
  server instance is invisible to another. Must be documented prominently;
  the pluggable `JobStore` interface is the intended escape hatch, not
  shipped as a solved problem.
- **Goroutine/resource exhaustion.** `MaxConcurrentJobs` bounds concurrent
  execution, but a client that submits many small jobs in a burst still
  grows the in-memory store until `JobRetention` sweeps completed entries.
- **Unbounded job duration.** `JobTimeout` is optional by design, so a
  handler that ignores context cancellation can run indefinitely and block
  a clean `Shutdown` beyond its budget — the same documented limitation
  that already applies to `ShutdownContext` today.
- **Header/body size is already bounded.** `SubmitJob`'s body is the
  inner command's encoded bytes, already covered by the existing
  `MaxRequestBytes` limit in `ServeHTTP` — no new limit needed.
- **Fully encoder-independent.** The wrapped command and its result travel
  as opaque bytes in the envelope, and job metadata travels in `x-mvep-*`
  headers — mirroring how `CmdReq`/`CmdResp` already work. No encoder is
  invoked on the job envelope itself; encoders only touch the inner
  command/result types, exactly as they do for a synchronous call. The
  feature works with JSON, protobuf, protojson, and any future encoder,
  with no `RawPayload` trick and no `job_unsupported_encoder` path. (The
  production generated commands in `mvep.plain.go` are themselves
  plain structs today and so are JSON/CBOR-only regardless of this feature;
  protobuf works for `.proto`-generated types like the `iunet.pb.go` test
  fixture. The job feature inherits this existing reality rather than
  introducing a new restriction.)
- **Stored payload size.** Resolved: `MaxJobResultBytes` bounds a completed
  job's retained result, checked in `runJob` before `MarkSucceeded`. Without
  it, `MaxRequestBytes` bounds the input and `MaxResponseBytes` bounds what a
  client reads, but nothing bounds what sits in the store between completion
  and the retention sweep.
- **Submitter credentials are stored at rest.** `Job.Headers` holds the
  submitter's `auth` token so the inner command can authenticate as them when
  it eventually runs. This is load-bearing for the auth guarantee, but it
  means bearer tokens live in the job store for the job's lifetime plus
  `JobRetention`. In-memory that is a process-memory exposure; with a future
  Redis/DB store it becomes credentials persisted to disk. Consequences to
  document explicitly in `SERVER.md`: tokens must not be logged when a job is
  dumped for debugging, and a third-party `JobStore` implementer is now
  handling secrets. A token-exchange or job-scoped-credential design would
  avoid this but is out of scope for v1.
- **Auth is evaluated at execution time, not submission time.** A token that
  is valid at submit can be expired or revoked when the job actually runs
  (after queueing, or on a retry), so a job can fail `unauthorized` long
  after a 200 submit. Conversely, a job already running is not re-checked, so
  revoking a token does not stop it. Both are inherent to replaying the
  submitter's credential; document rather than fix.
- **Message headers are attacker-influenced text.** `job-progress-message`
  and `job-error-message` originate in handler code and inner-command errors
  and are written into HTTP response headers. Sanitization and truncation
  (see the wire model above) are mandatory, not cosmetic.
- **No idempotency key.** A client that retries `SubmitJob` after a timeout
  creates a second job running the same command. v1 offers no dedupe; callers
  needing exactly-once must make the wrapped command idempotent. A
  client-supplied `job-idempotency-key` header is a plausible follow-up.
- **`ErrorInfo.Code` does not survive HTTP.** Pre-existing, not introduced
  here, but it constrains this design: HTTP clients switch on status codes,
  not on `job_not_found`. See the wire model section.

## Verification

1. `go test ./runtime/go/mvep/... -count=1` and
   `go test -race ./runtime/go/mvep/... -count=1` (goroutines plus a shared
   map make `-race` essential here).
2. `go test -run TestJob_AuthPolicyNotBypassable ./runtime/go/mvep/` — must
   fail if `runJob` is changed to call `executeCmd` instead of `ServeCmdReq`.
3. `go test -run TestJob_SubmitReturnsBeforeInnerCompletes` — a
   channel-gated fake command proves `SubmitJob` returns before the wrapped
   command finishes.
4. `go test -run TestServer_ShutdownDrainsInFlightJob ./runtime/go/mvep/server/`
   — `ShutdownContext` waits for a running job, bounded by the shutdown
   context.
5. `go test -run TestJobsEndpoint_SameAuthAsCommand ./runtime/go/mvep/server/`
   — with an `AuthInterceptor` installed, `GET /jobs/{id}` without
   `x-mvep-auth` is rejected `unauthorized`, and **with** a valid
   `x-mvep-auth` it succeeds. The second half is the one that catches the
   dropped-headers bug: a handler that passes only `{"job-id": id}` fails it.
   (Note: `AuthInterceptor` in `interceptors.go` is token-presence plus a
   `TokenValidator` — there is no per-command `OnlyCommands` policy in the
   runtime, contrary to an earlier draft of this plan. Per-command
   authorization, if tested, must be supplied by a test-local interceptor.)
6. `go test -run TestJob_StatusOfFailedJobIsHTTP200 ./runtime/go/mvep/server/`
   — polling a job that failed with an `unauthorized` inner error returns
   HTTP 200 with `x-mvep-job-status: failed` and
   `x-mvep-job-error-code: unauthorized`, **not** HTTP 401.
7. `go test -run TestJobsEndpoint_ContentTypeMatchesSubmitEncoder ./runtime/go/mvep/server/`
   — a job submitted as `application/x-protobuf` is served from
   `GET /jobs/{id}` with `Content-Type: application/x-protobuf`.
8. Manual `curl`: `POST /api/{pkg}/cmd` with `x-mainvec-cmd: SubmitJob` and
   `x-mvep-job-cmd: <RealCommand>` → capture `x-mvep-job-id` from the
   response header → `GET /api/{pkg}/jobs/{jobId}` a few times, observing
   `x-mvep-job-status: pending → running → succeeded` and the result body.
9. `go test -run TestJob_ProtobufRoundTrip ./runtime/go/mvep/` — submit a
   job over `application/x-protobuf` using the `iunet.pb.go` test fixture
   (a real `proto.Message`) and confirm the result comes back decodable as
   protobuf. This proves encoder-independence and would have failed under
   the earlier JSON-only design.
10. `go vet ./runtime/go/... && go build ./runtime/go/...`.

## Rollout

1. Ship as **three sequential PRs** against this issue, not one. The
   original single-PR plan bundled the handler, the server lifecycle, and a
   new client API surface into one review; each stage below is independently
   reviewable and independently inert:
   - **PR 1 — core**: `job.go` (types, store, sanitization, ID generation),
     the `envelope.go` error codes, and `PackageHandler` dispatch. Usable
     in-process via `ServeCmdReq`; no HTTP surface change.
   - **PR 2 — server**: `ServerConfig` plumbing,
     `PackageRegistration.Handler`, `PackageHandler.Shutdown` wired into
     drain, the shared response-writer extracted from `ServeHTTP`, and the
     `/jobs/{id}` route.
   - **PR 3 — client**: `SendEnvelope` plus the three job helpers.
2. The feature is inert unless `EnableAsyncJobs` is explicitly set.
3. Keep `EnableAsyncJobs` off by default through at least one release.
4. Follow-up issues, not part of this plan:
   - `CancelJob` command and cooperative cancellation contract.
   - A persistent/distributed `JobStore` implementation (Redis or similar),
     shipped as a separate package so `runtime/go/mvep` stays dependency-free.
   - TypeScript client-side job submission and polling helpers.
   - Round-tripping `ErrorInfo.Code` over HTTP (a transport fix, not a job
     fix) so clients can switch on stable codes instead of statuses.
   - `job-idempotency-key` for safe submit retries.
   - A status-only poll variant that omits the result body.

## Decision Log

- **Wrapper command, not a header flag or spec flag.** A reserved
  `SubmitJob{Cmd, Payload}` command needs no transport-level branching on
  the same request/response shape, and needs no toolkit/codegen change,
  unlike a per-command spec flag.
- **Both a command and a URL for polling.** `GetJobStatus` preserves the
  normal interceptor/auth chain for programmatic clients; `GET /jobs/{id}`
  is a convenience for curl/dashboards, but is implemented by recursing
  through `ServeCmdReq` so it carries identical auth semantics rather than
  becoming a second, weaker code path.
- **In-memory `JobStore` behind a pluggable interface, not a bundled
  persistent store.** Mirrors the philosophy of keeping the runtime
  storage-agnostic: it defines the seam, applications supply persistence
  if their deployment topology needs it.
- **`JobStore` uses explicit state-transition methods, not a mutate
  closure.** A closure argument doesn't serialize across a network boundary,
  so a future remote-backed store would have nowhere to hook in.
- **Reject-immediately on `job_queue_full`, no internal queueing.** Simplest
  correct v1 behavior; queueing/backpressure is deferred until there's a
  concrete need.
- **`JobTimeout` optional, unlike the (unimplemented) CmdChain plan's
  mandatory `ChainTimeout`.** A chain's lock-duration risk doesn't apply
  here; a job running long is the entire point of the feature.
- **No relationship to CmdChain.** plans/019 is draft and unimplemented.
  This plan does not block on it, share a dispatch registry with it, or
  define nesting rules against it — any resemblance in shape is
  independently derived.
- **No new third-party dependency for ID generation.** `crypto/rand`-backed
  `GenerateJobID`; the workspace has no existing UUID dependency to reuse.
- **Toolkit and TypeScript runtime untouched in v1.** Keeps this a
  contained, runtime-only, opt-in change with zero regeneration required
  for any existing generated package.
- **`job_queue_full` maps to HTTP 429, not 503.** The server itself is
  healthy — only this handler's concurrency budget is exhausted — so 429
  ("too many requests", retryable) fits better than 503 ("service
  unavailable").
- **`/jobs/{id}` is a per-package literal route, not a `{pkg}` wildcard.**
  Matches the existing `cmdPath` registration model exactly (one route per
  registered package) and needs no package-name validation in the handler,
  unlike a catch-all route would.
- **Encoder-independent envelopes, not a typed `SubmitJob`/`GetJobStatusResult`
  wire struct.** `CmdReq`/`CmdResp` are already plain structs never passed to
  an encoder; the job feature reuses that property — wrapped command as
  opaque `[]byte` payload, job metadata in `x-mvep-*` headers. This avoids
  the JSON-only trap a hand-written wire struct would fall into (the
  protobuf/protojson encoders require `proto.Message`), works with every
  encoder today, and needs no `RawPayload`/JSON-allowlist/
  `job_unsupported_encoder` machinery.
- **`job_store_error` for `JobStore` operational failures.** `Create`/
  `MarkRunning`/`MarkSucceeded`/`MarkFailed`/`Get`/`SetProgress` can all
  return errors; without a dedicated code they'd surface as a misleading
  generic 500 with no stable `ErrorInfo.Code` for clients to switch on.
- **A failed job is an HTTP 200 status response, not a `CmdResp.Error`.**
  Reversed from the first draft. `ServeHTTP` discards `CmdResp.Payload` on
  error, derives the HTTP status from the *inner* command's code, and makes
  the client's send path return a Go error — so "the job failed" and "your
  poll was rejected" became indistinguishable. Job failure now travels as
  `job-status: failed` + `job-error-code`/`job-error-message` headers.
  `CmdResp.Error` is reserved for query-level failures.
- **The plan does not fix `ErrorInfo.Code` propagation over HTTP.** The
  transport rewrites it to `http_<status>` on the client side. Fixing that
  is a transport-wide behavior change affecting every existing error code,
  so it belongs in its own issue; this design works within the limitation
  instead of quietly depending on a fix.
- **`Job.Encoder` is recorded at submit time.** Needed twice: to replay the
  inner command through `ServeCmdReq`, and to serve `GET /jobs/{id}` with a
  truthful `Content-Type`. The first draft called the encoder "immaterial"
  for the convenience route, which would have served protobuf bytes labeled
  `application/json`.
- **`job-*` headers are stripped before replay; inner response headers are
  dropped.** Prevents job metadata leaking into an ordinary command's header
  namespace on the way in, and prevents an inner command overwriting
  `job-status` on the way out.
- **A raw envelope client path (`SendEnvelope`) is a prerequisite, not an
  incidental.** Every existing client send path resolves `NameOf(cmd)` and
  `InstanceOf(cmd+"Result")` against the package registry, which reserved
  commands are not in; `SendRawCmd` uses the legacy header-less transport.
  The first draft assumed an API that does not exist.
- **`MaxJobResultBytes` enforced in `runJob`, not in `JobStore`.** Keeps the
  cap uniform across third-party store implementations instead of relying on
  each to enforce it.

## Open Questions

These should be resolved before this plan leaves draft:

1. **Numeric defaults.** `DefaultMaxConcurrentJobs` (proposed 100),
   `DefaultMaxJobResultBytes` (proposed 4 MiB), `MaxJobMessageBytes`
   (proposed 512), and the default `JobRetention` (proposed 10 minutes) are
   placeholders — do these need to be configurable per-deployment defaults
   instead of constants?
2. ~~**Issue filing.**~~ **Resolved:** [#21](https://github.com/mainvec/mvep/issues/21).
3. ~~**Independent size cap on stored job payloads.**~~ **Resolved: yes.**
   `MaxJobResultBytes`, checked in `runJob` before `MarkSucceeded`; over the
   cap the job becomes `failed` with `job_result_too_large`.
4. ~~**Is storing the submitter's bearer token acceptable for v1?**~~
   **Resolved: yes for v1.** Async execution replays the submitter's
   credential as-is; the credential-at-rest risk is documented prominently
   in `SERVER.md` as a security caveat. A short-lived job-scoped credential
   is deferred to a follow-up issue.

## Progress

> Provisional. Do not begin until this plan leaves draft and an issue number
> is assigned. T1–T4 are PR 1, T5–T6 are PR 2, T7–T8 are PR 3, T9 lands with
> whichever PR is last.

- [x] T1 — Add failing tests for `InMemoryJobStore` transitions/retention sweep, header sanitization, and the `HTTPStatusForErrorCode` job cases
- [x] T2 — Implement `runtime/go/mvep/job.go` core types, in-memory store, sanitization helper, and the `HTTPStatusForErrorCode` cases in `envelope.go`
- [x] T3 — Add failing tests for `SubmitJob`/`GetJobStatus` dispatch, guards, failed-job-is-not-`CmdResp.Error`, the auth-bypass regression, and protobuf round-trip
- [x] T4 — Implement `PackageHandler` async job dispatch (`handleSubmitJob`, `runJob`, `handleGetJobStatus`, `SetJobProgress`, `Shutdown`)
- [x] T5 — Add failing tests for shutdown draining an in-flight job, `/jobs/{id}` auth parity (both directions), and `Content-Type` matching the submit encoder
- [x] T6 — Extract the shared response writer from `ServeHTTP`; implement `ServerConfig`/`PackageRegistration.Handler`/drain wiring/per-package `/jobs/{id}` route
- [x] T7 — Add failing round-trip tests for `SendEnvelope` and the Go client job helpers
- [x] T8 — Implement `PackageClient.SendEnvelope`, then `SubmitJob`/`GetJobStatus`/`WaitForJob`
- [ ] T9 — Update `SERVER.md`, `README.md`, and `CHANGELOG.md`

## Tasks

### T1 — `InMemoryJobStore`, sanitization, and error-code tests

- **Outcome**: Store-transition, sanitization, and error-code behavior is
  specified before any implementation exists.
- **Verification**: Tests fail to compile/run because `job.go` doesn't exist
  yet.
- **Notes**: Cover retention sweep only affecting completed jobs, that
  pending/running jobs are never swept regardless of age, and a table-driven
  case for each new `HTTPStatusForErrorCode` entry (`job_not_found`→404,
  `job_queue_full`→429, `nested_job_forbidden`→400, `job_store_error`→500)
  plus an assertion that `job_result_too_large` is deliberately **absent**.
  Sanitization cases: embedded `\r\n`, non-ASCII, and over-length messages.
  No `RawPayload` round-trip test — that type no longer exists.

### T2 — Core types, in-memory store, and error-code mapping

- **Outcome**: `job.go` compiles; T1 tests pass; `HTTPStatusForErrorCode` in
  `envelope.go` maps `job_not_found`→404, `job_queue_full`→429,
  `nested_job_forbidden`→400, `job_store_error`→500 instead of falling
  through to 500.
- **Verification**: `go test ./runtime/go/mvep/ -run 'TestInMemoryJobStore|TestHTTPStatusForErrorCode|TestJobHeaderSanitize'`.
- **Notes**: `GenerateJobID` uses `crypto/rand`; no new dependency. The
  `envelope.go` edit is easy to forget since it's adjacent to, not inside,
  `job.go` — add an explicit table-driven case for each new code.
  `MarkSucceeded` takes no headers argument; `Job` has `Encoder`, `Headers`,
  `Payload` and no `ResultHeaders`.

### T3 — Dispatch and guard tests

- **Outcome**: Submit-returns-immediately, status transitions, nested-job
  rejection, unknown-inner-command, disabled-feature, queue-full,
  post-shutdown submit, oversized result, failed-job signalling, and
  auth-not-bypassable are all specified as failing tests.
- **Verification**: Tests fail because `SubmitJob`/`GetJobStatus` aren't
  dispatched yet.
- **Notes**: The auth-bypass test is the reason `runJob` must call
  `ServeCmdReq`, not `executeCmd` — it must fail if that changes. Use a
  test-local interceptor that authorizes per command name; the runtime has no
  `OnlyCommands` policy. A separate test must assert that a failed job's
  status response has `resp.Error == nil` and carries `job-error-code` — that
  is the regression guard for the HTTP-200 decision. The protobuf round-trip
  test submits a job over `application/x-protobuf` using the `iunet.pb.go`
  fixture and proves encoder-independence. Also assert `job-*` headers are
  stripped from the replayed inner request.

### T4 — `PackageHandler` dispatch implementation

- **Outcome**: T3 tests pass; `SubmitJob` returns before the wrapped command
  finishes; `GetJobStatus` reflects live state; `PackageHandler.Shutdown`
  exists and is safe to call with no jobs ever submitted.
- **Verification**: `go test -race ./runtime/go/mvep/...`.
- **Notes**: Lazy `sync.Once` init keeps existing struct-literal
  construction of `PackageHandler` working unchanged, but `Shutdown` must run
  the same init path (so `jobCancel` is never nil) and set `jobsClosed` so a
  post-shutdown submit is rejected cleanly instead of starting a job on a
  dead context. `Shutdown` lives here rather than in the server package so
  the handler is self-contained for in-process users; PR 2 only wires it in.

### T5 — Server lifecycle and endpoint tests

- **Outcome**: Shutdown-drains-job, endpoint-auth-parity, and
  `Content-Type`-matches-submit-encoder behavior specified as failing tests.
- **Verification**: Tests fail because the `/jobs/{id}` route doesn't exist
  yet (`PackageHandler.Shutdown` already landed in T4, so its test here is
  about the server's drain wiring, not the method).
- **Notes**: Reuse the `newTestServer` helper from `lifecycle_test.go`. The
  auth-parity test must assert **both** directions — rejected without
  `x-mvep-auth`, accepted with it — since a handler that forwards only
  `{"job-id": id}` passes a rejection-only test while being completely
  broken.

### T6 — Server lifecycle and endpoint implementation

- **Outcome**: T5 tests pass; `ShutdownContext` waits for in-flight jobs up
  to its budget; `GET {BasePath}/{pkgName}/jobs/{jobId}` (registered per
  package in `RegisterPackage`, literal package name, same model as
  `cmdPath`) has identical auth posture to `GetJobStatus`.
- **Verification**: `go test ./runtime/go/mvep/server/...`.
- **Notes**: Extract the `CmdResp`-to-`http.ResponseWriter` logic out of
  `ServeHTTP` into a shared helper first, then use it from both places —
  duplicating the error-mapping and `x-mvep-*` prefixing will drift. The
  endpoint must call `ServeCmdReq`, never read the job store directly, must
  forward the extracted `x-mvep-*` request headers (not just `job-id`), must
  not read/set `x-mainvec-cmd`, and must set the response `Content-Type` from
  the `job-encoder` response header rather than a hardcoded JSON default.

### T7 — Client round-trip tests

- **Outcome**: Submit → poll → wait-for-result specified as failing tests,
  over at least two encoders (JSON and protobuf via the `iunet.pb.go` fixture).
- **Verification**: Tests fail because `SendEnvelope` and the job methods
  don't exist yet.
- **Notes**: Exercise both `GetJobStatus` and `WaitForJob` against a real
  test server; confirm `GetJobStatus` reads status from response headers
  and `WaitForJob` decodes the opaque result `Payload` with the same
  encoder used to submit. Add a case asserting that polling a **failed** job
  returns a populated `JobStatusResult` with a nil Go error, and that a
  client interceptor observes the `SubmitJob` request (guarding against
  `SendEnvelope` skipping the chain).

### T8 — Client implementation

- **Outcome**: T7 tests pass.
- **Verification**: `go test ./runtime/go/mvep/client/...`.
- **Notes**: Add `PackageClient.SendEnvelope` **first** — no existing send
  path can carry a reserved command (`SendCmdReq` needs `NameOf`/`InstanceOf`
  registry hits; `SendRawCmd` uses the header-less legacy transport). It must
  run the client interceptor chain the way `sendCmdReqInternal` does.
  `SubmitJob` then encodes `cmd` with the client's encoder and sends it as the
  `CmdReq.Payload` (opaque bytes), with `job-cmd` in headers. `GetJobStatus`
  returns a client-side `JobStatusResult` struct assembled from response
  headers (not a wire type), reading failure from `job-error-code`/
  `job-error-message`, not from `resp.Error`. `WaitForJob` reads the inner
  command name from `resp.Headers["job-cmd"]` and decodes `resp.Payload`
  via `p.pkg.InstanceOf(innerCmd + "Result")` using the client's encoder —
  the same `<CmdName>Result` convention `SendCmdReq` already uses.

### T9 — Documentation

- **Outcome**: `SERVER.md`, `README.md`, and `CHANGELOG.md` describe the new
  config fields, reserved command names, the `/jobs/{id}` endpoint, the
  in-memory-only caveat, and the stored-credential caveat.
- **Verification**: Manual review.
- **Notes**: Mirror the tone and depth of existing `SERVER.md` sections. The
  credential-at-rest note is a security caveat, not a footnote — anyone
  implementing a third-party `JobStore` needs to know they are handling
  bearer tokens.
