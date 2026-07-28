# Plan 016 — Caller-owned MVEP server lifecycle

- **Issue**: [#16](https://github.com/mainvec/mvep/issues/16) — feat(runtime/go): caller-owned server lifecycle
- **Branch**: `feat/016-server-lifecycle`
- **Target release**: `runtime/go/v0.8.0`

## Problem / Goal

The Go runtime's `mvep/server` package currently owns process-wide
`SIGINT`/`SIGTERM` handling. `Server.Start()` launches untracked
`http.Serve` goroutines and then waits on a private channel that only the
internal signal goroutine can release. `Server.Shutdown()` merely closes
listeners: it does not unblock `Start()`, invoke `OnShutdown`, drain active
requests, stop signal subscriptions, or surface errors returned by
`http.Serve`.

This makes the server difficult to embed in daemons, desktop applications,
mobile bindings, tests, and processes containing more than one server. The
owning application—not a library—must decide which signals matter, how long
shutdown may take, and how HTTP draining is sequenced with application cleanup.

The goal is a single-use, concurrency-safe MVEP server lifecycle in which:

1. mvep binds listeners and serves HTTP;
2. the owning application handles signals and other shutdown triggers;
3. the owner calls mvep shutdown as one explicit step in its larger shutdown
   sequence; and
4. every start, serve, and shutdown result is observable without polling.

## Goals

- Remove all `os/signal`, `syscall`, SIGINT, and SIGTERM handling from
  `mvep/server`.
- Keep `Start()` as a blocking API, but make it wait for explicit shutdown or a
  fatal serve error rather than a process signal.
- Add `StartAsync()` that binds all listeners synchronously and returns only
  when the server is ready to accept requests.
- Add `Wait()`, `Done()`, and `Err()` so owners can coordinate server lifetime
  without readiness polling or private error channels.
- Add context-aware graceful shutdown backed by `http.Server.Shutdown(ctx)`.
- Preserve `Shutdown() error` as a bounded compatibility wrapper and add
  `ShutdownContext(context.Context) error` as the preferred API.
- Export `DefaultShutdownTimeout` as the single documented budget used by
  `Shutdown()` and by internally-initiated shutdown.
- Make shutdown idempotent and safe when called concurrently.
- Track and propagate unexpected `Serve` errors; one failed listener shuts down
  the remaining listeners.
- Bind all configured listeners transactionally: a startup failure closes every
  listener already acquired during that attempt.
- Add `GetListeners()` while retaining `GetListener()` as the primary-listener
  compatibility helper.
- Define and test listener ownership, state transitions, timeout behavior, and
  shutdown error semantics.
- Update runtime documentation and toolkit guidance to show caller-owned signal
  handling.

## Non-goals

- Application-specific signal selection or process exit codes.
- Application cleanup ordering beyond the HTTP lifecycle; engines, databases,
  child processes, and persistence remain the owner's responsibility.
- A built-in `/stop` endpoint. Applications may register one, but must invoke
  shutdown asynchronously after writing the response.
- Unix-socket stale-file detection, filesystem permissions, ownership, or
  directory creation. Callers that need these policies should pass a
  pre-created listener.
- Certificate loading or certificate rotation. Existing pre-created TLS
  listeners remain supported.
- Changes to the MVEP command wire format, client API, encoders, interceptors,
  package routing, health response, or TypeScript runtime.
- Migrating Zirafa, Linkvec, Droy, or other downstream applications in this PR.
  Those are follow-up changes after the new runtime is tagged.

## Proposed Public API

```go
// DefaultShutdownTimeout bounds Shutdown() and every shutdown mvep initiates
// itself (fatal serve error, blocking Start teardown).
const DefaultShutdownTimeout = 30 * time.Second

type ServerConfig struct {
    Listeners         []ListenerConfig
    ListenAddress     string // deprecated; unchanged
    BasePath          string
    EnableHealthCheck bool
    HealthCheckPath   string
    EnableCORS        bool
    Interceptor       mvep.CmdInterceptor

    // Removed in v0.8.0:
    // OnShutdown func()
}

// Start binds listeners, starts serving, and blocks until the server has
// completely stopped. It is equivalent to StartAsync followed by Wait.
func (s *Server) Start() error

// StartAsync binds every configured listener synchronously, starts serving,
// and returns when all listeners are ready. The server is single-use.
func (s *Server) StartAsync() error

// Wait blocks until shutdown or fatal serve failure completes, then returns
// the final joined lifecycle error. It may be called multiple times.
func (s *Server) Wait() error

// ShutdownContext stops accepting requests, drains active requests until ctx
// expires, force-closes remaining connections on timeout, and waits for the
// server to stop. Concurrent calls are safe and observe the same shutdown.
func (s *Server) ShutdownContext(ctx context.Context) error

// Shutdown is a compatibility wrapper bounded by DefaultShutdownTimeout.
func (s *Server) Shutdown() error

// Done closes exactly once after the final lifecycle result is available. It
// never closes for a server that is neither started nor shut down.
func (s *Server) Done() <-chan struct{}

// Err returns the final lifecycle error after Done closes; while running it
// returns nil.
func (s *Server) Err() error

// GetListener retains the existing primary-listener API.
func (s *Server) GetListener() net.Listener

// GetListeners returns a copy of every bound listener in configuration order.
func (s *Server) GetListeners() []net.Listener
```

`OnShutdown` is removed rather than redefined. Cleanup callbacks obscure
application ordering and can recurse into `Server.Shutdown`, which is especially
dangerous in the current Linkvec-style bi-directional shutdown pattern. The
owner should express cleanup explicitly around `ShutdownContext`.

This is a behavior and source change, so it targets `runtime/go/v0.8.0` rather
than a v0.7 patch.

## Lifecycle Contract

### States

```text
new ──StartAsync──> starting ──all binds succeed──> running
 │                       │
 │                       └─bind/setup failure──────────────┐
 │                                                        v
 └─ShutdownContext──────────────────────────────────────> stopped

running ──ShutdownContext / fatal Serve error──> stopping ──> stopped
```

- A `Server` is single-use. Calling `Start` or `StartAsync` more than once, or
  after shutdown, returns a stable lifecycle error.
- `StartAsync` serializes against shutdown. Shutdown requested while startup is
  in progress begins after startup either commits or rolls back.
- Shutdown before start is allowed, is idempotent, transitions directly to
  `stopped`, and prevents later start.
- `Done` closes once. The final error is stored before `Done` closes.
- `Wait` supports multiple concurrent waiters and is repeatable after the
  lifecycle ends.
- `Done`/`Wait` are only released by a terminal transition: startup failure,
  fatal serve error, or shutdown. A server that is constructed and never started
  has no terminal transition, so `Wait()` blocks and `Done()` never closes. This
  is intentional: readiness and completion are start-driven, and adding an
  implicit "never started" completion would make `Wait` ambiguous for owners
  that construct a server before deciding to start it. Owners that must unblock
  such a server call `Shutdown`/`ShutdownContext`, which transitions it to
  `stopped`.

### Listener ownership

- Once `StartAsync` begins, the server owns both automatically-created and
  caller-provided `ListenerConfig.Listener` values.
- If any listener fails to resolve or bind, the server closes all listeners
  already acquired in that startup attempt, records the startup error, and
  transitions to `stopped`.
- `GetListeners()` returns configuration order and never exposes the mutable
  backing slice.
- Closing a server-owned listener externally is treated as an unexpected serve
  failure and shuts down the other listeners.

### Normal shutdown

1. The first caller initiates shutdown; subsequent calls join the same lifecycle.
2. All listener-specific `http.Server` instances receive
   `http.Server.Shutdown(ctx)` concurrently.
3. New connections are rejected while active handlers drain.
4. Serve goroutines finish and expected `http.ErrServerClosed` values are
   ignored.
5. The final joined error is stored and `Done` closes.

The context supplied by the first shutdown initiator is the global graceful
shutdown budget. A later caller may stop waiting when its own context expires,
but it does not replace the in-progress shutdown budget.

If the initiating context expires, mvep calls `http.Server.Close()` on any
remaining servers so lifecycle completion is bounded. `ShutdownContext`
returns an error containing the context error and any close/serve errors.
Handlers that ignore request-context cancellation cannot be forcibly terminated
by Go; this limitation must be documented.

### Fatal serve failure

An error from any `http.Server.Serve` other than `http.ErrServerClosed` is
recorded as the primary lifecycle error and initiates bounded shutdown of every
remaining listener. `Wait()` and `Err()` expose the failure. Normal shutdown
must not report listener-close noise as a serve failure.

Because no caller context exists on this path, mvep initiates the shutdown with
its own `context.WithTimeout(context.Background(), DefaultShutdownTimeout)`. The
same budget applies to any other mvep-initiated shutdown, so `Shutdown()` and
serve-error teardown share one documented bound.

### Shutdown from an HTTP handler

Calling `ShutdownContext` synchronously from a handler served by the same
server can wait for that handler and deadlock until the context expires. A stop
handler must write its response and then trigger shutdown asynchronously:

```go
srv.Handle("/stop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("stopping\n"))

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
        defer cancel()
        _ = srv.ShutdownContext(ctx)
    }()
}))
```

## Internal Design

The server owns one `http.Server` per listener because each listener may have
different middleware and all servers share the same mux:

```go
type serverState uint8

const (
    stateNew serverState = iota
    stateStarting
    stateRunning
    stateStopping
    stateStopped
)

type Server struct {
    config   ServerConfig
    mux      *http.ServeMux
    packages []*PackageRegistration

    lifecycleMu sync.Mutex
    state       serverState
    listeners   []net.Listener
    httpServers []*http.Server

    serveWG      sync.WaitGroup
    shutdownOnce sync.Once
    done         chan struct{}

    errMu sync.Mutex
    err   error
}
```

Implementation notes:

- Copy and normalize `ServerConfig` in `NewServer`; do not mutate the caller's
  configuration object and do not retain the caller's pointer. The current
  implementation stores `config *ServerConfig` and writes
  `config.HealthCheckPath` in place, so later caller mutations race with the
  server; v0.8 dereferences once into a value field.
- `Listeners` is copied into a server-owned slice so caller mutation after
  `NewServer` cannot change what gets bound.
- Initialize `done` in `NewServer` so callers may select on it immediately.
- Register the health handler exactly once before listeners begin serving.
- `StartAsync` holds the lifecycle transition lock through listener acquisition,
  then launches all serve goroutines before publishing `stateRunning`.
- Construct an `http.Server{Handler: handler}` for every listener instead of
  calling package-level `http.Serve`.
- An internal `recordError` preserves errors without races. Use `errors.Join`
  where more than one independent lifecycle error matters.
- Shutdown work runs exactly once. Concurrent callers wait on `Done` or their
  own context.
- Do not hold lifecycle or error mutexes while calling `http.Server.Shutdown`,
  `Close`, or waiting on goroutines.
- The package must no longer import `os`, `os/signal`, or `syscall`.

## Owner Usage

The recommended daemon pattern becomes:

```go
srv, err := server.NewServer(config)
if err != nil {
    return err
}
if err := srv.RegisterPackage(pkg, runner); err != nil {
    return err
}
if err := srv.StartAsync(); err != nil {
    return err
}

signalCtx, stopSignals := signal.NotifyContext(
    context.Background(),
    syscall.SIGINT,
    syscall.SIGTERM,
)
defer stopSignals()

select {
case <-signalCtx.Done():
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    // Stop accepting commands and drain active handlers first.
    serverErr := srv.ShutdownContext(shutdownCtx)

    // Then clean up application resources explicitly.
    applicationErr := engine.Shutdown(shutdownCtx)
    return errors.Join(serverErr, applicationErr)

case <-srv.Done():
    // A fatal serve/start error stopped the HTTP surface.
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    applicationErr := engine.Shutdown(shutdownCtx)
    return errors.Join(srv.Err(), applicationErr)
}
```

Applications may choose a different order where their domain requires it, but
mvep itself does not make that policy decision.

## Compatibility and Migration

### Preserved

- `NewServer`, `RegisterPackage`, `Handle`, `Start`, `Shutdown`, and
  `GetListener` remain available.
- Blocking `Start()` remains blocking.
- Listener address parsing, per-listener middleware, CORS, health checks,
  package routing, and interceptors retain their existing behavior.
- No wire or client compatibility change.

### Changed

- `Start()` no longer subscribes to or waits for OS signals. Without an owner
  calling `Shutdown`, it continues serving until a fatal serve error or process
  termination.
- `Shutdown()` now genuinely drains HTTP requests, bounded by
  `DefaultShutdownTimeout`, and unblocks `Start()`.
- `OnShutdown` is removed from `ServerConfig`.
- Pre-created listeners are explicitly owned and closed by the server after
  startup begins.
- `NewServer` no longer retains or mutates the caller's `*ServerConfig`. Callers
  that relied on editing the config struct after construction (including reading
  back the defaulted `HealthCheckPath`) must set values before `NewServer`.

### Downstream migration

Each daemon using mvep/server must:

1. remove `ServerConfig.OnShutdown`;
2. install signal handling in its executable/process owner;
3. call `ShutdownContext` with a bounded context;
4. run application cleanup explicitly in the required order;
5. replace `go srv.Start()` plus `GetListener()` polling with `StartAsync()`;
6. use `Wait`, `Done`, or `Err` for completion/failure reporting; and
7. ensure HTTP stop handlers initiate shutdown asynchronously.

Zirafa is the first intended consumer. Linkvec and Droy require separate audits
because they currently contain callback-based or bidirectional shutdown wiring.

## Affected Files

| File | Change |
| --- | --- |
| `runtime/go/mvep/server/server.go` | Replace signal-driven lifecycle with state, tracked `http.Server` values, async start/wait, graceful shutdown, serve-error propagation, and all-listener access |
| `runtime/go/mvep/server/server_test.go` | Replace sleep/poll/signal assumptions with deterministic lifecycle tests |
| `runtime/go/mvep/server/SERVER.md` | Rewrite lifecycle and examples around caller-owned signals |
| `runtime/go/README.md` | Update basic server example and configuration reference |
| `toolkit/MVEP_SKILL.md` | Update generated/manual server guidance; remove `OnShutdown` example |
| `CHANGELOG.md` | Document the v0.8 lifecycle change and migration requirement |
| `docs/server-lifecycle-migration.md` | Add focused v0.7 → v0.8 migration guide for daemon owners |

No TypeScript runtime files or Go client files should change.

## Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Existing daemons relied on mvep catching SIGTERM | Breaking change is explicit in v0.8; migration guide provides a complete `signal.NotifyContext` owner example |
| `Start()` remains blocked because shutdown does not close completion state | Regression test requires programmatic shutdown to release blocking `Start()` within a deadline |
| Active requests are dropped during nominal shutdown | Test a blocked handler and assert shutdown waits until the handler is released |
| Shutdown can hang forever | Context-aware API plus forced `Close` on deadline; `Shutdown()` and mvep-initiated shutdown both use `DefaultShutdownTimeout` |
| Owner calls `Wait()` on a never-started server and hangs | Documented in the lifecycle contract and in godoc for `Wait`/`Done`; `Shutdown` on a new server is the documented escape |
| Stop endpoint deadlocks itself | Documentation and integration test require response-first, asynchronous shutdown |
| One listener fails while others continue silently | Any unexpected serve failure records the error and shuts down every listener |
| Partial startup leaks listeners | Transactional bind test verifies earlier listeners close when a later bind fails |
| Concurrent shutdown races or closes `Done` twice | State mutex + once-only shutdown; run the full runtime under `go test -race` |
| Callers mutate the returned listener slice | `GetListeners` returns a copy; test mutation does not affect server state |
| `OnShutdown` removal breaks keyed literals | Expected source break for v0.8; migration is mechanical and documented |

## Verification

### Focused tests

Add deterministic tests for:

1. `StartAsync` returns with a port-zero listener immediately reachable—no
   polling or sleeps.
2. Blocking `Start` returns after programmatic `ShutdownContext`.
3. `ShutdownContext` is idempotent with many concurrent callers.
4. `Done` closes once and `Wait` is repeatable for multiple waiters.
5. Graceful shutdown waits for an active handler to complete.
6. Deadline expiration force-closes connections and returns a context-bearing
   error.
7. An asynchronous `/stop` handler returns 200 and then stops the server.
8. Multiple listeners are all reachable and all stop.
9. `GetListeners` returns all actual bound addresses in configuration order and
   returns a defensive copy.
10. Middleware remains listener-specific.
11. A fatal `Serve` error on one listener appears in `Wait`/`Err` and stops the
    other listeners.
12. Failure to bind a later listener rolls back earlier listeners.
13. Shutdown before start is safe and prevents later start, and releases a
    waiter blocked in `Wait()` on the never-started server.
14. Repeated start returns the documented lifecycle error.
15. Existing health, CORS, package registration, pre-created listener, and
    deprecated `ListenAddress` behavior remains intact.
16. `NewServer` does not mutate the caller's `ServerConfig` and ignores later
    caller mutations of the config struct and its `Listeners` slice.

Tests must use deadlines/channels instead of unbounded sleeps. Any test-created
listener and server must be closed with `t.Cleanup`.

### Commands

From `runtime/go`:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

From the repository root:

```bash
go test ./runtime/go/... ./toolkit/...
go vet ./runtime/go/... ./toolkit/...
go build ./runtime/go/... ./toolkit/...
```

Additional checks:

```bash
rg 'os/signal|signal\.Notify|syscall\.SIG(INT|TERM)|OnShutdown' runtime/go/mvep/server
```

The final search should find no process-signal handling or `OnShutdown` in the
server implementation, tests, or server docs.

## Rollout

1. Tracking issue [#16](https://github.com/mainvec/mvep/issues/16) confirms the
   `runtime/go/v0.8.0` target.
2. Implement and merge the mvep runtime change with migration documentation.
3. Tag `runtime/go/v0.8.0` only after CI and the race detector pass.
4. Update Zirafa to v0.8 and replace its temporary owned HTTP lifecycle with
   `mvep/server`.
5. Audit and migrate Linkvec and Droy in separate PRs before their next release.
6. Watch consumers for shutdown timeouts, recursive cleanup callbacks, and
   assumptions that `Start()` catches signals.

## Decision Log

- **Caller owns signals.** Process signals are application policy and do not
  belong in an embeddable server library.
- **Caller owns application cleanup.** mvep drains its HTTP surface; the daemon
  sequences engines, databases, child processes, and persistence explicitly.
- **Remove `OnShutdown`.** It hides ordering and enables recursive shutdown.
- **Keep blocking `Start`.** Existing control flow remains familiar, but its
  unblock condition becomes explicit server shutdown rather than a signal.
- **Add `StartAsync`, not readiness polling.** Successful return is the readiness
  boundary and includes all configured listeners.
- **Keep `Shutdown()` temporarily.** It is a bounded compatibility wrapper;
  `ShutdownContext` is preferred for production daemons.
- **One exported default timeout (`DefaultShutdownTimeout = 30s`).** Both the
  compatibility wrapper and mvep-initiated shutdown need a budget when no caller
  context exists. A single exported constant keeps that budget documented,
  testable, and referenceable by consumers instead of hidden per call site. 30s
  is chosen to sit under the common 90s systemd `TimeoutStopSec` while allowing
  long-poll style handlers to finish.
- **`Wait`/`Done` never complete for a never-started server.** Completion is a
  terminal-transition signal, not a "not running" signal. Auto-closing `done` for
  an unstarted server would make `Wait() == nil` ambiguous between "never ran"
  and "ran and stopped cleanly".
- **`NewServer` copies the config.** Retaining the caller's pointer (today's
  behavior) lets post-construction mutation race the serve goroutines and makes
  defaulting observable as a side effect on the caller's struct.
- **Single-use server.** Restart requires constructing a new `Server`, avoiding
  mux re-registration and stale lifecycle state.
- **Server owns passed listeners after start begins.** This gives startup rollback
  and shutdown deterministic resource ownership.
- **One listener failure stops the server.** A multi-listener server represents
  one service lifecycle, not independent partial availability.
- **No wire change.** v0.8 is required because lifecycle behavior and
  `ServerConfig` source compatibility change, not because clients must upgrade.

## Progress

- [x] T1 — Add failing lifecycle, graceful-shutdown, error-propagation, and listener rollback tests
- [x] T2 — Implement lifecycle state, `StartAsync`, `Wait`, `Done`, `Err`, and `GetListeners`
- [x] T3 — Implement context-aware graceful and forced shutdown with tracked `http.Server` values
- [x] T4 — Remove signal handling and `OnShutdown`; preserve blocking `Start` and bounded `Shutdown`
- [x] T5 — Update server/runtime/toolkit docs, changelog, and migration guide
- [x] T6 — Run full tests, race detector, vet, build, and a caller-owned-signal smoke test
- [ ] T7 — Tag and verify `runtime/go/v0.8.0`

## Tasks

### T1 — Add failing lifecycle tests

- **Outcome**: Tests precisely encode the new API and lifecycle contract before
  implementation.
- **Verification**: New tests fail because `StartAsync`, `Wait`, `Done`, `Err`,
  `GetListeners`, and `ShutdownContext` do not exist or current shutdown remains
  blocked/non-graceful.
- **Notes**: Replace existing `time.Sleep` readiness patterns as tests are
  touched. Do not send SIGINT/SIGTERM to the test process.
- **Result**: New tests live in `runtime/go/mvep/server/lifecycle_test.go`; the
  existing `server_test.go` cases were converted to `StartAsync` + `t.Cleanup`
  and no longer poll `GetListener` or sleep. A `failingListener` whose `Accept`
  always errors injects the fatal serve failure deterministically.

### T2 — Implement start, wait, state, and listener discovery

- **Outcome**: Listener binding is synchronous and transactional;
  `StartAsync` establishes readiness; `Start` delegates to `StartAsync` +
  `Wait`; all listeners and terminal state are observable.
- **Verification**: Port-zero, multi-listener, repeated-start, pre-start
  shutdown, rollback, config-copy, and defensive-copy tests pass.
- **Notes**: Preserve `GetListener` and deprecated `ListenAddress` behavior.
  `NewServer` dereferences the config pointer into a value field and copies
  `Listeners`. Document in godoc that `Wait`/`Done` do not complete for a
  never-started server.
- **Result**: `StartAsync` holds `lifecycleMu` across the whole bind phase, so a
  concurrent shutdown always observes either a committed startup or a completed
  rollback and `stateStarting` is never externally visible.

### T3 — Implement graceful shutdown and serve-error propagation

- **Outcome**: The server tracks one `http.Server` per listener, drains requests,
  bounds shutdown by context, force-closes after timeout, and stops all listeners
  on a fatal serve error.
- **Verification**: Active-handler, timeout, concurrent-idempotence, async-stop,
  and injected-serve-error tests pass under `-race`.
- **Notes**: Ignore `http.ErrServerClosed` during normal shutdown and join only
  actionable independent errors. Add `DefaultShutdownTimeout = 30 * time.Second`
  and use it for `Shutdown()` and for the serve-error teardown path, which has no
  caller context.
- **Result**: The serve goroutine spawns the teardown in a new goroutine because
  `drain` waits on `serveWG`; doing it inline would deadlock. Each listener's
  `http.Server` sets `ReadHeaderTimeout` (10s) since constructing `http.Server`
  explicitly otherwise leaves the previous `http.Serve` Slowloris exposure in
  place with no default.

### T4 — Remove server-owned signals and callback cleanup

- **Outcome**: `mvep/server` has no process-signal knowledge and no
  `ServerConfig.OnShutdown`; blocking `Start` ends only through explicit
  lifecycle transitions.
- **Verification**: Source search finds no signal imports/constants or
  `OnShutdown`; an owner-level smoke program handles SIGTERM and invokes
  `ShutdownContext` itself.
- **Notes**: Do not add an opt-out flag. Signal handling is removed rather than
  disabled by configuration.

### T5 — Update documentation and migration guidance

- **Outcome**: All examples show caller-owned signals, explicit application
  cleanup ordering, asynchronous stop handlers, and the new lifecycle methods.
- **Verification**: Runtime README, SERVER.md, MVEP_SKILL.md, changelog, and the
  migration guide agree on API names and shutdown semantics; stale examples are
  absent under `rg`.
- **Notes**: Explicitly warn that synchronous shutdown from a served handler can
  self-wait and that handlers must observe request-context cancellation.
- **Result**: `MVEP_SKILL.md` server snippets were also corrected — they
  documented a `NewServer(ServerConfig{Addr, EnableHealth}, handler)` signature
  that never existed in this runtime.

### T6 — Full verification

- **Outcome**: Runtime and toolkit remain healthy and the lifecycle works in a
  real owner process.
- **Verification**: All commands in the Verification section pass; the smoke
  process receives SIGTERM in its owning `main`, drains, and exits zero.
- **Notes**: Any unrelated existing failure must be recorded rather than hidden
  by weakening lifecycle tests.
- **Result**: `go build`, `go vet`, `go test`, and `go test -race` pass for
  `./runtime/go/...` and `./toolkit/...`. The signal search finds no matches in
  the server implementation or tests; the remaining `SERVER.md` hits are the
  owner-side examples the plan requires. A throwaway module consuming the runtime
  via a local replace started the server, received SIGTERM in its own `main`,
  drained via `ShutdownContext`, and exited zero.

### T7 — Release Go runtime v0.8.0

- **Outcome**: Consumers can resolve the caller-owned lifecycle from the Go
  module proxy.
- **Verification**: `runtime/go/v0.8.0` points at the merged commit and a clean
  throwaway module can `go get github.com/mainvec/mvep/runtime/go@v0.8.0` and
  compile the owner example.
- **Notes**: Do not tag until the migration guide is merged. Downstream consumer
  updates remain separate PRs.
