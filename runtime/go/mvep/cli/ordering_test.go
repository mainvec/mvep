package cli

import (
	"bytes"
	"context"
	"testing"
)

// TestHelpIsByteStableAcrossRuns verifies T11: the --help output for the root
// command and for a subcommand is byte-identical across repeated runs within
// one process. This guards against nondeterminism in either the descriptor
// (ordered slices, T1) or ugo's help rendering (command/flag iteration order).
func TestHelpIsByteStableAcrossRuns(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	getHelp := func(args ...string) string {
		var stdout, stderr bytes.Buffer
		_ = app.RunWithIO(context.Background(), args, &stdout, &stderr)
		// ugo writes help to stderr.
		return stderr.String() + stdout.String()
	}

	// Root help: lists all subcommands.
	firstRoot := getHelp("--help")
	if firstRoot == "" {
		t.Fatal("root --help produced empty output")
	}
	for i := 0; i < 10; i++ {
		if got := getHelp("--help"); got != firstRoot {
			t.Fatalf("root --help pass %d differs from pass 0 (non-deterministic)\n--- pass 0 ---\n%s\n--- pass %d ---\n%s", i, firstRoot, i, got)
		}
	}

	// Subcommand help: lists the command's flags.
	firstSub := getHelp("echo_cmd", "--help")
	if firstSub == "" {
		t.Fatal("subcommand --help produced empty output")
	}
	for i := 0; i < 10; i++ {
		if got := getHelp("echo_cmd", "--help"); got != firstSub {
			t.Fatalf("subcommand --help pass %d differs from pass 0 (non-deterministic)\n--- pass 0 ---\n%s\n--- pass %d ---\n%s", i, firstSub, i, got)
		}
	}
}

// TestCommandsListedInDeclarationOrder verifies T11: commands appear in the
// help output in the order they were declared in the descriptor (deterministic
// by construction via ordered slices), not in an arbitrary or map-iteration
// order. The t8Desc declares EchoCmd before PingCmd, so EchoCmd must appear
// before PingCmd in the root --help output.
func TestCommandsListedInDeclarationOrder(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	out := stderr.String() + stdout.String()

	echoIdx := indexOf(out, "echo_cmd")
	pingIdx := indexOf(out, "ping_cmd")
	if echoIdx < 0 {
		t.Fatal("echo_cmd not found in help output")
	}
	if pingIdx < 0 {
		t.Fatal("ping_cmd not found in help output")
	}
	if echoIdx > pingIdx {
		t.Errorf("echo_cmd (idx %d) should appear before ping_cmd (idx %d) in help — declaration order", echoIdx, pingIdx)
	}
}

// TestFlagsListedInHelp verifies T11: a command's flags are listed in the
// subcommand help output. ugo's flag.FlagSet.VisitAll iterates flags in lexical
// order (a stdlib guarantee, deterministic across runs); the descriptor's
// fnum order is preserved in codegen emission (T3) and in the binding
// (bindFlags walks Fields in slice order). The T11 guarantee is byte-stability
// across runs, not that help matches fnum order.
func TestFlagsListedInHelp(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"echo_cmd", "--help"}, &stdout, &stderr)
	out := stderr.String() + stdout.String()

	// Both flags must appear in the subcommand help.
	if indexOf(out, "-in") < 0 {
		t.Error("--in flag not found in subcommand help")
	}
	if indexOf(out, "-count") < 0 {
		t.Error("--count flag not found in subcommand help")
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
