# runtime/go
mainvec platform for golang
A Go library for building MVEP (MainVec Package) servers and clients with HTTP/Unix socket transport.

## Features

- **Server**: Create MVEP package servers with HTTP or Unix socket transports
- **Client Library**: Connect to MVEP servers with automatic command encoding/decoding
- **Multiple Transports**: Support for HTTP, HTTPS, and Unix sockets
- **Flexible Encoding**: Built-in support for JSON, Protocol Buffers, and custom encoders
- **Request/Response Headers**: Pass authentication tokens, trace IDs, and custom metadata
- **CORS Support**: Optional CORS headers for web applications
- **Health Checks**: Built-in health check endpoints

## Installation

```bash
go get github.com/mainvec/mvep/runtime/go/mvep
```

## Server Usage

### Basic Server Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/mainvec/mvep/runtime/go/mvep/server"
)

// Implement your Package interface
type MyPackage struct{}

func (p *MyPackage) GetName() string {
    return "mypackage"
}

func (p *MyPackage) InstanceOf(compName string) (any, bool) {
    // Return instance based on command name
    switch compName {
    case "MyCommand":
        return &MyCommand{}, true
    case "MyCommandResult":
        return &MyCommandResult{}, true
    }
    return nil, false
}

func (p *MyPackage) NameOf(comp any) string {
    // Return command name based on type
    switch comp.(type) {
    case *MyCommand:
        return "MyCommand"
    }
    return ""
}

// Implement your CommandRunner interface
type MyCommandRunner struct{}

func (r *MyCommandRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
    switch c := cmd.(type) {
    case *MyCommand:
        // Handle the command
        return &MyCommandResult{Data: "result"}, nil
    }
    return nil, fmt.Errorf("unknown command")
}

func main() {
    // Create server configuration
    config := &server.ServerConfig{
        ListenAddress:     "127.0.0.1:8080",
        BasePath:          "/api",
        EnableHealthCheck: true,
        EnableCORS:        true,
    }
    
    // Create server
    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Register your package
    pkg := &MyPackage{}
    runner := &MyCommandRunner{}
    err = srv.RegisterPackage(pkg, runner)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the server (blocks until shutdown signal)
    log.Println("Starting server...")
    if err := srv.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Unix Socket Server

```go
config := &server.ServerConfig{
    ListenAddress:     "unix:///tmp/myapp.sock",
    BasePath:          "/api",
    EnableHealthCheck: true,
}

srv, err := server.NewServer(config)
// ... rest of setup
```

### Configuration Options

**ServerConfig** fields:
- `ListenAddress` - Address to listen on (e.g., "127.0.0.1:8080", "unix:///tmp/socket")
- `BasePath` - Base URL path for endpoints (e.g., "/api")
- `EnableHealthCheck` - Adds a `/health` endpoint if true
- `HealthCheckPath` - Custom path for health check (default: "/health")
- `EnableCORS` - Adds CORS headers to all responses if true
- `OnShutdown` - Callback function called when server receives shutdown signal

## Client Usage

### Basic Client Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/mainvec/mvep/runtime/go/mvep/client"
)

