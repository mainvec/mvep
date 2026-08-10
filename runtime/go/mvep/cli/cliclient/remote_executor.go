// Package cliclient provides the remote Executor adapter for mvep/cli, keeping
// the mvep/cli package free of a dependency on mvep/client. The subpackage
// exists solely to preserve that import boundary.
package cliclient

import (
	"context"

	"github.com/mainvec/mvep/runtime/go/mvep/cli"
	"github.com/mainvec/mvep/runtime/go/mvep/client"
)

// EXPERIMENTAL: see cli/executor.go.

// RemoteExecutor adapts a *client.PackageClient to the cli.Executor
// interface, so a single CLI binary can target a remote server by swapping
// in this executor for a cli.LocalExecutor.
//
// It calls PackageClient.SendCmdReq (not SendCmd): SendCmd flattens the
// server error into a string and drops the *mvep.CmdResp carrying
// Error.Code, which the CLI's exit-code classification (T14) needs. On a
// command failure the executor returns a *cli.ErrorCode wrapping the code.
type RemoteExecutor struct {
	pc *client.PackageClient
}

// NewRemoteExecutor returns a RemoteExecutor over the given PackageClient.
func NewRemoteExecutor(pc *client.PackageClient) *RemoteExecutor {
	return &RemoteExecutor{pc: pc}
}

// Run sends the command to the remote server via SendCmdReq and returns the
// result. On a server-side error it returns *cli.ErrorCode carrying the
// ErrorInfo.Code so T14 can classify the exit code without string parsing.
func (e *RemoteExecutor) Run(ctx context.Context, cmd any) (any, error) {
	result, resp, err := e.pc.SendCmdReq(ctx, cmd, nil)
	if err != nil {
		// If we have a CmdResp with a structured Error, wrap its Code so T14's
		// exit-code classification can key on the class. Otherwise return the
		// raw error (e.g. transport failure, invalid command name).
		if resp != nil && resp.Error != nil {
			return nil, &cli.ErrorCode{
				Code:    resp.Error.Code,
				Message: resp.Error.Message,
				Err:     err,
			}
		}
		return nil, err
	}
	return result, nil
}

// Compile-time assertion that *RemoteExecutor satisfies cli.Executor.
var _ cli.Executor = (*RemoteExecutor)(nil)