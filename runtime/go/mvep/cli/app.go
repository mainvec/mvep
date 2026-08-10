// Package cli provides a runtime CLI builder that drives a generated package's
// commands via a *mvep.PackageDesc and an Executor (local or remote).
package cli

import (
	"context"
	"fmt"
	"io"
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
}

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
// accessors, dispatches through the Executor, and renders the result.
func (a *App) runCommand(ctx *cli.Context, cmdDesc *mvep.CommandDesc, bindings []flagBinding) error {
	cmd := cmdDesc.New()

	// Write the parsed flag values into the command struct via the descriptor's
	// Ptr accessors. T9 owns the full type-switch.
	applyBindings(cmd, bindings)

	result, err := a.executor.Run(ctx, cmd)
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