func main() {
    // Create client configuration
    config := &client.ClientConfig{
        BaseURL:  "http://127.0.0.1:8080",
        BasePath: "/api",
        Encoder:  "application/json", // optional, defaults to JSON
    }
    
    // Create client
    c, err := client.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Register your package
    pkg := &MyPackage{}
    pkgClient, err := c.RegisterPackage(pkg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Send commands
    ctx := context.Background()
    cmd := &MyCommand{Input: "test"}
    
    result, err := pkgClient.SendCmd(ctx, cmd)
    if err != nil {
        log.Fatal(err)
    }
    
    // Type assert the result
    cmdResult := result.(*MyCommandResult)
    log.Printf("Result: %s", cmdResult.Data)
}
```

### Unix Socket Client

```go
config := &client.ClientConfig{
    BaseURL:  "unix:///tmp/myapp.sock",
    BasePath: "/api",
}

c, err := client.NewClient(config)
// ... rest of setup
```

### Advanced Client Usage

#### Auto-detect Package from Command

```go
// Client will automatically find which package handles the command
result, err := c.SendCmd(ctx, &MyCommand{Input: "test"})
```

#### Send Raw Command Data

```go
// Send raw bytes without automatic encoding
respBytes, err := pkgClient.SendRawCmd(ctx, "MyCommand", []byte(`{"input":"test"}`))
```

#### Custom Encoder

```go
// Use a different encoder for this specific command
result, err := pkgClient.SendCmdWithEncoder(ctx, cmd, "application/x-protobuf")
```

#### Health Check

```go
status, err := c.Ping(ctx)
if err != nil {
    log.Fatal("Server not responding:", err)
}
log.Println("Server status:", status)
```

#### Sending Commands with Headers

```go
// Send command with authentication and custom headers
headers := map[string]string{
    "auth":       "Bearer token123",
    "request-id": "abc-456",
}

result, resp, err := pkgClient.SendCmdReq(ctx, &MyCommand{Input: "test"}, headers)
if err != nil {
    log.Fatal(err)
}

// Access response headers
newToken := resp.GetHeader("new-token")
rateLimit := resp.GetHeader("rate-limit")
```

#### Accessing Headers in Command Handlers (Server)

```go
func (r *MyCommandRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
    switch c := cmd.(type) {
    case *MyCommand:
        // Get request headers from context
        authToken := mvep.GetRequestHeader(ctx, "auth")
        requestID := mvep.GetRequestHeader(ctx, "request-id")
        
        // Validate auth token...
        
        // Set response headers
        ctx = mvep.SetResponseHeader(ctx, "new-token", "refreshed-token")
        ctx = mvep.SetResponseHeader(ctx, "rate-limit", "99")
        
        return &MyCommandResult{Data: "result"}, nil
    }
    return nil, fmt.Errorf("unknown command")
}
```

### Configuration Options

**ClientConfig** fields:
- `BaseURL` - Base URL of the MVEP server (e.g., "http://127.0.0.1:8080" or "unix:///tmp/socket")
- `BasePath` - Base URL path for endpoints (e.g., "/api")
- `Encoder` - Content type for encoding (default: "application/json")
- `Timeout` - HTTP client timeout (default: 30 seconds)
- `HTTPClient` - Custom HTTP client (optional, ignored for Unix sockets)

### PackageClient Methods

- `SendCmd(ctx, cmd)` - Send command with default encoder
- `SendCmdReq(ctx, cmd, headers)` - Send command with custom headers
- `SendCmdWithEncoder(ctx, cmd, encoder)` - Send with specific encoder
- `SendCmdReqWithEncoder(ctx, cmd, headers, encoder)` - Send with headers and specific encoder
- `SendRawCmd(ctx, cmdName, data)` - Send raw bytes
- `SetEncoder(encoder)` / `GetEncoder()` - Manage default encoder
- `Package()` - Get underlying MVEP package

## Transport Support

The library supports multiple transport mechanisms:

- **HTTP/HTTPS**: Standard HTTP protocol
- **Unix Sockets**: Local inter-process communication

The transport layer is abstracted, so your command handlers work the same regardless of transport.

## Encoding Support

Built-in support for:
- JSON (`application/json`)
- Protocol Buffers (`application/x-protobuf`)
- Custom encoders via the `oencoding` package

## Using with curl

You can test MVEP endpoints directly with curl:

```bash
# Simple command
curl -X POST http://127.0.0.1:8080/api/mypackage/cmd \
  -H "Content-Type: application/json" \
  -H "x-mainvec-cmd: MyCommand" \
  -d '{"input": "test"}'

# Command with custom headers (auth, request ID, etc.)
curl -X POST http://127.0.0.1:8080/api/mypackage/cmd \
  -H "Content-Type: application/json" \
  -H "x-mainvec-cmd: MyCommand" \
  -H "x-mvep-auth: Bearer token123" \
  -H "x-mvep-request-id: abc-456" \
  -d '{"input": "test"}'

# View response headers
curl -i -X POST http://127.0.0.1:8080/api/mypackage/cmd \
  -H "Content-Type: application/json" \
  -H "x-mainvec-cmd: MyCommand" \
  -d '{"input": "test"}'
```

Custom headers use the `x-mvep-` prefix. For example, `x-mvep-auth` becomes `auth` in the `headers` map.

## Project Structure

```
runtime/go/
├── mvp/
│   ├── envelope.go        # CmdReq/CmdResp types for headers support
│   ├── http_transport.go  # HTTP transport layer
│   ├── mvpackge.go        # Package and handler interfaces
│   ├── client/
│   │   └── client.go      # Client implementation
│   └── server/
│       ├── server.go      # Server implementation
│       └── SERVER.md      # Server documentation
├── util/
│   ├── protobuf/          # Protocol Buffer encoder
│   └── protojson/         # JSON encoder
└── test/
    └── api/               # Test API definitions
```

## Examples

See the `test/` directory for example API definitions and usage patterns.

## License

See LICENSE file for details.