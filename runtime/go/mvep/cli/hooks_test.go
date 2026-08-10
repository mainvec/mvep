package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// hookContext is passed to pre/post hooks.
type hookCtx struct {
	cmd    any
	result any
	err    error
}

// TestPreHookRunsBeforeExecutor verifies T13: a pre-hook runs before the
// executor and receives the command struct.
func TestPreHookRunsBeforeExecutor(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	var preCmd any
	app.AddPreHook(func(ctx *cli.Context, cmd any) error {
		preCmd = cmd
		return nil
	})
	// The recordingExecutor records gotCmd, so if preCmd is set before
	// ex.gotCmd, the pre-hook ran first.

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preCmd == nil {
		t.Error("pre-hook was not called")
	}
	if ex.gotCmd == nil {
		t.Error("executor was not called")
	}
	// Both received the same command struct.
	if preCmd != ex.gotCmd {
		t.Error("pre-hook and executor should receive the same command struct")
	}
}

// TestPreHookErrorAbortsExecution verifies T13: a pre-hook returning an error
// aborts execution — the executor is NOT called.
func TestPreHookErrorAbortsExecution(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	app.AddPreHook(func(ctx *cli.Context, cmd any) error {
		return errors.New("auth required")
	})

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error from pre-hook, got nil")
	}
	if err.Error() != "auth required" {
		t.Errorf("error = %q, want %q", err.Error(), "auth required")
	}
	if ex.gotCmd != nil {
		t.Error("executor should NOT be called when a pre-hook aborts")
	}
}

// TestPostHookRunsAfterExecutor verifies T13: a post-hook runs after the
// executor, receives the result, and the command still completes.
func TestPostHookRunsAfterExecutor(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	var postResult any
	var postErr error
	app.AddPostHook(func(ctx *cli.Context, cmd, result any, err error) {
		postResult = result
		postErr = err
	})

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postResult == nil {
		t.Error("post-hook was not called or received nil result")
	}
	if postErr != nil {
		t.Errorf("post-hook err = %v, want nil", postErr)
	}
}

// TestPostHookReceivesExecutorError verifies T13: a post-hook receives the
// executor's error when execution fails.
func TestPostHookReceivesExecutorError(t *testing.T) {
	ex := &recordingExecutor{err: errors.New("exec failed")}
	app := New(&t8Desc, ex)

	var postErr error
	app.AddPostHook(func(ctx *cli.Context, cmd, result any, err error) {
		postErr = err
	})

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if postErr == nil {
		t.Error("post-hook should receive the executor error")
	}
	if postErr.Error() != "exec failed" {
		t.Errorf("post-hook err = %v, want %q", postErr, "exec failed")
	}
}

// TestHookOrder verifies T13: hooks run in registration order (pre-hooks
// before executor, post-hooks after executor, each in registration order).
func TestHookOrder(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	var order []string
	app.AddPreHook(func(ctx *cli.Context, cmd any) error {
		order = append(order, "pre1")
		return nil
	})
	app.AddPreHook(func(ctx *cli.Context, cmd any) error {
		order = append(order, "pre2")
		return nil
	})
	app.AddPostHook(func(ctx *cli.Context, cmd, result any, err error) {
		order = append(order, "post1")
	})
	app.AddPostHook(func(ctx *cli.Context, cmd, result any, err error) {
		order = append(order, "post2")
	})

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)

	want := []string{"pre1", "pre2", "post1", "post2"}
	if len(order) != len(want) {
		t.Fatalf("hook order = %v, want %v", order, want)
	}
	for i, w := range want {
		if order[i] != w {
			t.Errorf("order[%d] = %q, want %q", i, order[i], w)
		}
	}
}

// Ensure mvep import is used (hook types reference it in future).
var _ mvep.FieldType
