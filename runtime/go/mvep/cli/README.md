# mvep/cli

A descriptor-driven CLI builder for MVEP packages. It walks a `*mvep.PackageDesc`
and builds a [ugo/cli](https://github.com/mainvec/ugo) command tree, binding flags
to struct fields via `FieldDesc.Ptr` and dispatching through an `Executor` (local
or remote). This replaces the legacy generated `main.go` that hardwired one
`prepareXxxCmd` function per command.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/mainvec/mvep/runtime/go/mvep/cli"
    "github.com/mainvec/ugo/cli" // for cli.LocalExecutor
)

func main() {
    // api.Describe() returns the *PackageDesc emitted by codegen (T3).
    // GetCommandRunner() returns the PkgCommandRunner wired by the implementor.
    app := cli.New(api.Describe(), &cli.LocalExecutor{Runner: runner})
    app.Root().Version = "v1.0.0"

    err := app.Run(context.Background())
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(cli.ExitCode(err))
    }
}
```

The generated `go_cli_runtime_main.txt` template emits this pattern; the
codegen `cli` gen_option (`runtime` | `legacy` | `none`) selects it.

## Key types

### `cli.New(desc *mvep.PackageDesc, executor Executor) *App`

Builds the command tree: one subcommand per `CommandDesc`, with flags bound
via `FieldDesc.Ptr`. The command name is the spec's `alias` (or snake_case
of the struct name if no alias). The snake_case name is registered as a ugo
alias so both forms work.

## Command groups

A command can be placed under a group by setting its `Group` field to a
`/`-separated path (empty or absent means the root). `cli.New` builds the
nested tree from `desc.Groups` in order, so `svc server start` dispatches to a
command with `Group: "server"` and `Alias: "start"`.

Group parents are built with their own title, description, aliases and hidden
flag, and have no `RunE`, so ugo prints their help when invoked with no
subcommand. They also carry the same unknown-subcommand guard as the root, so
`svc server bogus` is an error rather than a silently-printed help.

```go
// A command under the "server" group, reachable as `svc server start`.
mvep.CommandDesc{
    Name:  "StartServerCmd",
    Alias: "start",
    Group: "server",
    // ...
}
```

Groups are a CLI presentation concern: they do not affect `GetName()`, routes,
envelopes or encodings, and `Group` is not consulted by the server or client.
A spec that declares no group generates the same flat tree as before.

### `Executor`

```go
type Executor interface {
    Run(ctx context.Context, cmd any) (any, error)
}
```

One interface for both local and remote execution:

- `cli.LocalExecutor{Runner: mvep.CommandRunner}` — in-process.
- `cliclient.NewRemoteExecutor(pkgClient)` — remote via
  `PackageClient.SendCmdReq`, in the `mvep/cli/cliclient` subpackage (keeps
  `mvep/cli` free of a `mvep/client` dependency). Wraps `CmdResp.Error.Code`
  in `*cli.ErrorCode` for exit-code classification.

### `App`

- `app.Run(ctx)` — executes with `os.Args` and STDIO.
- `app.RunWithIO(ctx, args, stdout, stderr)` — testable entry point.
- `app.Root()` — the ugo `*cli.Command`; extend with global flags
  (`PersistentFlags()`), custom subcommands (`AddCommand()`), or override
  generated commands (`Find()` + replace `RunE`).
- `app.SetRenderer(func(w io.Writer, result any))` — swap the default
  text renderer (e.g. for JSON output driven by `--output`).
- `app.AddPreHook(func(ctx, cmd) error)` — runs before the executor; a
  non-nil error aborts. Use for auth, logging.
- `app.AddPostHook(func(ctx, cmd, result, err))` — runs after the executor.
  Use for logging, metrics.

### `cli.ExitCode(err) int`

Maps an error to a process exit code:

| Code | Class | Source |
|------|-------|--------|
| 0 | success | nil error |
| 2 | usage | missing required flag (T10) |
| 3 | not-found | `http_404` (unknown command, remote) |
| 4 | auth | `http_401`, `http_403` |
| 1 | other | all other execution errors |

## Flag binding (T9)

Every `FieldType` is bound via the `Ptr` accessor:

| Type | Flag | Notes |
|------|------|-------|
| string | `StringVar` | |
| bool | `BoolVar` | |
| int32, sint32 | `Int32Var` | both are `*int32` |
| int64 | `Int64Var` | |
| uint32 | `Uint32Var` | custom `flag.Value` (ugo ships no helper) |
| float32 | `Float32Var` | custom `flag.Value` |
| float64 | `Float64Var` | |
| bytes | `BytesVar` | base64 or `@file` |
| timestamp | custom `timeValue` | RFC3339 |
| duration | `DurationVar` | |
| uuid | string + `TextUnmarshaler` | no `google/uuid` dependency |
| map | `--<name>-json` | JSON object |
| record (`$ref`) | `--<name>-<field>` | depth-1 flattening |
| repeated string | `StringSliceVar` | accumulates `--flag` occurrences |
| repeated other | `--<name>-json` | JSON array |

The binding does **not** mutate the shared descriptor's `Ptr` — parsed values
are held in per-execution locals and written back via `Ptr` after parsing.

### Sub-field binding and repeated fields (T1, T8)

Depth-1 record flattening agrees with top-level binding on every `FieldType` ×
`Repeated` combination, so the two can never drift. `--<record>-<field>-file`
supersedes `--<record>-<field>-json`, which supersedes the flattened
sub-flags.

| Field | Top-level form | Sub-field form |
|-------|----------------|----------------|
| map | `--<name>-json`, `--<name>-file` | — |
| record (`$ref`) | `--<name>-<field>`, `--<name>-json`, `--<name>-file` | `--<name>-<field>` |
| repeated string | `StringSliceVar` (`--name a --name b`) | `--<record>-<field> 'a' --<record>-<field> 'b'` |
| repeated non-string | `--<name>-json`, `--<name>-file` | `--<record>-<field>-json`, `--<record>-<field>-file` |

Repeated `FieldString` binds as a repeatable flag (a JSON array is emitted).
Every other repeated type (UUID, timestamp, duration, bytes, numeric, bool,
map, `recRef`) binds via `--<name>-json` / `--<name>-file` as a JSON array,
matching the top-level `registerRepeatedFlag` fallback exactly. Malformed or
non-array `-json`/`-file` values error naming the sub-field flag.

## Required flags (T10)

`FieldDesc.Required` is enforced in `cli`, not ugo. A missing required flag
returns `"required flag --<name> is missing"` and maps to exit code 2. A
required field set to its zero value intentionally (e.g. `--count 0`) is
indistinguishable from missing — the standard CLI convention.

## The reserved `mvep` namespace (T2–T7)

Every generated CLI reserves a single `mvep` group (`svc mvep <verb>`) that
provides a spec-independent, machine-readable surface. It is overridable via
`cli.New(desc, executor, cli.WithNamespace("acme"))`, which also renames the
persistent output flag to `--acme-output`.

```
cat p.json | svc mvep exec StartServerCmd       # implicit stdin (pipe)
svc mvep exec --input p.json StartServerCmd     # flags precede the command name
svc mvep exec --input - StartServerCmd          # explicit stdin
cat reqs.ndjson | svc mvep send                 # CmdReq stream -> CmdResp stream
svc mvep list                                   # names + descriptions
svc mvep describe [command]                     # versioned schema projection
```

The `mvep` verbs address commands by their **descriptor `Name`** (e.g.
`StartServerCmd`), the same identity the server uses on the wire
(`InstanceOf` / `CmdReq.Cmd`). Names are unique and collision-free, so grouped
commands whose aliases repeat are still unambiguous here. Groups and aliases
are a human-facing presentation concern — use them on the flag path
(`svc server start`), not in the `mvep` namespace.

- **`exec`** reads a complete payload from `--input <path>`, `--input -`, or
  implicitly from stdin when stdin is not a terminal. Payload keys are
  validated against the descriptor (unknown keys hard-error), then decoded with
  the same encoder registry the server uses.
- **`send`** reads a stream of `CmdReq` envelopes (NDJSON or concatenated) and
  emits one `CmdResp` per record, flushing immediately so it works in a live
  pipeline. `CmdReq.Payload` is `[]byte`, so inputs carry base64 payloads, and
  `CmdReq.Cmd` carries the descriptor `Name`. `--fail-fast` halts at the first
  error; the process exits non-zero if any record errored. Request headers ride
  the context (`mvep.ContextWithCmdReq`), so header-reading interceptors behave
  identically under the CLI and over HTTP; response headers set via
  `mvep.SetResponseHeader` round-trip into the emitted `CmdResp`.
- **`list`** prints each command's `Name` and description (a JSON array of
  `{name, description}` objects under `--mvep-output json`).
- **`describe`** emits a minimal, versioned JSON projection — name, alias,
  group, description, fields (name, type, repeated, required, ref), result type.
  No argument describes all.

Because ugo inherits stdlib `flag` parsing, which stops at the first non-flag
argument, `--input`/`--fail-fast` must precede the command name inside the
namespace verbs.

## `--mvep-output json|text` (T5)

A persistent flag visible on every command (`--<namespace>-output`, default
`text`) selects the renderer:

- **`text`** (default) — today's human-readable output, byte for byte.
- **`json`** — the result is marshaled to stdout as JSON. Errors serialize to
  stdout as `{"error":{"code":...,"message":...}}` (using `mvep.ErrorInfo`),
  nothing on stderr, and the error is still returned so exit codes are
  unchanged. A failed `exec` is shaped exactly like a `send` record's
  `CmdResp.Error`, so one consumer parses both.

The implementor's own `--output` flag is unaffected — it never collides with
the namespaced `--mvep-output`.

## gen_options.cli

The CLI mode is a spec gen_option, not a CLI flag:

| Mode | Behaviour |
|------|-----------|
| `runtime` (default) | Emits `go_cli_runtime_main.txt` — descriptor-driven `cli.New` |
| `legacy` | Emits `go_cli_main.txt` — hand-wired `prepareXxxCmd` pattern |
| `none` | No CLI main generated |

`skipCmd=true` forces `none` regardless of the genopt. NOMVEP/NOMVGEN
markers are honoured in both `legacy` and `runtime` modes.

## Format support

The descriptor-driven runtime CLI (`mvep exec`/`send`/`list`/`describe`) is
**plain-format only**. It decodes payloads via `oenc.LookupEncoding("application/json")`,
which resolves to the plain stdlib JSON encoder — protojson is registered under
the short name `"protojson"` only, never `application/json`. Driving pb3
messages through stdlib `encoding/json` would mangle enums, `oneof`, and
well-known types, so pb3 is not supported by this CLI. The toolkit rejects
`format: pb3` combined with the runtime CLI at generation time; use `cli:
legacy` or `cli: none` for pb3 specs.