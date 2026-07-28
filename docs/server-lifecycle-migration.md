# Migrating to caller-owned server lifecycle (runtime/go v0.7 → v0.8)

`github.com/mainvec/mvep/runtime/go/mvep/server` no longer handles process
signals. The owning application decides which signals matter, how long shutdown
may take, and how HTTP draining is sequenced with the rest of its cleanup.

This guide covers the source and behavior changes daemon owners must make.

## What changed

| v0.7 | v0.8 |
| --- | --- |
| `Server.Start()` blocked until the server's own `SIGINT`/`SIGTERM` handler fired | `Start()` blocks until explicit shutdown or a fatal serve error |
| `Server.Shutdown()` closed listeners without draining or unblocking `Start()` | `Shutdown()` drains active requests, bounded by `DefaultShutdownTimeout`, and unblocks `Start()` |
| `ServerConfig.OnShutdown func()` | **Removed.** Sequence cleanup explicitly around `ShutdownContext` |
| `go srv.Start()` + polling `GetListener()` for readiness | `srv.StartAsync()` returns once every listener is serving |
| Serve errors were discarded | Recorded and exposed via `Wait()` / `Err()`; one failed listener stops the rest |
| Only the primary listener was reachable via `GetListener()` | `GetListeners()` returns every bound listener, in configuration order |
| `NewServer` retained and mutated the caller's `*ServerConfig` | The configuration is copied |

There is **no wire or client change**. A v0.8 server and a v0.7 client remain
compatible.

## New API

```go
const DefaultShutdownTimeout = 30 * time.Second

var (
    ErrServerStarted = errors.New("mvep/server: server already started")
    ErrServerStopped = errors.New("mvep/server: server already stopped")
)

func (s *Server) StartAsync() error
func (s *Server) Wait() error
func (s *Server) Done() <-chan struct{}
func (s *Server) Err() error
func (s *Server) ShutdownContext(ctx context.Context) error
func (s *Server) GetListeners() []net.Listener
```

A `Server` is single-use. `Start`/`StartAsync` return `ErrServerStarted` when the
server is already running and `ErrServerStopped` once it has stopped. Construct
a new `Server` to restart.

## Migration steps

### 1. Remove `ServerConfig.OnShutdown`

```diff
 config := &server.ServerConfig{
     Listeners:         []server.ListenerConfig{{Address: addr}},
     EnableHealthCheck: true,
-    OnShutdown:        func() { engine.Close() },
 }
```

The cleanup moves into the owner's shutdown sequence (step 3).

### 2. Replace goroutine-plus-polling startup

```diff
-go srv.Start()
-for srv.GetListener() == nil {
-    time.Sleep(10 * time.Millisecond)
-}
+if err := srv.StartAsync(); err != nil {
+    return err
+}
 log.Printf("listening on %s", srv.GetListener().Addr())
```

`StartAsync` binds every configured listener synchronously and returns only when
all of them are serving, so readiness polling is no longer needed. Binding is
transactional: if a later listener fails to bind, every listener already acquired
during that attempt is closed.

### 3. Own signals and cleanup ordering

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

Choose a different order where your domain requires it — mvep does not make that
policy decision.

### 4. Make stop endpoints asynchronous

Calling `ShutdownContext` synchronously from a handler served by the same server
waits for that handler and blocks until the context expires. Write the response
first, then trigger shutdown from a new goroutine:

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

### 5. Report failures through `Wait`, `Done`, or `Err`

Any `Serve` error other than `http.ErrServerClosed` is recorded as the lifecycle
error and shuts down the remaining listeners. A multi-listener server is one
service lifecycle, not independent partial availability.

```go
if err := srv.Wait(); err != nil {
    log.Printf("server stopped with error: %v", err)
}
```

### 6. Set configuration before `NewServer`

`NewServer` copies the configuration and no longer writes the defaulted
`HealthCheckPath` back into the caller's struct. Set every field, including the
`Listeners` slice, before calling `NewServer`.

## Behavior notes

- **Shutdown budget.** `Shutdown()` and any shutdown mvep initiates itself (for
  example after a fatal serve error) use `DefaultShutdownTimeout` (30s). Use
  `ShutdownContext` when the owner should choose the budget.
- **Forced close.** If the graceful budget expires, mvep calls
  `http.Server.Close()` so lifecycle completion stays bounded.
  `ShutdownContext` then returns an error wrapping `context.DeadlineExceeded`.
- **Uncancellable handlers.** Handlers that ignore their request context cannot
  be terminated by Go and keep running after `ShutdownContext` returns.
- **Listener ownership.** Once start begins, the server owns both auto-created
  and pre-created listeners and closes them on shutdown or rollback. Closing a
  server-owned listener externally is treated as a fatal serve failure.
- **`Wait` on a never-started server.** `Wait`/`Done` are released only by a
  terminal transition. A server that is constructed and never started blocks in
  `Wait` until `Shutdown` is called.
- **Slowloris mitigation.** Each listener's `http.Server` now sets a 10s
  `ReadHeaderTimeout`. Clients that trickle request headers are disconnected.

## Verification checklist

- No reference to `ServerConfig.OnShutdown` remains.
- Signal handling lives in the process owner, not in a library.
- Every shutdown path passes a bounded context.
- Stop endpoints respond before triggering shutdown.
- Readiness comes from `StartAsync` returning, not from sleeps or polling.
