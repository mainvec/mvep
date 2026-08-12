// Package cli provides a runtime CLI builder that drives a generated package's
// commands via a *mvep.PackageDesc and an Executor (local or remote).
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
	oenc "github.com/mainvec/ugo/oencoding"
	_ "github.com/mainvec/ugo/oencoding/json"
)

// App is a descriptor-driven CLI. New builds it from a *mvep.PackageDesc and
// an Executor; RunWithIO parses argv, dispatches the matching command through
// the Executor, and returns the result or error.
type App struct {
	root     *cli.Command
	desc     *mvep.PackageDesc
	executor Executor
	// namespace is the reserved framework command group name (default "mvep").
	// It hosts the spec-independent machine surface (exec, send, list,
	// describe). Overridable via WithNamespace.
	namespace string
	// commands indexes every CommandDesc by its CLI name (commandName), built
	// during New. Used by the mvep exec dispatcher to look up a command by
	// name without re-walking the tree.
	commands map[string]*mvep.CommandDesc
	// stdin is the reader used for explicit/implicit stdin payloads (T3/T4).
	// Defaults to os.Stdin; testable via the Option below.
	stdin io.Reader
	// stdout/stderr are the io writers for the current RunWithIO invocation,
	// captured so send can rebuild a *cli.Context enriched with request/response
	// headers (ContextWithCmdReq returns a plain context.Context).
	stdout, stderr io.Writer
	// outputMode is the --<namespace>-output value ("text" or "json"),
	// defaulting to "text". Registered as a root persistent flag so it is
	// visible on every command; the flag name follows the configured
	// namespace.
	outputMode string
	// preHooks run after flag binding + required check, before the executor.
	// A pre-hook returning an error aborts execution. Hooks run in
	// registration order. Use for auth, logging, metrics (T13).
	preHooks []PreHook
	// postHooks run after the executor, receiving the result and error, before
	// rendering. The command still completes (the error propagates). Hooks
	// run in registration order.
	postHooks []PostHook
	// renderer renders the result to stdout. Defaults to defaultRenderer;
	// swapped via SetRenderer (T14).
	renderer Renderer
}

// Option configures an App at construction. Options are applied in order.
type Option func(*App)

// WithNamespace overrides the reserved framework namespace group name. The
// default is "mvep"; overriding it to "acme" yields `svc acme exec ...` and
// `--acme-output`. The name must not collide with any descriptor command or
// group (New panics otherwise).
func WithNamespace(name string) Option {
	return func(a *App) { a.namespace = name }
}

