# MVEP Server

The MVEP Server is a reusable server component that eliminates boilerplate code when creating HTTP servers for MVEP packages. It handles common tasks like:

- HTTP server setup and listener binding
- CORS configuration
- Health check endpoints
- Context-aware graceful shutdown
- Package registration and routing

The server owns the **HTTP surface only**. It installs no process signal handlers:
the owning application decides which signals matter, how long shutdown may take,
and how HTTP draining is sequenced with the rest of its cleanup.

## Features

- **Zero Boilerplate**: Create a production-ready server with just a few lines of code
- **Multi-Package Support**: Register multiple MVEP packages on the same server
- **Flexible Configuration**: Support for TCP and Unix socket listeners
- **Built-in Health Checks**: Optional health check endpoint
- **CORS Support**: Enable CORS headers with a single flag
- **Caller-Owned Lifecycle**: `StartAsync`/`Wait`/`Done`/`Err` and context-aware shutdown

## Basic Usage

```go
package main

import (
    "context"
    "errors"
    "log"
    "os/signal"
    "syscall"
    "time"

    "github.com/mainvec/mvep/runtime/go/mvep/server"
    "github.com/yourorg/yourproject/api"
    "github.com/yourorg/yourproject/impl"
)

func main() {
    // Create server configuration
    config := &server.ServerConfig{
        Listeners:         []server.ListenerConfig{{Address: "127.0.0.1:8080"}},
        BasePath:          "",
        EnableHealthCheck: true,
        EnableCORS:        true,
        AllowedOrigins:    []string{"https://app.example.com"},
    }

    // Create the server
    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Register your MVEP package
    if err := srv.RegisterPackage(api.NewPackage(), impl.GetCommandRunner()); err != nil {
        log.Fatalf("Failed to register package: %v", err)
    }

    // Bind listeners and start serving. StartAsync returns only when every
    // listener is ready, so no readiness polling is needed.
    if err := srv.StartAsync(); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
    log.Printf("Listening on %s", srv.GetListener().Addr())

    // The application owns signal handling.
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

        if err := srv.ShutdownContext(shutdownCtx); err != nil {
            log.Fatalf("Shutdown error: %v", err)
        }

    case <-srv.Done():
        // A fatal serve or startup error stopped the HTTP surface.
        if err := srv.Err(); err != nil && !errors.Is(err, context.Canceled) {
            log.Fatalf("Server error: %v", err)
        }
    }
}
```

`Start()` remains available and still blocks, but it now returns when the owner
calls `Shutdown`/`ShutdownContext` or when a fatal serve error tears the server
down — not when the process receives a signal.

## Lifecycle

A `Server` is **single-use**. Once stopped it cannot be restarted; construct a
new `Server` instead.

| Method | Behavior |
| --- | --- |
| `Start() error` | `StartAsync` + `Wait`. Blocks until the server has completely stopped. |
| `StartAsync() error` | Binds every listener synchronously; returns when all are serving. |
| `Wait() error` | Blocks until a terminal transition, then returns the final lifecycle error. |
| `Done() <-chan struct{}` | Closed exactly once, after the final error is recorded. |
| `Err() error` | The final lifecycle error; `nil` while starting or running. |
| `ShutdownContext(ctx) error` | Drains until `ctx` expires, then force-closes. Idempotent and concurrency-safe. |
| `Shutdown() error` | Compatibility wrapper bounded by `server.DefaultShutdownTimeout` (30s). |
| `GetListener() net.Listener` | The primary listener, or `nil` before binding. |
| `GetListeners() []net.Listener` | A copy of every bound listener, in configuration order. |

Calling `Start`/`StartAsync` twice returns `server.ErrServerStarted`; calling
either after shutdown returns `server.ErrServerStopped`.

### Listener ownership

Once `Start`/`StartAsync` begins, the server owns both auto-created and
pre-created (`ListenerConfig.Listener`) listeners and closes them on shutdown.
Binding is transactional: if a later listener fails to bind, every listener
already acquired in that attempt is closed and the server transitions straight
to its stopped state with the error available from `Err()`.

### Fatal serve failures

A multi-listener server represents one service lifecycle. Any `Serve` error
other than `http.ErrServerClosed` is recorded as the lifecycle error and shuts
down the remaining listeners, bounded by `DefaultShutdownTimeout`.

### Waiting on a server that never started

`Wait` and `Done` are released only by a terminal transition (startup failure,
fatal serve error, or shutdown). A server that is constructed and never started
has no terminal transition, so `Wait` blocks until `Shutdown` is called.

### Stop endpoints must be asynchronous

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

Handlers that ignore their request context cannot be terminated by Go. Such a
handler keeps running after `ShutdownContext` returns.

## Configuration Options

### ServerConfig

`NewServer` copies the configuration. Mutating the struct afterwards has no
effect on the server, and `NewServer` does not modify it.

