# MVEP Toolkit - Mainvec Platform Code Generator

MVEP Toolkit is an internal code generation tool for the Mainvec Platform that converts service specifications into production-ready code. It generates Go implementations, JavaScript/TypeScript APIs, CLI tools, and Protocol Buffer artifacts from declarative JSON specifications.

## Overview

MVEP Toolkit enables declarative API development by:
- Converting MVEP specifications to working code
- Generating Protocol Buffer 3 schemas
- Creating complete Go implementations with CLI tools
- Validating specifications against JSON schemas
- Producing type-safe command handlers and runners

**Current Version:** v0.1.4
**Language Support:** Go, JavaScript/TypeScript

## Installation

```bash
go install <module>/mvepapi/cmd/mvep@latest
```

Preferred implementation path in this repository is `mvepapi` (plain format) with the `mvep` command.

Or build from source:

```bash
git clone https://github.com/mainvec/mvep
cd mvep/toolkit
go build -o mvep ./mvepapi/cmd/mvep
```

## Quick Start

### 1. Initialize a New Service Specification

```bash
mvep init --name myservice --ns myservicens
```

This creates a basic MVEP specification file.

### 2. Define Your Service

Create a file `myservice.jsonc`:

```json
{
  "$id": "myservice",
  "$schema": "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
  "name": "myservice",
  "namespace": "myservicens",
  "version": "v0.1",
  "gen_options": {
    "go_package": "github.com/myorg/myservice/mvepapi;myservice",
    "go_api_package": "github.com/myorg/myservice/mvepapi/api;api"
  },
  "commands": {
    "CreateUserCmd": {
      "title": "create a new user",
      "alias": "create_user",
      "fields": {
        "name": {"fnum": 1, "type": "string", "title": "user name"},
        "email": {"fnum": 2, "type": "string", "title": "user email"}
      },
      "resultFields": {
        "userId": {"fnum": 1, "type": "string", "title": "created user ID"}
      }
    }
  }
}
```

### 3. Validate Your Specification

```bash
mvep validate --in myservice.jsonc
```

### 4. Generate Code

```bash
mvep generate --in myservice.jsonc --lang go --outdir ./output --format plain
```

This generates:
- `api/myservice.plain.go` - Generated Go structs (plain mode)
- `api/myservice_package.go` - Command handlers and dispatcher
- `cmd/myservice/myservice_main_cmd.go` - CLI entry point
- `myservice_impl.go` - Implementation stubs
- `myservice_commands.go` - Command runner factory

If you use `--format pb3`, generated protobuf files are also produced (`api/myservice.proto`, `api/myservice.pb.go`).

### 5. Implement Your Commands

Edit the generated `myservice_impl.go`:

```go
func runCreateUserCmd(ctx context.Context, cmd *api.CreateUserCmd) (*api.CreateUserCmdResult, error) {
    // Your implementation here
    userId := generateUserId()

    return &api.CreateUserCmdResult{
        UserId: userId,
    }, nil
}
```

### 6. Build and Run

```bash
cd output
go mod init github.com/myorg/myservice
go mod tidy
go build -o myservice ./cmd/myservice

./myservice create_user --name "John Doe" --email "john@example.com"
```

## MVEP Specification Format

### Basic Structure

```json
{
  "$id": "service-id",
  "$schema": "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
  "name": "service-name",
  "namespace": "namespace",
  "version": "v0.1",
  "gen_options": {
    "go_package": "import/path;package",
    "go_api_package": "import/path/api;api"
  },
  "commands": { ... },
  "recordsDefs": { ... }
}
```

### Supported Field Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text string | `"hello"` |
| `boolean` | Boolean value | `true` or `false` |
| `int32` | 32-bit integer | `42` |
| `int64` | 64-bit integer | `9223372036854775807` |
| `recRef` | Reference to a record definition | See Records section |
| `map` | Key-value map | See Maps section |

### Commands

Commands define operations with input fields and result fields:

```json
{
  "commands": {
    "CommandName": {
      "title": "Human-readable description",
      "alias": "cli_command_name",
      "fields": {
        "fieldName": {
          "fnum": 1,
          "type": "string",
          "title": "Field description",
          "repeated": false
        }
      },
      "resultFields": {
        "resultField": {
          "fnum": 1,
          "type": "string",
          "title": "Result description"
        }
      }
    }
  }
}
```

### Record Definitions

Records are reusable data structures:

```json
{
  "recordsDefs": {
    "User": {
      "fields": {
        "id": {"fnum": 1, "type": "string"},
        "name": {"fnum": 2, "type": "string"},
        "email": {"fnum": 3, "type": "string"},
        "createdAt": {"fnum": 4, "type": "int64"}
      }
    }
  }
}
```

Reference a record in commands:

```json
{
  "fields": {
    "user": {
      "fnum": 1,
      "type": "recRef",
      "$ref": "#/recordsDefs/User"
    }
  }
}
```

### Maps

Define map fields:

```json
{
  "fields": {
    "metadata": {
      "fnum": 1,
      "type": "map",
      "valueType": "string"
    }
  }
}
```

### Repeated Fields (Arrays)

```json
{
  "fields": {
    "tags": {
      "fnum": 1,
      "type": "string",
      "repeated": true
    }
  }
}
```

## Generated File Structure

```
output/
├── api/
│   ├── myservice.plain.go        # Generated Go structs (DO NOT EDIT)
│   └── myservice_package.go      # Command handlers (DO NOT EDIT)
├── cmd/
│   └── myservice/
│       └── myservice_main_cmd.go # CLI entry point (DO NOT EDIT)
├── myservice_impl.go             # Implementations (EDIT THIS)
└── myservice_commands.go         # Command runner (DO NOT EDIT)
```