// WithStdin overrides the reader used for explicit/implicit stdin payloads.
// Defaults to os.Stdin. Testable injection point for the exec/send verbs.
func WithStdin(r io.Reader) Option {
	return func(a *App) { a.stdin = r }
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
// CommandDesc becomes a ugo subcommand under the root, or under a group parent
// when the command declares a Group (plan 040). Group parents are created
// lazily on first reference, in command-iteration order (commands are emitted
// sorted, so the tree is deterministic); each carries its own
// title/description/aliases/hidden flag and has no RunE so ugo prints its
// help. Flag binding (T9), required-flag enforcement (T10), and result
// rendering (T14) layer on top of this; T8 establishes the command tree and
// the execute-via-Executor seam.
func New(desc *mvep.PackageDesc, executor Executor, opts ...Option) *App {
	app := &App{
		desc:       desc,
		executor:   executor,
		renderer:   defaultRenderer,
		namespace:  "mvep",
		commands:   make(map[string]*mvep.CommandDesc, len(desc.Commands)),
		stdin:      os.Stdin,
		outputMode: "text",
	}
	for _, opt := range opts {
		opt(app)
	}

	root := &cli.Command{
		Usage: desc.Name,
		Short: desc.Title,
		Long:  desc.Desc,
		// The root takes no positional arguments: every argument is a
		// subcommand name or a flag. Without this, ugo prints help and
		// returns nil for an unrecognized command name (treating it as a
		// positional), which would silently swallow typos.
		Args: unknownSubcommandArgs,
	}
	if desc.SpecVersion != "" {
		root.Version = desc.SpecVersion
	}

	// Register the namespace-scoped persistent output flag on the root so it is
	// visible on every command. It is namespaced because a persistent root flag
	// claims a name on every generated command (--output stays the implementor's
	// to add). The name follows the configured namespace (T5).
	root.PersistentFlags().StringVar(&app.outputMode, app.namespace+"-output", "text", "output format (text|json)")

	// The reserved namespace claims one top-level identifier. A descriptor
	// command or group colliding with it is a construction error: the panic
	// names the reserved word. This is unreachable in generated code once T9
	// hard-errors at generation time, and matches ugo's AddCommand self-parent
	// panic.
	for i := range desc.Commands {
		if commandName(&desc.Commands[i]) == app.namespace {
			panic(fmt.Sprintf("reserved namespace %q collides with command %q", app.namespace, desc.Commands[i].Name))
		}
	}
	for i := range desc.Groups {
		if desc.Groups[i].Path == app.namespace {
			panic(fmt.Sprintf("reserved namespace %q collides with group %q", app.namespace, desc.Groups[i].Path))
		}
	}

	// groupByPath memoises a ugo *cli.Command per group path, created on
	// demand in desc.Groups order. The parent of a group is found by trimming
	// the last path segment, so a depth-2 group attaches under its depth-1
	// parent.
	groupByPath := make(map[string]*cli.Command, len(desc.Groups))
	var groupFor func(path string) *cli.Command
	groupFor = func(path string) *cli.Command {
		if path == "" {
			return root
		}
		if c, ok := groupByPath[path]; ok {
			return c
		}
		// Find the GroupDesc for this path (auto-created intermediates are
		// materialised in the descriptor by codegen, so it is always present).
		var gd *mvep.GroupDesc
		for i := range desc.Groups {
			if desc.Groups[i].Path == path {
				gd = &desc.Groups[i]
				break
			}
		}
		parentPath, name := splitGroupPath(path)
		parent := groupFor(parentPath)
		g := &cli.Command{
			Usage:   name,
			Short:   groupTitle(gd, name),
			Long:    groupLong(gd),
			Aliases: groupAliases(gd),
			Hidden:  groupHidden(gd),
			// A group takes no positional arguments: every argument is a
			// subcommand name or a flag. Without this, ugo treats an unknown
			// subcommand as a positional and silently prints help.
			Args: unknownSubcommandArgs,
		}
		parent.AddCommand(g)
		groupByPath[path] = g
		return g
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
		bindings := bindFlags(sub.Flags(), cmdDesc, desc)
		sub.RunE = func(ctx *cli.Context, args []string) error {
			return app.runCommand(ctx, cmdDesc, bindings)
		}
		groupFor(cmdDesc.Group).AddCommand(sub)
	}

	// Register the reserved namespace group under the root. Its verbs (exec,
	// send, list, describe) are added by later tasks; the group itself is the
	// self-documenting home for the framework surface. It is Runnable so it
	// appears in root help even before any verb is registered: running
	// `svc mvep` alone prints the group's help.
	ns := &cli.Command{
		Usage: app.namespace,
		Short: "framework commands",
		Args:  unknownSubcommandArgs,
	}
	ns.RunE = func(ctx *cli.Context, args []string) error {
		(&cli.CommandHelp{Command: ns}).WriteHelp(ctx)
		return nil
	}
	app.registerExec(ns)
	app.registerSend(ns)
	root.AddCommand(ns)

	app.root = root
	return app
}

// unknownSubcommandArgs is the Args guard shared by the root and every group
// parent: no positional arguments are accepted, so an unrecognized subcommand
// name is an error rather than a silently-printed help. It reports the full
// command path (e.g. "keys") via CommandPath so a nested group's error names
// the group, not just the leaf segment.
func unknownSubcommandArgs(cmd *cli.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return nil
}

// groupTitle returns the group's display title, falling back to the path's
// final segment when the group is undeclared (no metadata).
func groupTitle(gd *mvep.GroupDesc, name string) string {
	if gd != nil && gd.Title != "" {
		return gd.Title
	}
	return name
}

// groupLong returns the group's long description.
func groupLong(gd *mvep.GroupDesc) string {
	if gd != nil {
		return gd.Desc
	}
	return ""
}

// groupAliases returns the group's declared aliases.
func groupAliases(gd *mvep.GroupDesc) []string {
	if gd != nil {
		return gd.Aliases
	}
	return nil
}

// groupHidden reports whether the group is hidden.
func groupHidden(gd *mvep.GroupDesc) bool {
	return gd != nil && gd.Hidden
}

// splitGroupPath splits a group path into its parent path and final segment.
// The root's parent is "" and leaf is the whole path.
func splitGroupPath(path string) (parent, leaf string) {
	if path == "" {
		return "", ""
	}
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", path
	}
	return path[:idx], path[idx+1:]
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
//
// Under --<namespace>-output json, errors are serialized as {"error":...} on
// stdout (never stderr) and the error is still returned, so exit codes are
// unchanged. When flag parsing itself fails the output flag may never have been
// parsed, so the mode falls back to a pre-scan of the raw args (T5).
func (a *App) RunWithIO(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	a.stdout = stdout
	a.stderr = stderr

	fw := &cli.Framework{Root: a.root}
	var err error
	if args == nil {
		err = fw.Run(ctx)
	} else {
		err = fw.ExecuteWithIO(ctx, args, stdout, stderr)
	}

	if err != nil && a.jsonOutputFor(args) {
		renderJSONError(stdout, err)
	}
	return err
}

// jsonOutputFor reports whether the output mode is JSON, using the parsed
// outputMode when available and falling back to a pre-scan of the raw args
// when flag parsing failed before the output flag was read.
func (a *App) jsonOutputFor(args []string) bool {
	if a.outputMode == "json" {
		return true
	}
	if args == nil {
		return false
	}
	prefix := "--" + a.namespace + "-output"
	for i, arg := range args {
		switch {
		case arg == prefix:
			if i+1 < len(args) && args[i+1] == "json" {
				return true
			}
			return false
		case strings.HasPrefix(arg, prefix+"="):
			return strings.HasSuffix(arg, "=json")
		}
	}
	return false
}

// runCommand constructs the command struct, applies flag values via the Ptr
// accessors, enforces required flags, dispatches through the Executor, and
// renders the result. This is the flag path: it differs from the payload path
// (dispatch) only in how the command struct is populated; both share the
// identical tail in execute.
func (a *App) runCommand(ctx *cli.Context, cmdDesc *mvep.CommandDesc, bindings []flagBinding) error {
	cmd := cmdDesc.New()

	// Write the parsed flag values into the command struct via the descriptor's
	// Ptr accessors. T9 owns the full type-switch. A non-nil error means a
	// flag value could not be written (e.g. invalid JSON for --*-json flags);
	// abort before dispatch (#29).
	if err := applyBindings(cmd, bindings); err != nil {
		return err
	}

	_, err := a.execute(ctx, cmdDesc, cmd)
	return err
}

// dispatch is the payload path. It looks up a command by CLI name, validates
// the payload keys against the descriptor, decodes the payload via the same
// encoding registry the server uses, and runs the shared execution core,
// returning the result. exec renders via the shared tail; send (T6) uses the
// result directly to emit a CmdResp envelope.
func (a *App) dispatch(ctx *cli.Context, name string, payload []byte) (any, error) {
	cmdDesc, ok := a.commands[name]
	if !ok {
		return nil, fmt.Errorf("unknown command %q (valid: %s)", name, strings.Join(a.commandNames(), ", "))
	}

	if err := validatePayloadKeys(cmdDesc, a.desc, payload); err != nil {
		return nil, err
	}

	cmd := cmdDesc.New()
	enc, ok := oenc.LookupEncoding("application/json")
	if !ok {
		return nil, fmt.Errorf("no application/json encoder registered")
	}
	if err := enc.Decode(payload, cmd); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}

	return a.runCore(ctx, cmdDesc, cmd)
}

