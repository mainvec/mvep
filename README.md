# mvep

**Mainvec Platform** — a spec-driven code generator that turns declarative
JSON service specifications into production-ready Go code, CLI tools, and
Protocol Buffer / plain-struct APIs.

MVP Toolkit ("mvgen") lets you describe a service as a JSON/JSONC spec
(commands, fields, records) and generates:

- Go API packages (plain structs or protobuf)
- CLI tools with flag-parsed subcommands
- Command runners and implementation stubs
- JavaScript/TypeScript API clients
- Protocol Buffer 3 schemas

## Install

```bash
go install github.com/mainvec/mvep/mvgen/mvpapi/cmd/mvp@latest
```

Or build from source:

```bash
git clone https://github.com/mainvec/mvep
cd mvep/mvgen
go build -o mvp ./mvpapi/cmd/mvp
```

## Quick start

```bash
# 1. Initialize a service spec
mvp init --name myservice --ns myservicens

# 2. Validate it
mvp validate --in myservice.jsonc

# 3. Generate Go code
mvp generate --in myservice.jsonc --lang go --outdir . --format plain
```

A minimal spec looks like:

```json
{
  "$id": "myservice",
  "$schema": "https://spec.mainvec.com/mvpspec/0.2/schema/2026-01-15",
  "name": "myservice",
  "namespace": "myservicens",
  "commands": {
    "CreateUserCmd": {
      "title": "create a new user",
      "alias": "create_user",
      "fields": {
        "name":  { "fnum": 1, "type": "string" },
        "email": { "fnum": 2, "type": "string" }
      },
      "resultFields": {
        "userId": { "fnum": 1, "type": "string" }
      }
    }
  }
}
```

## Documentation

Full documentation lives in [`mvgen/README.md`](mvgen/README.md), including:

- Spec format reference (commands, fields, records, maps)
- Field type table (string, int32, int64, boolean, recRef, map)
- Generated file structure and protection markers (`NOMVGEN`)
- CLI command reference (`init`, `validate`, `generate`)
- Architecture and template system

Additional guides:

- [`mvgen/AGENT.md`](mvgen/AGENT.md) — guide for AI agents working in the codebase
- [`mvgen/MVP_SKILL.md`](mvgen/MVP_SKILL.md) — integration guide for MVP Toolkit projects

## Repository layout

```
mvep/
├── mvgen/                  # the code generator (single Go module)
│   ├── mvpapi/             # current implementation + `mvp` CLI (plain format)
│   ├── gengen/             # self-generator: regenerates mvpapi/ from its own spec
│   ├── resources/
│   │   ├── codegen_templates/   # Go & JS code generation templates
│   │   ├── mvepspec/0.1/        # legacy schema (still supported)
│   │   └── mvpspec/0.2/         # current schema
│   └── testdata/           # test fixtures
└── docs/design/            # architecture diagrams
```

## Contributing

Contributions are welcome! Please open an issue or pull request on
[GitHub](https://github.com/mainvec/mvep).

## License

Licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE)
file for details.



