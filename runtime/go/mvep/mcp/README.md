# mvep/mcp

Serve any MVEP package as a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
tool server with zero extra dependencies.

```go
import "github.com/mainvec/mvep/runtime/go/mvep/mcp"
```

Every command in a generated `mvep.PackageDesc` becomes an MCP tool returned by
`tools/list`, and every `tools/call` dispatches through `mvep.PackageHandler.ServeCmdReq`
so the full interceptor chain (`RecoveryInterceptor`, `AuthInterceptor`,
`LoggingInterceptor`, request IDs) applies automatically.

## Quick start

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/mvep/runtime/go/mvep/mcp"
	myservice "github.com/yourorg/yourproject/mvepapi/go"
	"github.com/yourorg/yourproject/mvepapi/go/api"
)

func main() {
	// Stdio transport owns os.Stdout — send all application logs to os.Stderr.
	log.SetOutput(os.Stderr)

	handler := &mvep.PackageHandler{
		Package:       api.NewPackage(),
		CommandRunner: myservice.GetCommandRunner(),
		Interceptor:   mvep.Chain(mvep.RecoveryInterceptor()),
	}

	srv, err := mcp.NewServer(api.Describe(), handler)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Serve(context.Background(), mcp.Stdio()); err != nil {
		log.Fatal(err)
	}
}
```

## Tool naming and groups

By default (`mcp.DefaultToolNamer`):

- The leaf name is `CommandDesc.Alias` when set, otherwise `snake_case(CommandDesc.Name)`.
- When `CommandDesc.Group` is set, its `/`-separated segments are joined with `_`
  and prefixed: e.g. `Group: "server/tls"`, `Alias: "rotate"` → `server_tls_rotate`.
- Any tool-name collision fails `NewServer` / `SchemasFromPackage` immediately.

Pass `mcp.WithToolNamer(mcp.DescriptorNameToolNamer)` to expose descriptor
`Name`s verbatim (e.g. `StartServerCmd`), or supply a custom `mcp.ToolNamer`.

## Schema mapping

`SchemasFromPackage` maps all 14 `mvep.FieldType` values onto JSON Schema:

| `mvep.FieldType` | JSON Schema |
| --- | --- |
| `FieldString` | `{"type": "string"}` |
| `FieldBool` | `{"type": "boolean"}` |
| `FieldInt32`, `FieldInt64`, `FieldSint32` | `{"type": "integer"}` |
| `FieldUint32` | `{"type": "integer", "minimum": 0}` |
| `FieldFloat`, `FieldDouble` | `{"type": "number"}` |
| `FieldBytes` | `{"type": "string", "format": "byte"}` |
| `FieldTimestamp` | `{"type": "string", "format": "date-time"}` |
| `FieldDuration` | `{"type": "integer", "description": "duration in nanoseconds"}` |
| `FieldUUID` | `{"type": "string", "format": "uuid"}` |
| `FieldMap` | `{"type": "object"}` (permissive — descriptor carries no value type) |
| `FieldRecord` | Nested object resolved via `PackageDesc.Record`, with cycle guard |
| `Repeated: true` | `{"type": "array", "items": ...}` |

Unknown arguments and missing required arguments are rejected before dispatch
and returned as MCP tool errors (`isError: true`).

## Constraints and out of scope (v1)

- **Plain format only.** `tools/call` decodes JSON arguments via
  `application/json` (stdlib JSON), matching `mvep/cli`.
- **No async jobs in v1.** `SubmitJob` / `GetJobStatus` rely on MVEP request/response
  headers, which MCP has no channel for.
- **Stdio transport in v1.** `Transport` is an interface so HTTP/SSE transports
  can be added without changing `Server`.