// execute is the shared tail of both the flag and payload paths: required-flag
// checking, pre-hooks, executor.Run, post-hooks, and rendering. It returns the
// result (and error). Keeping this identical across both paths is what
// guarantees hooks, required-checking, and rendering cannot drift between
// flag-driven and payload-driven invocation.
func (a *App) execute(ctx *cli.Context, cmdDesc *mvep.CommandDesc, cmd any) (any, error) {
	result, err := a.runCore(ctx, cmdDesc, cmd)
	if err != nil {
		return nil, err
	}

	// Render the result (T14). The renderer writes to the command's stdout
	// (the ugo Context implements io.Writer). Under --<namespace>-output json,
	// a JSON renderer is selected (T5); otherwise the default text renderer.
	if result != nil {
		if a.outputMode == "json" {
			jsonRenderer(ctx, result)
		} else {
			a.renderer(ctx, result)
		}
	}
	return result, nil
}

// runCore is the shared execution core (no rendering): required-flag checking,
// pre-hooks, executor.Run, and post-hooks. execute wraps it with rendering;
// send (T6) uses it directly and emits CmdResp envelopes instead.
func (a *App) runCore(ctx *cli.Context, cmdDesc *mvep.CommandDesc, cmd any) (any, error) {
	// Enforce required flags (T10): a required field left at its zero value
	// after parsing is a usage error, not an execution error. Enforcement
	// lives in cli, not ugo, so the behaviour is ours to define. The error
	// names the missing flag so the caller can surface it and exit 2.
	if err := checkRequired(cmdDesc, cmd); err != nil {
		return nil, err
	}

	// Pre-hooks (T13): run in registration order before the executor. A
	// pre-hook returning an error aborts — the executor is NOT called.
	for _, h := range a.preHooks {
		if err := h(ctx, cmd); err != nil {
			return nil, err
		}
	}

	result, err := a.executor.Run(ctx, cmd)

	// Post-hooks (T13): run after the executor, receiving the result and any
	// error, before rendering. The error propagates regardless of what the
	// post-hook does.
	for _, h := range a.postHooks {
		h(ctx, cmd, result, err)
	}

	return result, err
}

// commandNames returns the sorted CLI names of every command, for error and
// list output.
func (a *App) commandNames() []string {
	names := make([]string, 0, len(a.commands))
	for n := range a.commands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
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

// commandName converts a CommandDesc to the CLI subcommand name. The spec's
// alias field is the shell-friendly name (e.g. "generate", "init", "validate");
// when absent, the CamelCase name is snake_cased (e.g. "EchoCmd" -> "echo_cmd").
func commandName(d *mvep.CommandDesc) string {
	if d.Alias != "" {
		return d.Alias
	}
	return toSnake(d.Name)
}

// aliasesFor returns the aliases for a command: the snake_case of the
// descriptor name (so the CamelCase name is still reachable) plus any
// additional spec-declared aliases beyond the primary one.
func aliasesFor(d *mvep.CommandDesc) []string {
	var out []string
	// When the alias is the primary name, the snake_case name becomes an alias.
	if d.Alias != "" {
		out = append(out, toSnake(d.Name))
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