```go
type ServerConfig struct {
    // Listeners defines the set of listeners the server will serve on.
    // Each entry sets Address (auto-created) or Listener (pre-created),
    // with optional per-listener Middleware.
    Listeners []ListenerConfig

    // Deprecated: use Listeners instead.
    // Examples: "127.0.0.1:8080", "tcp://0.0.0.0:8080", "unix:///tmp/server.sock"
    ListenAddress string

    // BasePath is the base URL path for all endpoints (e.g., "/api")
    BasePath string

    // EnableHealthCheck adds a health check endpoint if true
    EnableHealthCheck bool

    // HealthCheckPath is the path for the health check (default: "/health")
    HealthCheckPath string

    // EnableCORS adds CORS headers to responses for allowed origins.
    // With no AllowedOrigins it emits nothing (fail closed) and warns at startup.
    EnableCORS bool

    // AllowedOrigins lists the origins CORS will respond to. An allowed origin
    // is echoed back, never "*". Preflight (OPTIONS) is handled automatically.
    AllowedOrigins []string

    // MaxRequestBytes bounds command request bodies (default: 4 MiB).
    // An oversized body returns 413 with code payload_too_large.
    MaxRequestBytes int64

    // VerboseErrors reflects raw handler error text to callers. Default false:
    // the response carries a stable code and a generic message, and the full
    // error is logged server-side with the request id.
    VerboseErrors bool

    // Interceptor is the global interceptor chain applied to all commands
    Interceptor mvep.CmdInterceptor
}
```

> **Removed in v0.8.0:** `OnShutdown`. Cleanup callbacks hide application
> ordering and can recurse into `Shutdown`. Express cleanup explicitly around
> `ShutdownContext` instead.

## Multiple Packages

You can register multiple MVEP packages on the same server:

```go
srv, _ := server.NewServer(config)

// Register package 1
pkg1 := api1.NewPackage()
runner1 := impl1.GetCommandRunner()
srv.RegisterPackage(pkg1, runner1)

// Register package 2
pkg2 := api2.NewPackage()
runner2 := impl2.GetCommandRunner()
srv.RegisterPackage(pkg2, runner2)

srv.StartAsync()
```

Each package will be available at:
- `<BasePath>/<PackageName>/cmd`

## Listen Address Formats

The daemon supports multiple address formats:

```go
// TCP with IP and port
{Address: "127.0.0.1:8080"}
{Address: "tcp://0.0.0.0:8080"}

// Unix socket
{Address: "unix:///tmp/server.sock"}
```

## TLS and Per-Listener Authentication

The server does **not** load certificates, generate keys, set socket
permissions, or authenticate requests. It provides three composition points and
the application supplies the policy:

| Layer | Where | Scope |
| --- | --- | --- |
| Transport (TLS, Unix socket) | `ListenerConfig.Listener` | one listener |
| HTTP request | `ListenerConfig.Middleware` | one listener, all paths |
| MVEP command | `ServerConfig.Interceptor` | all listeners, `/<pkg>/cmd` only |

### TLS

Build the `tls.Listener` yourself and pass it in. `Address` is ignored when
`Listener` is set, and the server closes the listener on shutdown.

```go
cert, err := tls.LoadX509KeyPair(certFile, keyFile)
if err != nil {
    return nil, fmt.Errorf("load key pair: %w", err)
}

baseLn, err := net.Listen("tcp", "127.0.0.1:8443")
if err != nil {
    return nil, fmt.Errorf("listen: %w", err)
}

tlsLn := tls.NewListener(baseLn, &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
})
```

For mutual TLS, set `ClientCAs` and `ClientAuth: tls.RequireAndVerifyClientCert`
on the same `tls.Config`; the handshake then rejects unauthenticated clients
before any handler runs.

### Mixing an authenticated TLS listener with a trusted Unix socket

A common daemon layout is a token-authenticated TLS port for remote clients plus
a Unix socket whose authorization is filesystem permissions. `mvep.LocalTrustMiddleware`
marks the request context as locally trusted, and `mvep.AuthInterceptor` skips
token validation for those requests.

```go
// Unix socket: permissions are the authorization boundary.
sockLn, err := net.Listen("unix", socketPath)
if err != nil {
    return nil, fmt.Errorf("listen unix: %w", err)
}
if err := os.Chmod(socketPath, 0o660); err != nil {
    _ = sockLn.Close()
    return nil, fmt.Errorf("chmod socket: %w", err)
}

config := &server.ServerConfig{
    Listeners: []server.ListenerConfig{
        {
            Listener:   sockLn,
            Middleware: mvep.LocalTrustMiddleware,
        },
        {
            Listener: tlsLn, // no local trust: token is required
        },
    },
    EnableHealthCheck: true,
    Interceptor: mvep.Chain(
        mvep.RecoveryInterceptor(),
        mvep.AuthInterceptor(tokenValidator),
    ),
}
```

The server owns both listeners once `Start`/`StartAsync` begins. Removing a
stale socket file before `net.Listen` is the application's responsibility.

### Authenticating non-command endpoints

`ServerConfig.Interceptor` only runs for MVEP command requests. Handlers
registered with `srv.Handle` bypass it entirely and must be wrapped explicitly:

