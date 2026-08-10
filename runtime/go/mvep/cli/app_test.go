package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// T8 test fixtures -------------------------------------------------------

// t8EchoCmd is a command struct with string + int32 fields.
type t8EchoCmd struct {
	In    string
	Count int32
}

// t8EchoCmdResult is the matching result.
type t8EchoCmdResult struct {
	Out string
}

// t8PingCmd is a no-field command.
type t8PingCmd struct{}

// t8PingCmdResult is the matching result.
type t8PingCmdResult struct {
	Pong bool
}

// t8Desc is a descriptor with two commands: EchoCmd (string+int32 fields) and
// PingCmd (no fields). Ptr closures reference the real structs so the
// generated-package property — a wrong field is a compile error — holds.
var t8Desc = mvep.PackageDesc{
	Name:        "t8pkg",
	SpecVersion: "0.2",
	Commands: []mvep.CommandDesc{
		{
			Name:  "EchoCmd",
			Alias: "echo",
			Desc:  "Echo a string N times",
			New:   func() any { return &t8EchoCmd{} },
			Fields: []mvep.FieldDesc{
				{
					Name: "in", Fnum: 1, Type: mvep.FieldString, Required: true,
					Desc: "input string",
					Ptr:  func(c any) any { return &c.(*t8EchoCmd).In },
				},
				{
					Name: "count", Fnum: 2, Type: mvep.FieldInt32,
					Desc: "repetition count",
					Ptr:  func(c any) any { return &c.(*t8EchoCmd).Count },
				},
			},
			Result: &mvep.ResultDesc{
				Name: "EchoCmdResult",
				New:  func() any { return &t8EchoCmdResult{} },
			},
		},
		{
			Name: "PingCmd",
			Desc: "Ping the server",
			New:  func() any { return &t8PingCmd{} },
			Result: &mvep.ResultDesc{
				Name: "PingCmdResult",
				New:  func() any { return &t8PingCmdResult{} },
			},
		},
	},
}

// recordingExecutor records the command it received and returns a canned result.
type recordingExecutor struct {
	gotCmd any
	result any
	err    error
}

func (e *recordingExecutor) Run(ctx context.Context, cmd any) (any, error) {
	e.gotCmd = cmd
	return e.result, e.err
}

// TestNewBuildsCommandTree verifies T8: cli.New walks a PackageDesc and builds
// a ugo command tree with one subcommand per described command. We verify by
// running --help and checking the listed subcommand names.
func TestNewBuildsCommandTree(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	combined := strings.ToLower(stdout.String() + stderr.String())
	if !strings.Contains(combined, "echo") {
		t.Errorf("help should list echo command; got: %s", combined)
	}
	if !strings.Contains(combined, "ping") {
		t.Errorf("help should list ping command; got: %s", combined)
	}
}

// TestAppRunExecutesEcho verifies T8: App.Run parses argv, binds flags into the
// command struct, dispatches via the Executor, and returns nil on success.
func TestAppRunExecutesEcho(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--count", "3"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd, ok := ex.gotCmd.(*t8EchoCmd)
	if !ok {
		t.Fatalf("executor received %T, want *t8EchoCmd", ex.gotCmd)
	}
	if cmd.In != "hello" {
		t.Errorf("In = %q, want %q", cmd.In, "hello")
	}
	if cmd.Count != 3 {
		t.Errorf("Count = %d, want 3", cmd.Count)
	}
}

// TestAppRunExecutesPing verifies T8: a no-field command dispatches with a
// zero-value struct.
func TestAppRunExecutesPing(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &t8PingCmdResult{Pong: true}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"ping_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := ex.gotCmd.(*t8PingCmd); !ok {
		t.Fatalf("executor received %T, want *t8PingCmd", ex.gotCmd)
	}
}

// TestAppRunPropagatesExecutorError verifies T8: an Executor error is returned
// from App.Run, so the caller can set the exit code.
func TestAppRunPropagatesExecutorError(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{err: errors.New("boom")}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"ping_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "boom")
	}
}

// TestAppRunUnknownCommand verifies T8: an unknown subcommand name yields an
// error, not a silent no-op.
func TestAppRunUnknownCommand(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"nonexistent_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
}

// TestAppRunHelp verifies T8: --help on the root lists the commands.
func TestAppRunHelp(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	// Help goes to stderr in ugo; check both.
	combined := stdout.String() + stderr.String()
	if !strings.Contains(strings.ToLower(combined), "echo") {
		t.Errorf("help output should list echo command; got: %s", combined)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
