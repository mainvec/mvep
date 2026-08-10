# CLI Builder Migration (plan 025, #25)

The toolkit's CLI generation has moved from a hand-wired `main.go` (one
`prepareXxxCmd` function per command) to a descriptor-driven library
(`mvep/cli`). This document describes what changed and how to migrate.

## What changed

### Before (legacy)

`go_cli_main.txt` emitted a `package main` with one `prepareXxxCmd` function
per command. Each function declared local flag variables, registered them on
a `cli.Command`, and called the typed handler directly. The template
hardcoded `package main`, supported only string/boolean/int32 flags,
discarded command results, and returned no meaningful exit codes. Anything
the template didn't emit — a global `--endpoint` flag, an extra subcommand,
JSON output, an auth hook — required editing generated code (lost on regen)
or forking the template.

### After (runtime)

`go_cli_runtime_main.txt` emits a `package main` that calls
`cli.New(api.Describe(), &cli.LocalExecutor{Runner: runner})`. The command
tree, flag binding, required-flag enforcement, and exit codes are all driven
by the `PackageDesc` and the `mvep/cli` library. The implementor extends via
`App.Root()` (global flags, custom subcommands, overrides), `AddPreHook`/
`AddPostHook` (auth, logging), and `SetRenderer` (JSON output).

## Selecting the mode

The `cli` gen_option controls which template is emitted:

| `gen_options.cli` | Template | Behaviour |
|---|---|---|
| `runtime` (default) | `go_cli_runtime_main.txt` | Descriptor-driven `cli.New` |
| `legacy` | `go_cli_main.txt` | Hand-wired `prepareXxxCmd` (old behaviour) |
| `none` | — | No CLI main generated |

`skipCmd=true` forces `none` regardless of the genopt.

```jsonc
{
    "gen_options": {
        "cli": "runtime",
        "format": "plain"
    }
}
```

Absent `cli` genopt → `runtime` (the default flipped from `legacy`).

## What the generated main looks like

```go
func main() {
    app := cli.New(api.Describe(), &cli.LocalExecutor{Runner: runner})
    app.Root().Version = resolveVersion()
    err := app.Run(context.Background())
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(cli.ExitCode(err))
    }
}
```

The `mvep_main_cmd.go` file is protected by `// NOMVEP` (or legacy
`// NOMVGEN`/`// NOWOGEN`) so hand-customized entry points are not
overwritten on regeneration.

## Command names

The CLI uses the spec's `alias` as the command name (e.g. `generate`,
`init`, `validate`). The snake_case of the struct name (`generate_cmd`) is
registered as a ugo alias, so both forms work. The `Short` description is
populated from the spec's `title` (falling back to `desc`).

## Flag binding

Every spec field type is bound via the `FieldDesc.Ptr` accessor — no
reflection in the binding path. See [mvep/cli/README.md](../runtime/go/mvep/cli/README.md#flag-binding-t9)
for the full type table. Records flatten to depth 1 (`--record-field`);
maps and deeper nesting bind via `--<name>-json`.

## Exit codes

| Code | Class | Source |
|------|-------|--------|
| 0 | success | nil error |
| 2 | usage | missing required flag |
| 3 | not-found | `http_404` (unknown command, remote) |
| 4 | auth | `http_401`, `http_403` |
| 1 | other | all other execution errors |

## Extending the CLI

```go
app := cli.New(api.Describe(), &cli.LocalExecutor{Runner: runner})

// Global flag (inherited by all subcommands)
var endpoint string
app.Root().PersistentFlags().StringVar(&endpoint, "endpoint", "localhost:8080", "server endpoint")

// Custom subcommand
app.Root().AddCommand(&cli.Command{
    Usage: "version",
    Short: "Print version and exit",
    RunE:  func(ctx *cli.Context, args []string) error { fmt.Println("v1.0.0"); return nil },
})

// Pre-hook (auth check)
app.AddPreHook(func(ctx *cli.Context, cmd any) error {
    if endpoint == "" { return errors.New("endpoint required") }
    return nil
})

// JSON output
var output string
app.Root().PersistentFlags().StringVar(&output, "output", "text", "output format")
app.SetRenderer(func(w io.Writer, result any) {
    if output == "json" { json.NewEncoder(w).Encode(result) } else { fmt.Fprintln(w, result) }
})
```

## Release ordering

The `runtime` CLI mode generates a `main.go` that imports
`github.com/mainvec/mvep/runtime/go/mvep/cli`. This package does not exist
in `runtime/go v0.9.0` — it ships in `runtime/go v0.10.0`. Until
`toolkit/go.mod` is bumped to require the newer runtime, the generated code
compiles only through `go.work` (local checkout). A toolkit release that
ships the `runtime` default must follow a `runtime/go` release tag.