```go
func requireToken(token string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions || mvep.IsLocalTrusted(r.Context()) {
            next.ServeHTTP(w, r)
            return
        }
        if subtle.ConstantTimeCompare([]byte(bearerToken(r)), []byte(token)) != 1 {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

srv.Handle("/events", requireToken(apiToken, sseHandler()))
```

Compare tokens with `crypto/subtle.ConstantTimeCompare` rather than `==` to
avoid leaking length and prefix information through timing.

> **`LocalTrustMiddleware` now verifies the peer before marking a request
> trusted.** A request is trusted only when it arrives over a Unix socket or
> from a loopback TCP address; any other peer is passed through *untrusted* (and
> the rejection is logged), so a listener accidentally exposed to the network
> fails closed instead of silently bypassing `AuthInterceptor`. UID filtering of
> Unix peers is platform-specific (`SO_PEERCRED`); where unsupported, the
> middleware falls back to socket-type and loopback checks.

### Command endpoint transport semantics

The command endpoint (`<BasePath>/<pkg>/cmd`) enforces real HTTP semantics:

- **Method**: `POST` only. Any other method returns `405` with code
  `method_not_allowed` and never reaches the runner.
- **Content-Type**: parsed as a media type, so `application/json; charset=utf-8`
  resolves the JSON encoder. An unregistered type returns `415`.
- **Body limit**: bounded by `MaxRequestBytes`; overruns return `413`.
- **Status mapping**: outcomes map to meaningful statuses — `401` unauthorized,
  `403` forbidden, `404` unknown command, `415` unsupported media type,
  `400` decode error, `413` payload too large, `405` method not allowed,
  `500` command error. The stable machine-readable code is in the
  `x-mainvec-error-code` response header.
- **Error redaction**: by default the response body is a generic message;
  handler error detail (SQL fragments, DSNs, file paths) is logged server-side
  only. Set `VerboseErrors: true` for local development.

> On the client side, never combine a custom `RootCAs` pool with
> `InsecureSkipVerify: true`. The latter takes precedence and disables
> certificate verification entirely, defeating the pinned CA.

## Endpoints

For a package named "myPackage", the daemon automatically creates:

- **Command Endpoint**: `<BasePath>/myPackage/cmd`
  - POST requests with `x-mainvec-cmd` header
- **Health Check**: `<BasePath>/health` (if enabled)
  - GET requests return "OK" with 200 status

## Comparison: Before and After

### Before (Manual Setup)

```go
// ~180 lines of boilerplate code
func main() {
    options, err := parseListenAddr(addr)
    // ... error handling

    ln, err := net.Listen(options.Network(), options.String())
    // ... error handling

    mux := http.NewServeMux()

    // Setup CORS
    mux.HandleFunc("/health", corsFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }))

    // Setup package handler
    pkg := api.NewPackage()
    runner := impl.GetCommandRunner()
    handler := &mvep.PackageHandler{
        Package: pkg,
        CommandRunner: runner,
    }
    mux.Handle("/"+pkg.GetName()+"/cmd", corsHandler(handler))

    // Start server in goroutine
    go http.Serve(ln, mux)

    // Wait for signals, then close the listener without draining
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
}

// Plus CORS middleware functions, address parsing, etc.
```

### After (With MVEP Server)

```go
func main() {
    config := &server.ServerConfig{
        Listeners:         []server.ListenerConfig{{Address: "127.0.0.1:8080"}},
        EnableHealthCheck: true,
        EnableCORS:        true,
    }

    srv, _ := server.NewServer(config)
    srv.RegisterPackage(api.NewPackage(), impl.GetCommandRunner())
    srv.StartAsync()

    // The application still owns signals and shutdown ordering.
    signalCtx, stopSignals := signal.NotifyContext(
        context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stopSignals()
    <-signalCtx.Done()

    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    _ = srv.ShutdownContext(ctx)
}
```

## Real-World Example: Dashboard Server

```go
package backend

import (
    "context"

    dashboard "github.com/mainvec/droy/droy-dashboard/mvepapi/go"
    dashapi "github.com/mainvec/droy/droy-dashboard/mvepapi/go/api"
    "github.com/mainvec/mvep/runtime/go/mvep/server"
)

type DashServer struct {
    srv *server.Server
}

func NewDashServer(config *DashServerConfig) (*DashServer, error) {
    serverConfig := &server.ServerConfig{
        Listeners:         []server.ListenerConfig{{Address: config.ListenAddress}},
        BasePath:          config.BasePath,
        EnableHealthCheck: true,
        EnableCORS:        true,
    }

    srv, err := server.NewServer(serverConfig)
    if err != nil {
        return nil, err
    }

    dashboardPkg := dashapi.NewPackage()
    dashboardCommandRunner := dashboard.GetCommandRunner()

    srv.RegisterPackage(dashboardPkg, dashboardCommandRunner)

    return &DashServer{srv: srv}, nil
}

func (d *DashServer) Start() error {
    return d.srv.StartAsync()
}

func (d *DashServer) Shutdown(ctx context.Context) error {
    return d.srv.ShutdownContext(ctx)
}
```

## License

Part of the MVEP Go framework.
