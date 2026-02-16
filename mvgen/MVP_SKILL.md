# MVP_SKILL — Mainvec Platform (MVP) Integration Guide

> **Audience:** AI coding agents and human developers integrating MVP into their projects.
>
> **What is MVP?** The Mainvec Platform is a **spec-driven, command-based API framework** that standardizes how applications define their commands and APIs. You write a declarative JSON spec, and the toolchain generates type-safe server code, client code, CLI tools, and API definitions in Go and JavaScript/TypeScript.

---

## Table of Contents

- [Ecosystem Overview](#ecosystem-overview)
- [Architecture](#architecture)
- [The MVEP Spec Format](#the-mvep-spec-format)
- [mvgen CLI Reference](#mvgen-cli-reference)
- [Project Integration Guide](#project-integration-guide)
- [Generated Code Structure](#generated-code-structure)
- [Core Generated Patterns](#core-generated-patterns)
- [mvpgo Runtime](#mvpgo-runtime)
- [ugo Utilities](#ugo-utilities)
- [Best Practices](#best-practices)
- [Common Pitfalls](#common-pitfalls)
- [For AI Agents — Quick Reference](#for-ai-agents--quick-reference)

---

## Ecosystem Overview

MVP consists of three core components:

| Component | Module | Purpose |
|-----------|--------|---------|
| **mvgen** | `github.com/mainvec/mvep/mvgen` | Code generator — transforms MVEP specs into Go, JS/TS, and Protobuf code |
| **mvpgo** | `github.com/mainvec/mvp/mvpgo` | Runtime library — `mvp.Package` and `mvp.CommandRunner` interfaces, HTTP/Unix socket server & client, middleware/interceptor system |
| **ugo** | `github.com/mainvec/ugo` | Go utilities — CLI framework (`cli`), ordered maps (`omap`), encoding registry (`oencoding`), validation, collections |

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        MVEP Spec (JSON/JSONC)                    │
│                    (your-service-spec.json)                       │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                       ▼
              ┌────────────────┐
              │     mvgen      │  Code Generator
              │  (CLI tool)    │
              └───────┬────────┘
                      │ generates
        ┌─────────────┼─────────────────────────┐
        ▼             ▼                         ▼
   Go Code        JS/TS Code             Proto Definitions
   ├── api/           ├── api/               ├── api/*.proto
   │   ├── *.plain.go │   ├── *.js           └── api/*.pb.go
   │   └── *_package.go   ├── *.d.ts
   ├── *_impl.go      │   └── *_package.js
   ├── *_commands.go   │
   └── cmd/*/main.go   │
        │               │
        ▼               ▼
  ┌──────────────────────────────────────────┐
  │   Implements mvp.Package &               │
  │   mvp.CommandRunner interfaces           │
  │            (from mvpgo)                  │
  └──────────────────┬───────────────────────┘
                     │
                     ▼
  ┌──────────────────────────────────────────┐
  │   PackageHandler                         │
  │   ├── Interceptor chain (auth, logging)  │
  │   ├── ServeCmd / ServeCmdReq             │
  │   └── HTTP / Unix socket transport       │
  └──────────────────────────────────────────┘
                     │
              ┌──────┴──────┐
              ▼             ▼
           Server        Client
       (mvp/server)   (mvp/client)
```

**Foundation layer:** `ugo` provides the CLI framework (used by generated `cmd/` code), ordered maps (used by mvgen for deterministic output), and the encoding registry (used by mvpgo for JSON/Protobuf serialization).

---

## The MVEP Spec Format

MVEP (MainVec Endpoint) specs are JSON or JSONC files validated against a JSON Schema (`2020-12` draft).

**Schema URL:** `https://spec.mainvec.com/mvepspec/0.1/schema/2026-01-15`

### Spec Structure

```jsonc
{
  "$id": "acmeapp",
  "$schema": "https://spec.mainvec.com/mvepspec/0.1/schema/2026-01-15",
  "name": "acmeapp",
  "namespace": "acmeappns",
  "title": "Acme Application API",
  "desc": "API specification for the Acme application",
  "version": "v0.1",

  "gen_options": {
    "go_package": "github.com/acme/acmeapp/mvpapi/go;acmeapp",
    "go_api_package": "github.com/acme/acmeapp/mvpapi/go/api;api",
    "format": "plain",           // "plain" (Go structs + JSON) or "pb3" (protobuf)
    "edition": "2023",
    "go_default_api_level": "API_OPAQUE"
  },

  "commands": {
    // Commands define the API operations
    "UserRegisterCmd": {
      "title": "Register a new user",
      "alias": "register",
      "desc": "Creates a new user account with email and password",
      "fields": {
        "email":    { "fnum": 1, "type": "string", "title": "User email",    "tags": ["required"] },
        "password": { "fnum": 2, "type": "string", "title": "User password", "tags": ["required"] },
        "name":     { "fnum": 3, "type": "string", "title": "Display name" }
      },
      "resultFields": {
        "userID": { "fnum": 1, "type": "string", "title": "Created user ID" },
        "token":  { "fnum": 2, "type": "string", "title": "Auth token" }
      }
    },

    "UserGetProfileCmd": {
      "title": "Get user profile",
      "alias": "get_profile",
      "fields": {
        "userID": { "fnum": 1, "type": "string", "tags": ["required"] }
      },
      "resultFields": {
        "user": { "fnum": 1, "type": "recRef", "$ref": "#/recordsDefs/User", "title": "User record" }
      }
    },

    "UserUpdateSettingsCmd": {
      "title": "Update user settings",
      "alias": "update_settings",
      "fields": {
        "userID":   { "fnum": 1, "type": "string", "tags": ["required"] },
        "settings": { "fnum": 2, "type": "map", "valueType": "string", "title": "Key-value settings" }
      },
      "resultFields": {
        "updated": { "fnum": 1, "type": "boolean" }
      }
    },

    "OrderCreateCmd": {
      "title": "Create a new order",
      "alias": "create_order",
      "fields": {
        "customerID": { "fnum": 1, "type": "string", "tags": ["required"] },
        "items":      { "fnum": 2, "type": "recRef", "$ref": "#/recordsDefs/OrderItem", "repeated": true, "title": "Order items" },
        "notes":      { "fnum": 3, "type": "string" }
      },
      "resultFields": {
        "orderID":  { "fnum": 1, "type": "string" },
        "total":    { "fnum": 2, "type": "double" },
        "createdAt": { "fnum": 3, "type": "timestamp" }
      }
    }
  },

  "recordsDefs": {
    // Shared data structures referenced by commands
    "User": {
      "name": "User",
      "title": "User record",
      "fields": {
        "id":        { "fnum": 1, "type": "string" },
        "email":     { "fnum": 2, "type": "string" },
        "name":      { "fnum": 3, "type": "string" },
        "active":    { "fnum": 4, "type": "boolean" },
        "createdAt": { "fnum": 5, "type": "timestamp" },
        "metadata":  { "fnum": 6, "type": "map", "valueType": "string" }
      }
    },
    "OrderItem": {
      "name": "OrderItem",
      "title": "An item in an order",
      "fields": {
        "productID": { "fnum": 1, "type": "string" },
        "quantity":  { "fnum": 2, "type": "int32" },
        "price":     { "fnum": 3, "type": "double" }
      }
    }
  }
}
```

### Required Top-Level Fields

| Field | Description |
|-------|-------------|
| `$id` | Unique identifier for the service |
| `name` | Service name (used for file naming) |
| `namespace` | Service namespace (used for protobuf package) |

### Field Type Reference

| Type | Go Type | JS/TS Type | Description |
|------|---------|------------|-------------|
| `string` | `string` | `string` | UTF-8 string |
| `boolean` | `bool` | `boolean` | True/false |
| `int32` | `int32` | `number` | 32-bit signed integer |
| `int64` | `int64` | `number` | 64-bit signed integer |
| `uint32` | `uint32` | `number` | 32-bit unsigned integer |
| `sint32` | `int32` | `number` | ZigZag-encoded signed integer |
| `float` | `float32` | `number` | 32-bit floating point |
| `double` | `float64` | `number` | 64-bit floating point |
| `bytes` | `[]byte` | `Uint8Array` | Raw bytes |
| `timestamp` | `*timestamppb.Timestamp` / `time.Time` | `Date` | Point in time |
| `duration` | `*durationpb.Duration` | `number` | Time duration |
| `uuid` | `string` | `string` | UUID string |
| `recRef` | Pointer to struct | Object | Reference to a `recordsDefs` entry |
| `map` | `map[string]T` | `Object` | String-keyed map (use `valueType`) |
| `recDef` | Inline struct | Object | Inline record definition |
| `oneOf` | Interface | Union type | One-of discriminated union |

### Special Field Properties

| Property | Description |
|----------|-------------|
| `fnum` | **Required.** Unique field number (maps to protobuf field number). Never reuse. |
| `type` | **Required.** One of the types from the table above. |
| `repeated` | `true` for arrays/slices of the type. |
| `tags` | Array of tags, e.g. `["required"]`. |
| `alias` | CLI flag alias for the field. |
| `$ref` | For `recRef` type: `"#/recordsDefs/RecordName"`. |
| `valueType` | For `map` type: the value type (e.g. `"string"`, `"boolean"`, `"int32"`). |
| `title` | Short human-readable label. |
| `desc` | Longer description. |

### Gen Options

| Option | Values | Description |
|--------|--------|-------------|
| `format` | `plain`, `pb3` | `plain` = Go structs with JSON tags (no protobuf dependency). `pb3` = full protobuf. |
| `go_package` | Module path | Go module path for the generated package |
| `go_api_package` | Module path | Go module path for the `api/` sub-package |
| `edition` | `2023` | Protobuf edition (when using `pb3` format) |
| `go_default_api_level` | `API_OPAQUE`, etc. | Protobuf API level |

---

## mvgen CLI Reference

### Installation

```bash
go install github.com/mainvec/mvep/mvgen/mvpapi/cmd/mvgen@latest
```

### Commands

| Command | Description | Required Flags |
|---------|-------------|----------------|
| `generate` | Generate code from an MVEP spec | `--in`, `--lang` |
| `validate` | Validate a spec against the JSON Schema | `--in` |
| `init` | Initialize a new MVEP spec file | `--name`, `--ns` |

### Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--in` | `generate`, `validate` | Path to the MVEP spec file |
| `--lang` | `generate` | Target language(s): `go`, `js`, or comma-separated `go,js` |
| `--outdir` | `generate` | Output directory for generated code |
| `--format` | `generate` | Output format: `plain` (default) or `pb3` |
| `--name` | `init` | Service name for the new spec |
| `--ns` | `init` | Namespace for the new spec |

### Examples

```bash
# Generate Go code (plain structs mode)
mvgen generate --in ./spec/acmeapp-spec.json --lang go --outdir ./go --format=plain

# Generate JavaScript/TypeScript code
mvgen generate --in ./spec/acmeapp-spec.json --lang js --outdir ./js --format=plain

# Generate both Go and JS in one invocation
mvgen generate --in ./spec/acmeapp-spec.json --lang go,js --outdir ./out

# Generate Go code with protobuf
mvgen generate --in ./spec/acmeapp-spec.json --lang go --outdir ./go --format=pb3

# Validate a spec
mvgen validate --in ./spec/acmeapp-spec.json

# Initialize a new spec
mvgen init --name myservice --ns myservicens
```

---

## Project Integration Guide

### Step 1: Create the Project Structure

```
your-project/
└── mvpapi/
    ├── generate_api.sh          # Code generation script
    └── spec/
        └── acmeapp-spec.json    # Your MVEP specification
```

### Step 2: Write Your MVEP Spec

Create `mvpapi/spec/<name>-spec.json` following the [spec format](#the-mvep-spec-format) above. Start with:

```jsonc
{
  "$id": "acmeapp",
  "$schema": "https://spec.mainvec.com/mvepspec/0.1/schema/2026-01-15",
  "name": "acmeapp",
  "namespace": "acmeappns",
  "title": "Acme Application API",
  "version": "v0.1",
  "gen_options": {
    "go_package": "github.com/acme/acmeapp/mvpapi/go;acmeapp",
    "go_api_package": "github.com/acme/acmeapp/mvpapi/go/api;api",
    "format": "plain"
  },
  "commands": {},
  "recordsDefs": {}
}
```

### Step 3: Create the Generation Script

Create `mvpapi/generate_api.sh`:

```bash
#!/bin/bash

# Generate API code from MVEP spec
# Usage: cd mvpapi && bash generate_api.sh

SPEC="./spec/acmeapp-spec.json"

mvgen generate -in "$SPEC" -lang go -outdir ./go -format=plain \
&& \
mvgen generate -in "$SPEC" -lang js -outdir ./js -format=plain

if [ $? -eq 0 ]; then
  echo "✓ API generated successfully"
else
  echo "✗ API generation failed"
  exit 1
fi
```

```bash
chmod +x mvpapi/generate_api.sh
```

### Step 4: Generate the Code

```bash
cd mvpapi
bash generate_api.sh
```

### Step 5: Implement Your Commands

Open the generated `*_impl.go` file. Each command has a stub like:

```go
func runUserRegisterCmd(ctx context.Context, cmd *api.UserRegisterCmd) (*api.UserRegisterCmdResult, error) {
    return nil, errors.New("command not implemented")
}
```

Replace the stub with your business logic:

```go
func runUserRegisterCmd(ctx context.Context, cmd *api.UserRegisterCmd) (*api.UserRegisterCmdResult, error) {
    user, err := createUser(ctx, cmd.Email, cmd.Password, cmd.Name)
    if err != nil {
        return nil, fmt.Errorf("registration failed: %w", err)
    }
    token, err := issueToken(ctx, user.ID)
    if err != nil {
        return nil, fmt.Errorf("token generation failed: %w", err)
    }
    return &api.UserRegisterCmdResult{
        UserID: user.ID,
        Token:  token,
    }, nil
}
```

### Step 6: Wire Up the Server (with mvpgo)

```go
package main

import (
    "github.com/acme/acmeapp/mvpapi/go/api"
    acmeapp "github.com/acme/acmeapp/mvpapi/go"
    "github.com/mainvec/mvp/mvpgo/mvp"
    "github.com/mainvec/mvp/mvpgo/mvp/server"
)

func main() {
    pkg := api.NewPackage()
    runner := acmeapp.GetCommandRunner()

    handler := mvp.NewPackageHandler(pkg, nil, runner,
        mvp.Chain(
            mvp.RecoveryInterceptor(),
            mvp.LoggingInterceptor(),
            mvp.RequestIDInterceptor(nil),
        ),
    )

    srv := server.NewServer(server.ServerConfig{
        Addr:     ":8080",
        BasePath: "/api",
    }, handler)

    srv.Start()
}
```

---

## Generated Code Structure

After running `mvgen generate`, your `mvpapi/` directory will look like:

```
mvpapi/
├── generate_api.sh
├── spec/
│   └── acmeapp-spec.json           # Your MVEP specification (you write this)
├── go/
│   ├── go.mod                      # Generated Go module file
│   ├── acmeapp_impl.go             # ✏️  EDIT THIS — command implementations
│   ├── acmeapp_commands.go         # ⛔ Generated — GetCommandRunner() factory
│   ├── api/
│   │   ├── acmeapp.plain.go        # ⛔ Generated — Go structs (plain mode)
│   │   └── acmeapp_package.go      # ⛔ Generated — Package, CommandRunner, handlers
│   └── cmd/
│       └── acmeapp/
│           └── acmeapp_main_cmd.go # ⛔ Generated — CLI entry point
└── js/
    └── api/
        ├── acmeapp.js              # ⛔ Generated — JS classes with JSDoc
        ├── acmeapp.d.ts            # ⛔ Generated — TypeScript type definitions
        ├── acmeapp_package.js      # ⛔ Generated — Package utilities
        └── client/                 # ✏️  Hand-written — typed client (see JS/TS Client section)
            ├── acmeapp_client.ts
            ├── acmeapp_package.ts
            └── index.ts
```

**With `--format=pb3`** (protobuf mode), the `api/` directory contains instead:

```
api/
├── acmeapp.proto                   # ⛔ Generated — Protobuf definition
├── acmeapp.pb.go                   # ⛔ Generated — Compiled protobuf Go code
└── acmeapp_package.go              # ⛔ Generated — Package, CommandRunner, handlers
```

### File Edit Safety

| File | Safe to Edit? | Notes |
|------|---------------|-------|
| `*_impl.go` | **Yes** | This is where you implement your business logic |
| `*_commands.go` | No | Regenerated — wires handler functions to the runner |
| `api/*_package.go` | No | Regenerated — handler types, PkgCommandRunner, dispatch |
| `api/*.plain.go` / `api/*.pb.go` | No | Regenerated — data structures |
| `api/*.proto` | No | Regenerated — protobuf definitions |
| `cmd/*/main.go` | No | Regenerated — CLI tool |
| `js/api/*` | No | Regenerated — JavaScript/TypeScript code |

### Protecting Files from Overwrite

Add `// NOMVGEN` (or `// NOWOGEN`) as the **first line** of any generated file to prevent mvgen from overwriting it:

```go
// NOMVGEN
package acmeapp

// This file is now protected from regeneration.
// ... your customizations ...
```

---

## Core Generated Patterns

### Handler Types (in `*_package.go`)

For each command, mvgen generates a typed handler function signature:

```go
type UserRegisterCmdHandler func(context.Context, *UserRegisterCmd) (*UserRegisterCmdResult, error)
type UserGetProfileCmdHandler func(context.Context, *UserGetProfileCmd) (*UserGetProfileCmdResult, error)
type OrderCreateCmdHandler func(context.Context, *OrderCreateCmd) (*OrderCreateCmdResult, error)
```

### PkgCommandRunner (in `*_package.go`)

A struct holding one handler per command, implementing `mvp.CommandRunner`:

```go
type PkgCommandRunner struct {
    RunUserRegisterCmd      UserRegisterCmdHandler
    RunUserGetProfileCmd    UserGetProfileCmdHandler
    RunUserUpdateSettingsCmd UserUpdateSettingsCmdHandler
    RunOrderCreateCmd       OrderCreateCmdHandler
}

// Implements mvp.CommandRunner — dispatches by type
func (r *PkgCommandRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
    switch cmd := cmd.(type) {
    case *UserRegisterCmd:
        return r.RunUserRegisterCmd(ctx, cmd)
    case *UserGetProfileCmd:
        return r.RunUserGetProfileCmd(ctx, cmd)
    case *OrderCreateCmd:
        return r.RunOrderCreateCmd(ctx, cmd)
    // ...
    }
    return nil, fmt.Errorf("unknown command type: %T", cmd)
}
```

### Package Interface (in `*_package.go`)

Implements `mvp.Package` for component resolution:

```go
// InstanceOf creates a zero-value instance by command name
func (p *acmeappPackage) InstanceOf(compName string) (any, bool) {
    switch compName {
    case "UserRegisterCmd":
        return &UserRegisterCmd{}, true
    case "UserRegisterCmdResult":
        return &UserRegisterCmdResult{}, true
    // ...
    }
    return nil, false
}

// NameOf returns the command name from a typed instance
func (p *acmeappPackage) NameOf(comp any) string {
    switch comp.(type) {
    case *UserRegisterCmd:
        return "UserRegisterCmd"
    // ...
    }
    return ""
}
```

### GetCommandRunner Factory (in `*_commands.go`)

Wires the impl functions to the runner:

```go
func GetCommandRunner() *api.PkgCommandRunner {
    return &api.PkgCommandRunner{
        RunUserRegisterCmd:       runUserRegisterCmd,
        RunUserGetProfileCmd:     runUserGetProfileCmd,
        RunUserUpdateSettingsCmd: runUserUpdateSettingsCmd,
        RunOrderCreateCmd:        runOrderCreateCmd,
    }
}
```

### Implementation Stubs (in `*_impl.go`)

Each command gets a stub — this is where you write your logic:

```go
func runUserRegisterCmd(ctx context.Context, cmd *api.UserRegisterCmd) (*api.UserRegisterCmdResult, error) {
    return nil, errors.New("command not implemented")
}

func runOrderCreateCmd(ctx context.Context, cmd *api.OrderCreateCmd) (*api.OrderCreateCmdResult, error) {
    return nil, errors.New("command not implemented")
}
```

---

## mvpgo Runtime

`github.com/mainvec/mvp/mvpgo` provides the runtime infrastructure for serving and consuming MVP APIs.

### Core Interfaces

```go
// mvp.Package — component registry for a service
type Package interface {
    GetName() string
    InstanceOf(compName string) (any, bool)  // factory: name → zero-value instance
    NameOf(comp any) string                   // reverse: instance → name
}

// mvp.CommandRunner — command execution
type CommandRunner interface {
    RunCmd(ctx context.Context, cmd any) (any, error)
}
```

### Request/Response Envelope

Commands are wrapped in envelopes with headers for transport:

```go
type CmdReq struct {
    Cmd     any
    Headers map[string]string  // headers use "x-mvp-" prefix in HTTP
    Payload []byte
}

type CmdResp struct {
    Headers map[string]string
    Payload []byte
    Error   *ErrorInfo
}
```

### PackageHandler

The bridge between your package, command runner, and transport layer:

```go
handler := mvp.NewPackageHandler(pkg, transporter, runner, interceptor)

// Server-side: handle incoming commands
handler.ServeCmdReq(ctx, req) *CmdResp

// Client-side: send commands
handler.SendCmdReq(ctx, req) *CmdResp
```

### Interceptor / Middleware System

Interceptors wrap command execution for cross-cutting concerns:

```go
type CmdHandler func(ctx context.Context, req *CmdReq) *CmdResp
type CmdInterceptor func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp
```

**Built-in interceptors:**

| Interceptor | Purpose |
|-------------|---------|
| `LoggingInterceptor()` | Logs command timing and structured metadata |
| `AuthInterceptor(validator)` | Token validation via `TokenValidator` interface |
| `RecoveryInterceptor()` | Catches panics and returns error responses |
| `RequestIDInterceptor(generator)` | Adds unique request IDs |

**Composition helpers:**

```go
// Chain multiple interceptors
chain := mvp.Chain(
    mvp.RecoveryInterceptor(),
    mvp.LoggingInterceptor(),
    mvp.AuthInterceptor(myValidator),
)

// Skip auth for specific commands
auth := mvp.SkipCommands(
    mvp.AuthInterceptor(myValidator),
    "UserRegisterCmd", "UserLoginCmd",
)

// Apply only to specific commands
admin := mvp.OnlyCommands(
    adminCheckInterceptor,
    "AdminDeleteUserCmd", "AdminResetCmd",
)
```

### Server

```go
srv := server.NewServer(server.ServerConfig{
    Addr:         ":8080",
    BasePath:     "/api",
    EnableHealth: true,
    EnableCORS:   true,
    OnShutdown:   func() { /* cleanup */ },
}, handler)

srv.Start()
```

### Go Client

`github.com/mainvec/mvp/mvpgo/mvp/client` provides a full-featured Go client for calling MVP services.

#### `ClientConfig`

```go
type ClientConfig struct {
    BaseURL     string              // Required. e.g. "http://localhost:8080" or "unix:///tmp/my.sock"
    BasePath    string              // URL path prefix, e.g. "/api"
    Encoder     string              // Content type, default "application/json"
    Timeout     time.Duration       // HTTP timeout, default 30s
    HTTPClient  *http.Client        // Optional custom HTTP client
    Interceptor mvp.ClientInterceptor // Optional interceptor chain
}
```

#### Creating a Client

```go
import (
    "github.com/mainvec/mvp/mvpgo/mvp/client"
    "github.com/mainvec/mvp/mvpgo/mvp"
    "github.com/acme/acmeapp/mvpapi/go/api"
)

mvpClient, err := client.NewClient(client.ClientConfig{
    BaseURL:  "http://localhost:8080",
    BasePath: "/api",
})
if err != nil {
    log.Fatal(err)
}
defer mvpClient.Close()

// Register the generated package
pkg := api.NewPackage()
pkgClient, err := mvpClient.RegisterPackage(pkg)
if err != nil {
    log.Fatal(err)
}
```

#### Sending Commands

```go
// Simple — no headers
result, err := pkgClient.SendCmd(ctx, &api.UserRegisterCmd{
    Email:    "alice@example.com",
    Password: "s3cret",
    Name:     "Alice",
})
if err != nil {
    log.Fatal(err)
}
regResult := result.(*api.UserRegisterCmdResult)
fmt.Println("User ID:", regResult.UserID)
fmt.Println("Token:", regResult.Token)

// With headers — returns typed result + response envelope
result, resp, err := pkgClient.SendCmdReq(ctx, &api.UserGetProfileCmd{
    UserID: regResult.UserID,
}, map[string]string{
    "auth": regResult.Token,
})
if err != nil {
    log.Fatal(err)
}
profileResult := result.(*api.UserGetProfileCmdResult)
fmt.Println("Name:", profileResult.User.Name)
```

#### Client Interceptors

```go
type ClientInvoker func(ctx context.Context, req *CmdReq) (*CmdResp, error)
type ClientInterceptor func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error)
```

**Built-in client interceptors:**

| Interceptor | Description |
|-------------|-------------|
| `AuthHeaderInterceptor(tokenProvider)` | Dynamically injects `auth` header. `TokenProvider = func(ctx) (string, error)` |
| `StaticAuthHeaderInterceptor(token)` | Adds a fixed `auth` header to every request |
| `HeaderInterceptor(headers)` | Adds custom static headers |
| `ClientLoggingInterceptor()` | Logs request/response timing via `slog` |
| `RetryInterceptor(maxRetries, delay)` | Retries on transport errors |
| `ClientRequestIDInterceptor(generator)` | Adds `request-id` header |
| `SkipCommandsClient(interceptor, cmds...)` | Skips interceptor for listed commands |

#### Building a Typed Go Client (Full Example)

A common pattern is wrapping `PackageClient` in a typed struct with auth management:

```go
package acmeclient

import (
    "context"
    "fmt"
    "sync"

    "github.com/mainvec/mvp/mvpgo/mvp"
    "github.com/mainvec/mvp/mvpgo/mvp/client"
    "github.com/acme/acmeapp/mvpapi/go/api"
)

type AcmeClient struct {
    mvpClient *client.Client
    pkgClient *client.PackageClient
    mu        sync.RWMutex
    authToken string
}

type AcmeClientConfig struct {
    BaseURL  string
    BasePath string
    Timeout  time.Duration
}

func NewAcmeClient(cfg AcmeClientConfig) (*AcmeClient, error) {
    ac := &AcmeClient{}

    // Auth interceptor reads token from the client struct
    tokenProvider := func(ctx context.Context) (string, error) {
        ac.mu.RLock()
        defer ac.mu.RUnlock()
        return ac.authToken, nil
    }

    mvpClient, err := client.NewClient(client.ClientConfig{
        BaseURL:  cfg.BaseURL,
        BasePath: cfg.BasePath,
        Timeout:  cfg.Timeout,
        Interceptor: mvp.ChainClient(
            mvp.ClientLoggingInterceptor(),
            mvp.AuthHeaderInterceptor(tokenProvider),
            mvp.RetryInterceptor(3, time.Second),
        ),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    pkg := api.NewPackage()
    pkgClient, err := mvpClient.RegisterPackage(pkg)
    if err != nil {
        mvpClient.Close()
        return nil, fmt.Errorf("failed to register package: %w", err)
    }

    ac.mvpClient = mvpClient
    ac.pkgClient = pkgClient
    return ac, nil
}

func (c *AcmeClient) Close() error {
    return c.mvpClient.Close()
}

func (c *AcmeClient) SetAuthToken(token string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.authToken = token
}

// ── Typed command methods ──────────────────────────────────

func (c *AcmeClient) RegisterUser(ctx context.Context, email, password, name string) (*api.UserRegisterCmdResult, error) {
    result, _, err := c.pkgClient.SendCmdReq(ctx, &api.UserRegisterCmd{
        Email: email, Password: password, Name: name,
    }, nil)
    if err != nil {
        return nil, err
    }
    regResult := result.(*api.UserRegisterCmdResult)
    // Auto-set token on successful registration
    if regResult.Token != "" {
        c.SetAuthToken(regResult.Token)
    }
    return regResult, nil
}

func (c *AcmeClient) GetUserProfile(ctx context.Context, userID string) (*api.UserGetProfileCmdResult, error) {
    result, _, err := c.pkgClient.SendCmdReq(ctx, &api.UserGetProfileCmd{
        UserID: userID,
    }, nil)  // auth header injected automatically by interceptor
    if err != nil {
        return nil, err
    }
    return result.(*api.UserGetProfileCmdResult), nil
}

func (c *AcmeClient) CreateOrder(ctx context.Context, customerID string, items []*api.OrderItem, notes string) (*api.OrderCreateCmdResult, error) {
    result, _, err := c.pkgClient.SendCmdReq(ctx, &api.OrderCreateCmd{
        CustomerID: customerID, Items: items, Notes: notes,
    }, nil)
    if err != nil {
        return nil, err
    }
    return result.(*api.OrderCreateCmdResult), nil
}
```

**Using it:**

```go
func main() {
    client, err := acmeclient.NewAcmeClient(acmeclient.AcmeClientConfig{
        BaseURL:  "http://localhost:8080",
        BasePath: "/api",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Register — auto-sets auth token
    reg, err := client.RegisterUser(ctx, "alice@example.com", "s3cret", "Alice")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("User ID:", reg.UserID)

    // Subsequent calls include auth token automatically
    profile, err := client.GetUserProfile(ctx, reg.UserID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Name:", profile.User.Name)

    // Create an order
    order, err := client.CreateOrder(ctx, reg.UserID, []*api.OrderItem{
        {ProductID: "prod-1", Quantity: 2, Price: 29.99},
        {ProductID: "prod-2", Quantity: 1, Price: 49.99},
    }, "Rush delivery")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Order:", order.OrderID, "Total:", order.Total)
}
```

#### Unix Socket Client

For local IPC via Unix sockets, just change the `BaseURL`:

```go
mvpClient, err := client.NewClient(client.ClientConfig{
    BaseURL: "unix:///tmp/acmeapp.sock",
})
```

The client auto-detects the `unix://` scheme and configures the HTTP transport accordingly.

---

## ugo Utilities

`github.com/mainvec/ugo` provides foundational Go utilities used across the MVP ecosystem.

### `cli` — CLI Framework

Used by generated `cmd/` code. Provides structured command definitions with flags, subcommands, and aliases.

```go
type Command struct {
    Usage, Short, Long string
    Hidden  bool
    Aliases []string
}

type Framework struct {
    DefaultRunner Runner
    Initializers  []Initializer
    Root          *Command
}
```

### `omap` — Ordered Map

`OMap[K, V]` maintains insertion order and supports sorted iteration. Used by mvgen for deterministic code generation output.

```go
m := omap.New[string, int]()
m.Set("b", 2)
m.Set("a", 1)

// Iterate in key-sorted order
m.IteratorByKey(func(k string, v int) bool {
    fmt.Println(k, v)  // a 1, b 2
    return true
})
```

### `oencoding` — Encoding Registry

Pluggable encoding interface with global registry. Used by mvpgo for JSON and Protobuf serialization.

```go
type Encoding interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
    MimeType() string
}
```

---

## JavaScript / TypeScript Client Usage

The generated JS/TS code provides everything you need to build a type-safe client for your MVP API from browser or Node.js applications.

### What Gets Generated (JS side)

```
js/
└── api/
    ├── acmeapp.js              # JS classes with JSDoc — commands, records, constructors
    ├── acmeapp.d.ts            # TypeScript type definitions — interfaces & classes
    └── acmeapp_package.js      # Package utilities — instanceOf(), nameOf(), PACKAGE_NAME
```

### Generated JS Classes (`acmeapp.js`)

Each command and record becomes a JS class inside a namespace:

```javascript
// Code generated by mvgen. DO NOT EDIT.
export const acmeappns = (() => {
  const ns = {};

  // Record classes
  ns.User = class User {
    static _typeName = 'User';
    constructor(data = {}) {
      /** @type {string} */
      this.id = data.id ?? '';
      /** @type {string} */
      this.email = data.email ?? '';
      /** @type {string} */
      this.name = data.name ?? '';
      /** @type {boolean} */
      this.active = data.active ?? false;
      /** @type {Date} */
      this.createdAt = data.createdAt ?? null;
    }
    static verify(message) { /* validates fields */ }
    static fromObject(obj) { /* creates instance from plain object */ }
    static toObject(message) { /* converts to plain object */ }
    toJSON() { /* serializes to JSON string */ }
  };

  // Command classes
  ns.UserRegisterCmd = class UserRegisterCmd {
    static _typeName = 'UserRegisterCmd';
    constructor(data = {}) {
      /** @type {string} */
      this.email = data.email ?? '';
      /** @type {string} */
      this.password = data.password ?? '';
      /** @type {string} */
      this.name = data.name ?? '';
    }
    // ... verify, fromObject, toObject, toJSON
  };

  ns.UserRegisterCmdResult = class UserRegisterCmdResult {
    static _typeName = 'UserRegisterCmdResult';
    constructor(data = {}) {
      /** @type {string} */
      this.userID = data.userID ?? '';
      /** @type {string} */
      this.token = data.token ?? '';
    }
    // ...
  };

  return ns;
})();
```

### Generated TypeScript Definitions (`acmeapp.d.ts`)

```typescript
// Code generated by mvgen. DO NOT EDIT.
export declare namespace acmeappns {

  export interface IUser {
    id?: string;
    email?: string;
    name?: string;
    active?: boolean;
    createdAt?: Date | null;
  }

  export class User implements IUser {
    static readonly _typeName: string;
    constructor(properties?: IUser);
    public id?: string;
    public email?: string;
    public name?: string;
    public active?: boolean;
    public createdAt?: Date | null;
    public static verify(message: { [k: string]: any }): string | null;
    public static fromObject(object: { [k: string]: any }): User;
    public static toObject(message: User): { [k: string]: any };
    public toJSON(): string;
  }

  export interface IUserRegisterCmd {
    email?: string;
    password?: string;
    name?: string;
  }

  export class UserRegisterCmd implements IUserRegisterCmd {
    static readonly _typeName: string;
    constructor(properties?: IUserRegisterCmd);
    // ...
  }

  // ... and so on for every command + result + record
}
```

### Generated Package Utilities (`acmeapp_package.js`)

```javascript
// Code generated by mvgen. DO NOT EDIT.
import { acmeappns } from './acmeapp.js';

export const PACKAGE_NAME = 'acmeappPackage';

// Factory: name → zero-value instance
export function instanceOf(compName) {
  switch (compName) {
    case 'User':             return new acmeappns.User();
    case 'UserRegisterCmd':  return new acmeappns.UserRegisterCmd();
    case 'UserRegisterCmdResult': return new acmeappns.UserRegisterCmdResult();
    // ... one case per command, result, and record
  }
  return null;
}

// Reverse: instance → name
export function nameOf(cmd) {
  if (!cmd || !cmd.constructor) return '';
  return cmd.constructor._typeName ?? '';
}
```

### Building a TypeScript Client with `@mainvec/mvpjs`

The `@mainvec/mvpjs` npm package provides the client-side runtime for sending commands to an MVP server. Projects typically create a typed client wrapper in a `client/` directory alongside the generated code.

#### Package Adapter (`client/acmeapp_package.ts`)

Wraps the generated `acmeapp_package.js` to implement the mvpjs `Package` interface:

```typescript
import type { Package } from '@mainvec/mvpjs';
import * as acmeappPkg from '../acmeapp_package.js';

export class AcmeAppPackage implements Package {
  getName(): string {
    return acmeappPkg.PACKAGE_NAME;
  }

  instanceOf(cmdName: string): unknown | undefined {
    return acmeappPkg.instanceOf(cmdName) ?? undefined;
  }

  nameOf(cmd: unknown): string {
    return acmeappPkg.nameOf(cmd);
  }
}
```

#### Typed Client (`client/acmeapp_client.ts`)

A full client with auth token management, interceptors, and typed command methods:

```typescript
import {
  newClient,
  type Client,
  type PackageClient,
  type ClientConfig,
  type ClientInterceptor,
  chainClient,
} from '@mainvec/mvpjs';
import { AcmeAppPackage } from './acmeapp_package';
import { acmeappns } from '../acmeapp';
import type { acmeappns as types } from '../acmeapp';

// Auth interceptor — injects token into every request
function authHeaderInterceptor(
  tokenProvider: () => string
): ClientInterceptor {
  return async (ctx, req, invoker) => {
    const token = tokenProvider();
    if (token) {
      const headers = req.headers ?? {};
      headers['auth'] = token;       // sent as 'x-mvp-auth' HTTP header
      req.headers = headers;
    }
    return invoker(ctx, req);
  };
}

export interface AcmeClientConfig {
  baseUrl: string;
  basePath?: string;
  storageType?: 'localStorage' | 'sessionStorage' | 'none';
  timeout?: number;
  headers?: Record<string, string>;
}

export class AcmeClient {
  private pkgClient: PackageClient;
  private authToken: string = '';
  private storageType: 'localStorage' | 'sessionStorage' | 'none';

  private constructor(
    pkgClient: PackageClient,
    storageType: 'localStorage' | 'sessionStorage' | 'none',
  ) {
    this.pkgClient = pkgClient;
    this.storageType = storageType;
  }

  static async create(config: AcmeClientConfig): Promise<AcmeClient> {
    const storageType = config.storageType ?? 'localStorage';
    const tokenHolder = { token: '' };

    // Restore token from storage
    if (storageType !== 'none' && typeof window !== 'undefined') {
      const storage = storageType === 'sessionStorage' ? sessionStorage : localStorage;
      tokenHolder.token = storage.getItem('acme_auth_token') ?? '';
    }

    // Create client with auth interceptor
    const interceptor = chainClient(
      authHeaderInterceptor(() => tokenHolder.token)
    );

    const mvpClient = newClient({
      baseUrl: config.baseUrl,
      basePath: config.basePath,
      timeout: config.timeout,
      headers: config.headers,
      interceptor: interceptor ?? undefined,
      fetch: typeof window !== 'undefined' ? window.fetch.bind(window) : undefined,
    });

    // Register the package
    const pkg = new AcmeAppPackage();
    const pkgClient = mvpClient.registerPackage(pkg);

    const client = new AcmeClient(pkgClient, storageType);
    client.authToken = tokenHolder.token;

    // Sync token mutations back to the holder
    const origSet = client.setAuthToken.bind(client);
    client.setAuthToken = (token: string) => {
      origSet(token);
      tokenHolder.token = token;
    };

    return client;
  }

  // Token management
  setAuthToken(token: string): void {
    this.authToken = token;
    if (this.storageType !== 'none' && typeof window !== 'undefined') {
      const storage = this.storageType === 'sessionStorage' ? sessionStorage : localStorage;
      storage.setItem('acme_auth_token', token);
    }
  }

  clearAuthToken(): void {
    this.authToken = '';
    if (this.storageType !== 'none' && typeof window !== 'undefined') {
      const storage = this.storageType === 'sessionStorage' ? sessionStorage : localStorage;
      storage.removeItem('acme_auth_token');
    }
  }

  isAuthenticated(): boolean {
    return this.authToken !== '';
  }

  // Generic command sender
  private async sendCmd<T>(cmd: unknown): Promise<T> {
    return this.pkgClient.sendCmd<T>(cmd);
  }

  // ── Typed command methods ──────────────────────────────────

  async registerUser(
    email: string, password: string, name: string
  ): Promise<types.UserRegisterCmdResult> {
    const cmd = new acmeappns.UserRegisterCmd({ email, password, name });
    const result = await this.sendCmd<types.UserRegisterCmdResult>(cmd);
    // Auto-set auth token on successful registration
    if (result.token) {
      this.setAuthToken(result.token);
    }
    return result;
  }

  async getUserProfile(
    userID: string
  ): Promise<types.UserGetProfileCmdResult> {
    const cmd = new acmeappns.UserGetProfileCmd({ userID });
    return this.sendCmd<types.UserGetProfileCmdResult>(cmd);
  }

  async updateSettings(
    userID: string, settings: Record<string, string>
  ): Promise<types.UserUpdateSettingsCmdResult> {
    const cmd = new acmeappns.UserUpdateSettingsCmd({ userID, settings });
    return this.sendCmd<types.UserUpdateSettingsCmdResult>(cmd);
  }

  async createOrder(
    customerID: string,
    items: types.IOrderItem[],
    notes?: string
  ): Promise<types.OrderCreateCmdResult> {
    const orderItems = items.map(i => new acmeappns.OrderItem(i));
    const cmd = new acmeappns.OrderCreateCmd({ customerID, items: orderItems, notes });
    return this.sendCmd<types.OrderCreateCmdResult>(cmd);
  }
}

// Convenience factory function
export async function newAcmeClient(config: AcmeClientConfig): Promise<AcmeClient> {
  return AcmeClient.create(config);
}
```

#### Barrel Exports (`client/index.ts`)

```typescript
export { AcmeClient, newAcmeClient } from './acmeapp_client';
export type { AcmeClientConfig } from './acmeapp_client';
export { AcmeAppPackage } from './acmeapp_package';
export { acmeappns } from '../acmeapp';
export type { acmeappns as AcmeAppTypes } from '../acmeapp';
```

### Using the Client (Example)

```typescript
import { newAcmeClient } from './mvpapi/js/api/client';

const client = await newAcmeClient({
  baseUrl: 'http://localhost:8080',
  basePath: '/api',
});

// Register — auto-sets auth token
const reg = await client.registerUser('alice@example.com', 's3cret', 'Alice');
console.log('User ID:', reg.userID);

// Subsequent calls include the auth token automatically
const profile = await client.getUserProfile(reg.userID);
console.log('Profile:', profile.user);

// Create an order
const order = await client.createOrder(reg.userID, [
  { productID: 'prod-1', quantity: 2, price: 29.99 },
  { productID: 'prod-2', quantity: 1, price: 49.99 },
], 'Rush delivery please');
console.log('Order:', order.orderID, 'Total:', order.total);

// Token persists across page refreshes (localStorage)
// On next page load, client restores the token automatically
```

### Client-Side Directory Convention

When adding a typed client, the recommended structure is:

```
js/
└── api/
    ├── acmeapp.js                  # ⛔ Generated
    ├── acmeapp.d.ts                # ⛔ Generated
    ├── acmeapp_package.js          # ⛔ Generated
    ├── acmeapp_package.d.ts        # ⛔ Generated
    └── client/                     # ✏️  Hand-written
        ├── acmeapp_client.ts       # Typed client with auth & command methods
        ├── acmeapp_package.ts      # Package adapter (mvpjs Package interface)
        └── index.ts                # Barrel exports
```

The `client/` directory is **hand-written** (not generated by mvgen). It wraps the generated code with project-specific concerns like auth token management, storage persistence, and typed convenience methods.

---

## Best Practices

### Field Numbers (`fnum`)

- Every field **must** have a unique `fnum` within its command or record.
- **Never reuse** a field number, even after deleting a field. Assign the next available number.
- Field numbers are **stable** — they map directly to protobuf field numbers and must not change after initial assignment.
- Keep a mental or documented record of the highest `fnum` used per command/record.

### Command Naming

- Use **PascalCase** with a `Cmd` suffix: `UserRegisterCmd`, `OrderCreateCmd`.
- CLI aliases use **snake_case**: `"alias": "register"`, `"alias": "create_order"`.

### Workflow

1. **Edit the spec** (`spec/*.json`) — add/modify commands, fields, or records.
2. **Validate** — run `mvgen validate --in ./spec/your-spec.json`.
3. **Regenerate** — run `bash generate_api.sh` from the `mvpapi/` directory.
4. **Implement** — update `*_impl.go` with business logic for new commands.
5. **Protect** — add `// NOMVGEN` to `*_impl.go` once you've customized it, so regeneration won't overwrite your work.

### Format Choice

- **`plain`** — Simpler. Go structs with JSON tags. No protobuf dependency. Good for REST/JSON APIs and getting started.
- **`pb3`** — Full protobuf. Use when you need binary serialization, gRPC compatibility, or strict schema evolution guarantees.

---

## Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Editing generated files (not `_impl.go`) | Only edit `*_impl.go`. All other files are regenerated. |
| Reusing field numbers | Always use the next available `fnum`. Never reuse deleted field numbers. |
| Duplicate `fnum` within a command/record | Each `fnum` must be unique within its parent. |
| Missing `$ref` for `recRef` fields | `recRef` fields require `"$ref": "#/recordsDefs/RecordName"`. |
| Case sensitivity in command names | Command names are PascalCase and case-sensitive. |
| Running generate without validating | Always `mvgen validate` first to catch spec errors before generating. |
| Forgetting `NOMVGEN` on customized files | Add `// NOMVGEN` as the first line of any generated file you've customized. |
| Wrong `go_package` path | Must match your actual Go module path. |

---

## For AI Agents — Quick Reference

> This section is structured for AI coding agents operating on projects that use MVP. It can be embedded in a project's `mvpapi/README.md` or `AGENT.md`.

### Identifying an MVP Project

A project uses MVP if it has a `mvpapi/` directory containing:
- `spec/*.json` or `spec/*.jsonc` — MVEP specification file(s)
- `generate_api.sh` — Code generation script
- `go/` and/or `js/` — Generated output directories

### Key Rules

1. **NEVER edit generated files** except `*_impl.go`. Look for the header comment `// code generated by mvgen` to identify generated files.
2. **ONLY edit `*_impl.go`** for command implementations. This is the single file where business logic lives.
3. **Preserve field numbers** — when adding fields, use the next `fnum` after the highest existing one. Never reuse or change existing field numbers.
4. **Regenerate after spec changes** — run `bash generate_api.sh` from the `mvpapi/` directory.
5. **Validate before generating** — run `mvgen validate --in ./spec/<name>-spec.json` first.

### Adding a New Command

1. Open the spec file (`mvpapi/spec/*.json`).
2. Add the command under `"commands"`:
   ```json
   "NewCommandCmd": {
     "title": "Description",
     "alias": "new_command",
     "fields": {
       "fieldName": { "fnum": 1, "type": "string" }
     },
     "resultFields": {
       "resultField": { "fnum": 1, "type": "string" }
     }
   }
   ```
3. Regenerate: `cd mvpapi && bash generate_api.sh`
4. Implement in `*_impl.go`: fill in the generated `runNewCommandCmd` stub.

### Adding a Field to an Existing Command

1. Open the spec file.
2. Find the command, check the highest `fnum` in its `fields` or `resultFields`.
3. Add the new field with `fnum` = highest + 1.
4. Regenerate.

### Adding a Record Definition

1. Open the spec file.
2. Add under `"recordsDefs"`:
   ```json
   "MyRecord": {
     "name": "MyRecord",
     "title": "Description",
     "fields": {
       "fieldName": { "fnum": 1, "type": "string" }
     }
   }
   ```
3. Reference it in commands with `"type": "recRef", "$ref": "#/recordsDefs/MyRecord"`.
4. Regenerate.

### File Quick-Reference

| Pattern | Purpose | Edit? |
|---------|---------|-------|
| `spec/*-spec.json` | MVEP specification | Yes — source of truth |
| `generate_api.sh` | Regeneration script | Yes — if paths change |
| `go/*_impl.go` | Command implementations | **Yes** — your business logic |
| `go/*_commands.go` | Command runner factory | No |
| `go/api/*_package.go` | Handler types, dispatch | No |
| `go/api/*.plain.go` | Go struct definitions | No |
| `go/api/*.proto` | Protobuf definitions | No |
| `go/cmd/*/` | CLI entry point | No |
| `js/api/*` | JS/TS classes and types | No |
| `js/api/client/*` | Typed TS client wrapper | **Yes** — hand-written |

### Condensed Template for Project README

Projects can include this in their `mvpapi/README.md`:

```markdown
## MVP API

This project uses the [Mainvec Platform](https://github.com/mainvec/mvep/mvgen) (MVP) for API generation.

### Regenerate API Code

    cd mvpapi && bash generate_api.sh

### Spec Location

    mvpapi/spec/<name>-spec.json

### Implementation

Business logic lives in `mvpapi/go/<name>_impl.go`. Do not edit other generated files.
```