### File Protection

The generator respects file protection markers:
- `// NOMVGEN` - File will not be regenerated
- `// NOWOGEN` - Alternative protection marker

Place these at the top of files you want to protect from regeneration.

## CLI Commands

### generate

Generate code from MVEP specification:

```bash
mvep generate --in <spec-file> --lang <language> [--outdir <directory>] [--format plain|pb3]
```

**Flags:**
- `--in` (required) - Input MVEP specification file
- `--lang` (required) - Target language (`go`, `js`, or `go,js`)
- `--outdir` - Output directory (default: current directory)
- `--format` - Output format (`plain` default, `pb3` optional)

### gen

Alias of `generate`:

```bash
mvep gen --in <spec-file> --lang <language> [--outdir <directory>] [--format plain|pb3]
```

### validate

Validate an MVEP specification against the schema:

```bash
mvep validate --in <spec-file>
```

**Flags:**
- `--in` (required) - Input MVEP specification file

### init

Initialize a new MVEP specification:

```bash
mvep init --name <service-name> --ns <namespace>
```

**Flags:**
- `--name` - Service name
- `--ns` - Service namespace

## Integration Examples

### Example: mboxy Project

The [mboxy](https://github.com/mainvec/mboxy) project uses MVEP Toolkit to generate a message box service:

**Specification:** `spec/mboxy-spec.json` defines 11 commands:
- Account management (create, list)
- Chat operations (create, get)
- Message handling (create, get, list)

**Generated files:**
- Protocol buffers for message serialization
- CLI tool with 11 commands
- Command handlers and implementations

### Example: iulink Project

The [iulink](https://github.com/mainvec/iulink) project uses a related tool (wogen) following the same patterns:

**Specification:** `specs/iunet-spec.json` defines networking commands:
- Hub, serverlet, clientlet management
- Network exposure and connection
- Broker operations

## Architecture

### Code Generation Pipeline

```
MVEP JSON Spec
    ↓
Schema Validation
    ↓
Build SrvDef Structure
    ↓
    ├→ Generate Protobuf3 Definition
    │      ↓
    │   Compile to .proto
    │      ↓
    │   Run protoc
    │      ↓
    │   Generate .pb.go
    │
    └→ Generate Go Code from Templates
           ↓
        CLI Main
        Package Handlers
        Command Runner
        Implementation Stubs
```

### Key Components

| File | Purpose |
|------|---------|
| [toolkit.go](toolkit.go) | Core structures, validation, spec parsing |
| [toolkit_pb3.go](toolkit_pb3.go) | Protocol Buffer 3 generation |
| [toolkit_go.go](toolkit_go.go) | Go code generation, protoc compilation |
| [toolkit_runner.go](toolkit_runner.go) | Command execution, file generation |
| [resources/codegen_templates/go/](resources/codegen_templates/go/) | Go code templates (20+ templates) |

### Template System

The generator uses Go's `text/template` with custom functions:
- `ToUpper` - Convert to uppercase
- `ToTitle` - Convert to title case
- `ToCamel` - Convert to camelCase

Templates available:
- CLI frameworks (standard, cobra)
- Implementation stubs
- Package utilities
- Command runners
- API interfaces
- Server/client implementations
- Test generation
- NATS-based starters

## Dependencies

### Core Dependencies

- `github.com/bufbuild/protocompile` - Protocol Buffer compilation
- `github.com/jhump/protoreflect` - Protobuf reflection and dynamic descriptors
- `google.golang.org/protobuf` - Modern protobuf API
- `github.com/santhosh-tekuri/jsonschema/v6` - JSON Schema validation

### MainVec Ecosystem

- `github.com/mainvec/ugo` - Utilities (CLI framework, validation)
- `github.com/mainvec/mvep/runtime/go` - MainVec platform Go library

## Best Practices

### 1. Field Numbering

Field numbers (`fnum`) must be:
- Unique within a command or record
- Stable (never change existing numbers)
- Sequential starting from 1

### 2. Command Naming

- Use PascalCase with `Cmd` suffix: `CreateUserCmd`
- Use snake_case for CLI aliases: `create_user`

### 3. Record References

- Define records in `recordsDefs` section
- Reference using `"type": "recRef"` with `"$ref": "#/recordsDefs/RecordName"`

### 4. Version Management

- Keep `$schema` URL stable
- Use semantic versioning for your service
- Document breaking changes

### 5. Code Organization

- Keep implementation logic in `*_impl.go` files
- Don't modify generated files marked "DO NOT EDIT"
- Use `NOMVGEN` marker to protect custom files

## Troubleshooting

### Validation Errors

```
Error: invalid specification
```

Check:
1. JSON syntax is valid
2. `$schema` URL is correct
3. All required fields are present
4. Field numbers are unique

### Generation Failures

```
Error: protoc compilation failed
```

Ensure:
1. `protoc` is installed and in PATH
2. `protoc-gen-go` plugin is installed
3. Output directory exists and is writable

### Missing Dependencies

```
Error: package not found
```

Run:
```bash
go mod tidy
```

## Contributing

Contributions are welcome! Please open an issue or pull request on [GitHub](https://github.com/mainvec/mvep).

## License

Licensed under the Apache License, Version 2.0. See the [LICENSE](../LICENSE) file for details.

## See Also

- [MVEP Specification Schema](https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15)
- Legacy compatibility: `https://spec.mainvec.com/mvepspec/0.1/schema/2023-09-19` and `.../2026-01-15` remain supported.
- [Protocol Buffers Documentation](https://protobuf.dev/)
- [MainVec Platform Documentation](https://docs.mainvec.com/)
