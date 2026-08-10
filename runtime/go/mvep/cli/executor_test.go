package cli

import (
	"context"
	"errors"
	"testing"
)

// fakeRunner is a minimal mvep.CommandRunner that records the command it
// received and returns a canned result/error, for testing LocalExecutor.
type fakeRunner struct {
	gotCmd any
	result any
	err    error
}

func (r *fakeRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
	r.gotCmd = cmd
	return r.result, r.err
}

// TestLocalExecutor verifies T7: LocalExecutor adapts mvep.CommandRunner to
// the Executor interface — it forwards the command and returns the runner's
// result and error unchanged.
func TestLocalExecutor(t *testing.T) {
	t.Parallel()

	t.Run("forwards command and returns result", func(t *testing.T) {
		type echoCmd struct{ Msg string }
		cmd := &echoCmd{Msg: "hi"}
		runner := &fakeRunner{result: "ok"}
			ex := &LocalExecutor{Runner: runner}

		got, err := ex.Run(context.Background(), cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "ok" {
			t.Errorf("result = %v, want %q", got, "ok")
		}
		if runner.gotCmd != cmd {
			t.Errorf("runner received %v, want %v", runner.gotCmd, cmd)
		}
	})

	t.Run("propagates runner error", func(t *testing.T) {
		runner := &fakeRunner{err: errors.New("boom")}
		ex := &LocalExecutor{Runner: runner}

		_, err := ex.Run(context.Background(), &struct{}{})
		if err == nil || err.Error() != "boom" {
			t.Errorf("error = %v, want %q", err, "boom")
		}
	})
}

// TestLocalExecutorSatisfiesExecutor is a compile-time assertion that
// LocalExecutor satisfies the Executor interface.
func TestLocalExecutorSatisfiesExecutor(t *testing.T) {
	t.Parallel()

	var _ Executor = &LocalExecutor{Runner: &fakeRunner{}}
}

// TestExecutorInterfaceIsSatisfied is a compile-time assertion that both
// adapters satisfy the Executor interface, guarding against signature drift.
func TestExecutorInterfaceIsSatisfied(t *testing.T) {
	t.Parallel()

	var _ Executor = &LocalExecutor{Runner: &fakeRunner{}}
	// RemoteExecutor is covered in the cliclient subpackage tests.
}