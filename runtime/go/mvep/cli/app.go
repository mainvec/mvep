// Package cli provides a runtime CLI builder that drives a generated package's
// commands via a *mvep.PackageDesc and an Executor (local or remote).
package cli

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// EXPERIMENTAL: the App surface is core public API from day one but its shape
// may change for one release cycle. The marker is removed once the CLI builder
// (T16 of plan 025) has dogfooded the design.

// App is a descriptor-driven CLI. New builds it from a *mvep.PackageDesc and
// an Executor; RunWithIO parses argv, dispatches the matching command through
// the Executor, and returns the result or error.
type App struct {
	root     *cli.Command
	desc     *mvep.PackageDesc
	executor Executor
	// commandName maps a ugo command's Name() to the descriptor CommandDesc
	// that produced it, so RunE can look up the New() closure and fields.
	commands map[string]*mvep.CommandDesc
	// preHooks run after flag binding + required check, before the executor.
	// A pre-hook returning an error aborts execution. Hooks run in
	// registration order. Use for auth, logging, metrics (T13).
	preHooks []PreHook
	// postHooks run after the executor, receiving the result and error, before
	// rendering. The command still completes (the error propagates). Hooks
	// run in registration order.
	postHooks []PostHook
}

// PreHook runs before the executor dispatches a command. Returning an error
// aborts execution — the executor is NOT called. Use for auth checks, input
// validation, or logging.
type PreHook func(ctx *cli.Context, cmd any) error

// PostHook runs after the executor returns, receiving the result and any
// error. The command's error (if any) propagates to the caller regardless of
// what the post-hook does. Use for logging, metrics, or result inspection.
type PostHook func(ctx *cli.Context, cmd, result any, err error)

// AddPreHook registers a hook that runs before the executor. Hooks run in
// registration order. A pre-hook returning an error aborts execution.
func (a *App) AddPreHook(h PreHook) { a.preHooks = append(a.preHooks, h) }

// AddPostHook registers a hook that runs after the executor, receiving the
// result and error. Hooks run in registration order.
func (a *App) AddPostHook(h PostHook) { a.postHooks = append(a.postHooks, h) }

// New builds a CLI app from a package descriptor and an executor. Every
// CommandDesc becomes a ugo subcommand under the root. Flag binding (T9),
// required-flag enforcement (T10), and result rendering (T14) layer on top of
// this; T8 establishes the command tree and the execute-via-Executor seam.
func New(desc *mvep.PackageDesc, executor Executor) *App {
	app := &App{
		desc:     desc,
		executor: executor,
		commands: make(map[string]*mvep.CommandDesc, len(desc.Commands)),
	}

	root := &cli.Command{
		Usage: desc.Name,
		Short: desc.Title,
		Long:  desc.Desc,
		// The root takes no positional arguments: every argument is a
		// subcommand name or a flag. Without this, ugo prints help and
		// returns nil for an unrecognized command name (treating it as a
		// positional), which would silently swallow typos.
		Args: func(cmd *cli.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command %q for %q", args[0], cmd.Name())
			}
			return nil
		},
	}
	if desc.SpecVersion != "" {
		root.Version = desc.SpecVersion
	}

	for i := range desc.Commands {
		cmdDesc := &desc.Commands[i]
		cmdName := commandName(cmdDesc)
		app.commands[cmdName] = cmdDesc

		// Declare sub first so its FlagSet exists for bindFlags; the RunE
		// closure captures bindings, which is assigned next.
		sub := &cli.Command{
			Usage:   cmdName,
			Short:   cmdDesc.Desc,
			Aliases: aliasesFor(cmdDesc),
		}
		// Bindings are per-command but must be allocated once (not per RunE
		// call), because ugo parses flags against the command's FlagSet which
		// is created on first Flags() call. We register flags eagerly so the
		// same FlagSet and its parsed values are reused.
		bindings := bindFlags(sub.Flags(), cmdDesc)
		sub.RunE = func(ctx *cli.Context, args []string) error {
			return app.runCommand(ctx, cmdDesc, bindings)
		}
		root.AddCommand(sub)
	}

	app.root = root
	return app
}

// Root returns the underlying ugo root command, for callers that want to add
// custom subcommands or global flags (T12).
func (a *App) Root() *cli.Command { return a.root }

// Run executes the CLI with os.Args and STDIO.
func (a *App) Run(ctx context.Context) error {
	return a.RunWithIO(ctx, nil, nil, nil)
}

