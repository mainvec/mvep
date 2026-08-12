# mvep

**MVEP (Mainvec Engineering Platform)** — a spec-driven code generator plus the
Go and TypeScript runtimes for the services it produces.

The **toolkit** lets you describe a service as a JSON/JSONC spec (commands,
fields, records) and generates:

- Go API packages (plain structs or protobuf)
- CLI tools with flag-parsed subcommands
- Command runners and implementation stubs
- JavaScript/TypeScript API clients
- Protocol Buffer 3 schemas

Generated code runs on the bundled **runtimes**: the Go runtime
(`github.com/mainvec/mvep/runtime/go`) and the TypeScript runtime
(`@mainvec/mvep`).

## Install

```bash
go install github.com/mainvec/mvep/toolkit/mvepapi/cmd/mvep@latest
```

Or build from source:

```bash
git clone https://github.com/mainvec/mvep
cd mvep/toolkit
go build -o mvep ./mvepapi/cmd/mvep
```

## Quick start

```bash
# 1. Initialize a service spec
mvep init --name myservice --ns myservicens

# 2. Validate it
mvep validate --in myservice.jsonc

# 3. Generate Go code
mvep generate --in myservice.jsonc --lang go --outdir . --format plain
```

A minimal spec looks like:

```json
{
  "$id": "myservice",
  "$schema": "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
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

Full documentation lives in [`toolkit/README.md`](toolkit/README.md), including:

- Spec format reference (commands, fields, records, maps)
- Field type table (string, int32, int64, boolean, recRef, map)
- Generated file structure and protection markers (`NOMVEP`)
- CLI command reference (`init`, `validate`, `generate`)
- Architecture and template system

Additional guides:

- [`toolkit/AGENT.md`](toolkit/AGENT.md) — guide for AI agents working in the codebase
- [`toolkit/MVEP_SKILL.md`](toolkit/MVEP_SKILL.md) — integration guide for MVEP Toolkit projects

## The generated CLI and the `mvep` namespace

Every generated CLI (the `runtime` gen_option) is built by the descriptor-driven
`mvep/cli` builder and reserves a single `mvep` group that provides a
spec-independent, machine-readable surface:

```
cat p.json | svc mvep exec generate            # run a command from a JSON payload
svc mvep exec --input p.json generate          # flags precede the command name
cat reqs.ndjson | svc mvep send                # stream CmdReq -> CmdResp envelopes
svc mvep list                                  # command names
svc mvep describe [command]                    # versioned schema projection
```

- **`mvep exec`** reads a complete payload from a file, explicit stdin, or
  implicitly from a pipe, validates its keys against the descriptor, and decodes
  with the same encoder the server uses.
- **`mvep send`** streams `CmdReq` envelopes and emits one `CmdResp` per record,
  flushing immediately for live pipelines.
- **`--mvep-output json|text`** (a persistent flag on every command) renders
  results and errors as machine-readable JSON.

The namespace name is overridable via `cli.New(desc, executor,
cli.WithNamespace("acme"))`, which also renames the output flag to
`--acme-output`. A spec that declares a top-level command or group named `mvep`
fails generation (reserved-name validation). See
[`runtime/go/mvep/cli/README.md`](runtime/go/mvep/cli/README.md) for the full
guide.

## Repository layout

```
mvep/                         # multi-module monorepo (root go.work)
├── toolkit/                  # the code generator (module github.com/mainvec/mvep/toolkit)
│   ├── mvepapi/              # current implementation + `mvep` CLI (plain format)
│   ├── gengen/              # self-generator: regenerates mvepapi/ from its own spec
│   ├── resources/
│   │   ├── codegen_templates/   # Go & JS code generation templates
│   │   ├── mvepspec/0.1/        # legacy schema (still supported)
│   │   ├── mvepspec/0.2/        # current canonical schema
│   │   └── mvpspec/0.2/         # alias of 0.2 (legacy $schema URL, kept resolvable)
│   └── testdata/           # test fixtures
├── runtime/
│   ├── go/                   # Go runtime (module github.com/mainvec/mvep/runtime/go, package mvep)
│   └── ts/                   # TypeScript runtime (npm @mainvec/mvep)
├── go.work                   # ties toolkit + runtime/go for local dev
└── docs/design/            # architecture diagrams
```

## Contributing

Contributions are welcome! Please open an issue or pull request on
[GitHub](https://github.com/mainvec/mvep).

## License

Licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE)
file for details.



