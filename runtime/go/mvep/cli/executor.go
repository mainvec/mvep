package cli

import (
	"context"
	"fmt"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// EXPERIMENTAL: the Executor surface is core public API from day one but its
// shape may change for one release cycle. The marker is removed once the CLI
// builder (T16 of plan 025) has dogfooded the design.

// Executor runs a command and returns its result. One interface covers both
// in-process execution (via mvep.CommandRunner) and remote execution (via
// client.PackageClient), so a single CLI binary can target either by
// swapping the executor. This is the execution seam the CLI builder (T8)
// drives every command through.
type Executor interface {
	Run(ctx context.Context, cmd any) (any, error)
}

// LocalExecutor adapts an in-process mvep.CommandRunner to the Executor
// interface. It forwards the command and returns the runner's result and
// error unchanged.
type LocalExecutor struct {
	Runner mvep.CommandRunner
}

// Run delegates to the underlying CommandRunner.RunCmd.
func (e *LocalExecutor) Run(ctx context.Context, cmd any) (any, error) {
	return e.Runner.RunCmd(ctx, cmd)
}

// ErrorCode is a typed error wrapping a server-side mvep.ErrorInfo.Code so T14's
// exit-code classification can key on the code class without string parsing.
// RemoteExecutor returns *ErrorCode for command failures; LocalExecutor
// returns the runner's error directly (no code is available in-process).
type ErrorCode struct {
	Code    string
	Message string
	Err     error // wrapped for errors.Is/As chains
}

func (e *ErrorCode) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ErrorCode) Unwrap() error { return e.Err }