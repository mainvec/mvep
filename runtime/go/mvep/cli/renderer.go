package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// EXPERIMENTAL: see executor.go.

// Renderer renders a command result to the given writer. The default renderer
// prints a human-readable representation; a JSON renderer marshals the result.
// Implementors swap the renderer via App.SetRenderer (T14).
type Renderer func(w io.Writer, result any)

// defaultRenderer prints the result using fmt, which is a reasonable
// human-readable default for struct types (they format with their field
// names). T14's plan calls for a "default human renderer plus --output=json";
// the --output flag is the implementor's to add via App.Root().
// PersistentFlags(), and the renderer switch happens in the custom renderer.
func defaultRenderer(w io.Writer, result any) {
	if result == nil {
		return
	}
	fmt.Fprintln(w, result)
}

// SetRenderer replaces the default result renderer. The implementor uses this
// to switch between text and JSON output (e.g. based on a --output flag they
// added via App.Root().PersistentFlags()).
func (a *App) SetRenderer(r Renderer) { a.renderer = r }

// ExitCode maps a command error to a process exit code. The mapping keys on
// error-code classes, not HTTP statuses, so it is scriptable and honest:
//
//   0 — success (nil error)
//   2 — usage (flag parse errors, missing required flags — T10)
//   3 — not-found (unknown_command, http_404)
//   4 — auth (unauthorized, forbidden, http_401, http_403)
//   1 — all other execution errors
//
// One wrinkle: PackageHandler.executeCmd collapses every runner error to
// command_error, so classes 3 and 4 only surface for pre-dispatch failures
// (interceptor rejections, unknown command names); the common in-command
// failure lands in class 1.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	// *ErrorCode carries the wire error code (http_<status> for remote;
	// absent for local). Classify by code prefix.
	var ec *ErrorCode
	if errors.As(err, &ec) {
		return exitCodeForCode(ec.Code)
	}

	// Required-flag errors (T10) are usage errors → exit 2.
	if isUsageError(err) {
		return 2
	}

	// Everything else is a generic execution error → exit 1.
	return 1
}

// exitCodeForCode maps a wire error code to an exit-code class.
func exitCodeForCode(code string) int {
	switch {
	case strings.HasPrefix(code, "http_404"):
		return 3 // not-found
	case strings.HasPrefix(code, "http_401"), strings.HasPrefix(code, "http_403"):
		return 4 // auth
	default:
		return 1 // other execution error (http_500, command_error, etc.)
	}
}

// isUsageError reports whether err is a required-flag-missing error from T10.
// The check is string-based because T10 returns a plain error (not a typed
// one); a typed error could be added later if the string match is too loose.
func isUsageError(err error) bool {
	return strings.Contains(err.Error(), "required flag")
}