// RunWithIO executes the CLI with the given args and IO writers. When args is
// nil, os.Args[1:] is used; when stdout/stderr are nil, os.Stdout/os.Stderr are
// used. This mirrors ugo's Framework.ExecuteWithIO and is the testable entry
// point.
func (a *App) RunWithIO(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fw := &cli.Framework{Root: a.root}
	if args == nil {
		return fw.Run(ctx)
	}
	return fw.ExecuteWithIO(ctx, args, stdout, stderr)
}

// runCommand constructs the command struct, applies flag values via the Ptr
// accessors, enforces required flags, dispatches through the Executor, and
// renders the result.
func (a *App) runCommand(ctx *cli.Context, cmdDesc *mvep.CommandDesc, bindings []flagBinding) error {
	cmd := cmdDesc.New()

	// Write the parsed flag values into the command struct via the descriptor's
	// Ptr accessors. T9 owns the full type-switch.
	applyBindings(cmd, bindings)

	// Enforce required flags (T10): a required field left at its zero value
	// after parsing is a usage error, not an execution error. Enforcement
	// lives in cli, not ugo, so the behaviour is ours to define. The error
	// names the missing flag so the caller can surface it and exit 2.
	if err := checkRequired(cmdDesc, cmd); err != nil {
		return err
	}

	// Pre-hooks (T13): run in registration order before the executor. A
	// pre-hook returning an error aborts — the executor is NOT called.
	for _, h := range a.preHooks {
		if err := h(ctx, cmd); err != nil {
			return err
		}
	}

	result, err := a.executor.Run(ctx, cmd)

	// Post-hooks (T13): run after the executor, receiving the result and any
	// error, before rendering. The error propagates regardless of what the
	// post-hook does.
	for _, h := range a.postHooks {
		h(ctx, cmd, result, err)
	}

	if err != nil {
		return err
	}

	// T14 will add result rendering. For now, print a simple representation
	// so the caller can see the command ran.
	if result != nil {
		fmt.Fprintln(ctx, result)
	}
	return nil
}

// checkRequired returns an error naming the first required field whose value
// is still the zero value of its type after flag parsing. A required field set
// to its zero value intentionally (e.g. --count 0) is indistinguishable from a
// missing flag; this is the standard CLI convention for required flags.
func checkRequired(cmdDesc *mvep.CommandDesc, cmd any) error {
	for i := range cmdDesc.Fields {
		f := &cmdDesc.Fields[i]
		if !f.Required {
			continue
		}
		if isZeroValue(f.Ptr(cmd)) {
			return fmt.Errorf("required flag --%s is missing", f.Name)
		}
	}
	return nil
}

// isZeroValue reports whether v (a pointer returned by FieldDesc.Ptr) points
// at the zero value of its type. It type-switches on the concrete pointer type
// because Ptr returns any.
func isZeroValue(v any) bool {
	switch p := v.(type) {
	case *string:
		return *p == ""
	case *bool:
		return !*p
	case *int32:
		return *p == 0
	case *int64:
		return *p == 0
	case *uint32:
		return *p == 0
	case *float32:
		return *p == 0
	case *float64:
		return *p == 0
	case *[]byte:
		return len(*p) == 0
	case *[]string:
		return len(*p) == 0
	case *map[string]string:
		return len(*p) == 0
	default:
		// For pointer-to-struct fields (records) and types we can't probe
		// (time.Time, uuid.UUID), fall back to reflect. A nil pointer or a
		// zero struct is treated as missing.
		return isZeroReflect(v)
	}
}

// isZeroReflect uses reflect to test whether v points at a zero value for
// types not handled by the isZeroValue type-switch (time.Time, uuid.UUID,
// pointer-to-struct records). Ptr returns a pointer, so we deref one level.
func isZeroReflect(v any) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return true
	}
	elem := rv.Elem()
	if elem.Kind() == reflect.Ptr {
		// **Record: zero if the inner pointer is nil.
		return elem.IsNil()
	}
	return elem.IsZero()
}

// commandName converts a CommandDesc name to the CLI subcommand name. The
// descriptor carries Go CamelCase names (e.g. "EchoCmd"); the CLI uses
// snake_case (e.g. "echo_cmd") for a shell-friendly UX. The alias preserves
// the original name and the spec alias.
func commandName(d *mvep.CommandDesc) string {
	return toSnake(d.Name)
}

// aliasesFor returns the aliases for a command: the original CamelCase name and
// the spec-declared alias (if any).
func aliasesFor(d *mvep.CommandDesc) []string {
	var out []string
	if d.Alias != "" {
		out = append(out, d.Alias)
	}
	return out
}

// toSnake converts CamelCase to snake_case.
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
