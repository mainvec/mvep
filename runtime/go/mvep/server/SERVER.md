# MVEP Server

The MVEP Server is a reusable server component that eliminates boilerplate code when creating HTTP servers for MVEP packages. It handles common tasks like:

- HTTP server setup and lifecycle management
- CORS configuration
- Health check endpoints
- Signal handling for graceful shutdown
- Package registration and routing

## Features

- **Zero Boilerplate**: Create a production-ready server with just a few lines of code
- **Multi-Package Support**: Register multiple MVEP packages on the same server
- **Flexible Configuration**: Support for TCP and Unix socket listeners
- **Built-in Health Checks**: Optional health check endpoint
- **CORS Support**: Enable CORS headers with a single flag
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals properly

## Basic Usage

```go
package main

import (
    "log"
    "github.com/mainvec/mvep/runtime/go/mvep"
    "github.com/mainvec/mvep/runtime/go/mvep/server"
    "github.com/yourorg/yourproject/api"
    "github.com/yourorg/yourproject/impl"
)

func main() {
    // Create server configuration
    config := &server.ServerConfig{
        ListenAddress:     "127.0.0.1:8080",
        BasePath:          "",
        EnableHealthCheck: true,
        EnableCORS:        true,
    }

    // Create the server
    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }

    // Register your MVEP package
    pkg := api.NewPackage()
    runner := impl.GetCommandRunner()

    err = srv.RegisterPackage(pkg, runner)
    if err != nil {
        log.Fatalf("Failed to register package: %v", err)
    }

    // Start the server (blocks until shutdown)
    log.Println("Starting server...")
    err = srv.Start()
    if err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

## Configuration Options

### ServerConfig

```go
type ServerConfig struct {
    // ListenAddress is the address to listen on
    // Examples: "127.0.0.1:8080", "tcp://0.0.0.0:8080", "unix:///tmp/server.sock"
    ListenAddress string

    // BasePath is the base URL path for all endpoints (e.g., "/api")
    BasePath string

    // EnableHealthCheck adds a health check endpoint if true
    EnableHealthCheck bool

    // HealthCheckPath is the path for the health check (default: "/health")
    HealthCheckPath string

    // EnableCORS adds CORS headers to all responses if true
    EnableCORS bool

    // OnShutdown is called when the server receives a shutdown signal
    OnShutdown func()
}
```

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

srv.Start()
```

Each package will be available at:
- `<BasePath>/<PackageName>/cmd`

## Listen Address Formats

The daemon supports multiple address formats:

```go
// TCP with IP and port
ListenAddress: "127.0.0.1:8080"
ListenAddress: "tcp://0.0.0.0:8080"

// Unix socket
ListenAddress: "unix:///tmp/server.sock"
```

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

    // Wait for signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
}

// Plus CORS middleware functions, address parsing, etc.
```

### After (With MVEP Server)

```go
// ~20 lines of code
func main() {
    config := &server.ServerConfig{
        ListenAddress:     "127.0.0.1:8080",
        EnableHealthCheck: true,
        EnableCORS:        true,
    }

    srv, _ := server.NewServer(config)

    pkg := api.NewPackage()
    runner := impl.GetCommandRunner()
    srv.RegisterPackage(pkg, runner)

    srv.Start()
}
```

## Real-World Example: Dashboard Server

See [droy-dashboard/backend/backend.go](../../droy/droy-dashboard/backend/backend.go) for a complete example:

```go
package backend

import (
    dashboard "github.com/mainvec/droy/droy-dashboard/mvepapi/go"
    dashapi "github.com/mainvec/droy/droy-dashboard/mvepapi/go/api"
    "github.com/mainvec/mvep/runtime/go/mvep/server"
)

type DashServer struct {
    srv *server.Server
}

func NewDashServer(config *DashServerConfig) (*DashServer, error) {
    serverConfig := &server.ServerConfig{
        ListenAddress:     config.ListenAddress,
        BasePath:          config.BasePath,
        EnableHealthCheck: true,
        EnableCORS:        true,
        OnShutdown:        config.OnShutdown,
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
    return d.srv.Start()
}
```

## License

Part of the MVEP Go framework.
