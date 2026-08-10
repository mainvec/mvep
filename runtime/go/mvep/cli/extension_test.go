package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/ugo/cli"
)

// TestGlobalPersistentFlag verifies T12: an implementor can add a global
// --endpoint flag via App.Root().PersistentFlags(), and it is inherited by
// generated subcommands (parses successfully alongside command flags).
func TestGlobalPersistentFlag(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	// Add a global --endpoint persistent flag on the root.
	var endpoint string
	app.Root().PersistentFlags().StringVar(&endpoint, "endpoint", "localhost:8080", "server endpoint")

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--endpoint", "remote:9090"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The persistent flag must be parsed.
	if endpoint != "remote:9090" {
		t.Errorf("endpoint = %q, want %q", endpoint, "remote:9090")
	}
	// The command flag must still bind.
	cmd := ex.gotCmd.(*t8EchoCmd)
	if cmd.In != "hello" {
		t.Errorf("In = %q, want %q", cmd.In, "hello")
	}
}

// TestCustomSubcommand verifies T12: an implementor can add a custom subcommand
// via App.Root().AddCommand() without touching generated code, and it runs
// alongside the descriptor-driven commands.
func TestCustomSubcommand(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	// Add a custom subcommand.
	var customRan bool
	custom := &cli.Command{
		Usage: "custom_cmd",
		Short: "A custom subcommand added by the implementor",
		RunE: func(ctx *cli.Context, args []string) error {
			customRan = true
			return nil
		},
	}
	app.Root().AddCommand(custom)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"custom_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !customRan {
		t.Error("custom subcommand was not invoked")
	}
	// The generated executor must NOT have been called.
	if ex.gotCmd != nil {
		t.Error("generated executor should not be called for a custom subcommand")
	}
}

// TestOverrideGeneratedCommand verifies T12: an implementor can override a
// generated subcommand by adding one with the same name, and the override
// takes precedence (ugo's findSub returns the first match; since the override
// is added after the generated ones, it must be added before or replace the
// generated one). We test by adding the override via Root().AddCommand before
// the generated ones are built — but New already added them, so we test that
// an implementor can replace the generated command's RunE via Root() lookup.
func TestOverrideGeneratedCommand(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	// The implementor overrides the echo_cmd subcommand's RunE to add
	// pre-processing. We find it by name using ugo's Find.
	cmd, _ := app.Root().Find([]string{"echo_cmd"})
	if cmd == nil {
		t.Fatal("echo_cmd not found in command tree")
	}
	if cmd.Name() != "echo_cmd" {
		t.Fatalf("Find returned %q, want echo_cmd", cmd.Name())
	}

	// Override the RunE.
	var overriddenRan bool
	originalRunE := cmd.RunE
	cmd.RunE = func(ctx *cli.Context, args []string) error {
		overriddenRan = true
		// Still call the original to verify it's accessible.
		return originalRunE(ctx, args)
	}

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "test"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overriddenRan {
		t.Error("override RunE was not called")
	}
	if ex.gotCmd == nil {
		t.Error("original executor should have been called via the override chain")
	}
}

// TestCustomSubcommandAppearsInHelp verifies T12: custom subcommands appear in
// the root --help output alongside generated ones.
func TestCustomSubcommandAppearsInHelp(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t8Desc, ex)

	custom := &cli.Command{
		Usage: "extra_cmd",
		Short: "An extra command",
		RunE:  func(ctx *cli.Context, args []string) error { return nil },
	}
	app.Root().AddCommand(custom)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	out := strings.ToLower(stderr.String() + stdout.String())
	if !strings.Contains(out, "extra_cmd") {
		t.Error("custom subcommand should appear in --help")
	}
}
