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

## Required flags (T10)

`FieldDesc.Required` is enforced in `cli`, not ugo. A missing required flag
returns `"required flag --<name> is missing"` and maps to exit code 2. A
required field set to its zero value intentionally (e.g. `--count 0`) is
indistinguishable from missing — the standard CLI convention.

## gen_options.cli

The CLI mode is a spec gen_option, not a CLI flag:

| Mode | Behaviour |
|------|-----------|
| `runtime` (default) | Emits `go_cli_runtime_main.txt` — descriptor-driven `cli.New` |
| `legacy` | Emits `go_cli_main.txt` — hand-wired `prepareXxxCmd` pattern |
| `none` | No CLI main generated |

`skipCmd=true` forces `none` regardless of the genopt. NOMVEP/NOMVGEN
markers are honoured in both `legacy` and `runtime` modes.