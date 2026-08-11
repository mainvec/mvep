package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// Plan 040 T6/T7 fixtures ------------------------------------------------

// gServerCmd is a command under the "server" group.
type gServerCmd struct {
	Model string
}

// gServerCmdResult is the matching result.
type gServerCmdResult struct {
	PID int64
}

// gKeyCmd is a command under the "server/keys" group.
type gKeyCmd struct {
	Name string
}

// gKeyCmdResult is the matching result.
type gKeyCmdResult struct {
	Key string
}

// gRootCmd is a command at the root.
type gRootCmd struct{}

// gRootCmdResult is the matching result.
type gRootCmdResult struct {
	OK bool
}

// gDesc is a descriptor exercising groups at depth 1 and 2, an alias, and a
// hidden group, plus a root command.
var gDesc = mvep.PackageDesc{
	Name:        "gapp",
	SpecVersion: "0.2",
	Commands: []mvep.CommandDesc{
		{
			Name:  "StartServerCmd",
			Alias: "start",
			Group: "server",
			Desc:  "Start an LLM server",
			New:   func() any { return &gServerCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "model", Fnum: 1, Type: mvep.FieldString, Required: true, Ptr: func(c any) any { return &c.(*gServerCmd).Model }},
			},
			Result: &mvep.ResultDesc{Name: "StartServerCmdResult", New: func() any { return &gServerCmdResult{} }},
		},
		{
			Name:  "CreateKeyCmd",
			Alias: "create",
			Group: "server/keys",
			Desc:  "Create an API key",
			New:   func() any { return &gKeyCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "name", Fnum: 1, Type: mvep.FieldString, Ptr: func(c any) any { return &c.(*gKeyCmd).Name }},
			},
			Result: &mvep.ResultDesc{Name: "CreateKeyCmdResult", New: func() any { return &gKeyCmdResult{} }},
		},
		{
			Name:   "SecretCmd",
			Alias:  "secret",
			Group:  "hidden",
			Desc:   "A hidden command",
			New:    func() any { return &gRootCmd{} },
			Result: &mvep.ResultDesc{Name: "SecretCmdResult", New: func() any { return &gRootCmdResult{} }},
		},
		{
			Name:   "RootCmd",
			Alias:  "root",
			Desc:   "A root command",
			New:    func() any { return &gRootCmd{} },
			Result: &mvep.ResultDesc{Name: "RootCmdResult", New: func() any { return &gRootCmdResult{} }},
		},
	},
	Groups: []mvep.GroupDesc{
		{Path: "server", Name: "server", Title: "LLM Servers", Desc: "Start, stop and inspect servers"},
		{Path: "server/keys", Name: "keys", Title: "API Keys", Aliases: []string{"key"}},
		{Path: "hidden", Name: "hidden", Hidden: true},
	},
}

// TestGroupDispatchNested verifies T6: `server start --model x` reaches
// StartServerCmd with the field bound.
func TestGroupDispatchNested(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &gServerCmdResult{PID: 42}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server", "start", "--model", "llama3"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := ex.gotCmd.(*gServerCmd)
	if !ok {
		t.Fatalf("executor received %T, want *gServerCmd", ex.gotCmd)
	}
	if cmd.Model != "llama3" {
		t.Errorf("Model = %q, want %q", cmd.Model, "llama3")
	}
}

// TestGroupDispatchDepth2 verifies T6: a command at depth 2 resolves through
// both group levels.
func TestGroupDispatchDepth2(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &gKeyCmdResult{Key: "k"}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server", "keys", "create", "--name", "prod"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := ex.gotCmd.(*gKeyCmd)
	if !ok {
		t.Fatalf("executor received %T, want *gKeyCmd", ex.gotCmd)
	}
	if cmd.Name != "prod" {
		t.Errorf("Name = %q, want %q", cmd.Name, "prod")
	}
}

// TestGroupDispatchGroupAlias verifies T6: a group alias resolves the group.
func TestGroupDispatchGroupAlias(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &gKeyCmdResult{Key: "k"}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server", "key", "create"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := ex.gotCmd.(*gKeyCmd); !ok {
		t.Fatalf("executor received %T, want *gKeyCmd", ex.gotCmd)
	}
}

// TestGroupHelpPrintsGroupHelp verifies T6: `server` with no subcommand prints
// the group's help and exits 0 (no error).
func TestGroupHelpPrintsGroupHelp(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("group with no subcommand should print help, not error: %v", err)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "start") {
		t.Errorf("group help should list its subcommands; got: %s", combined)
	}
}

// TestGroupUnknownSubcommand verifies T6: `server bogus` returns an error
// naming the group path, and does not print help as if it succeeded.
func TestGroupUnknownSubcommand(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server", "bogus"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown subcommand under a group, got nil")
	}
	if !strings.Contains(err.Error(), "server") {
		t.Errorf("error should name the group path; got: %v", err)
	}
}

// TestGroupHiddenAbsentFromRootHelp verifies T6: a hidden group is absent from
// root help but still invocable.
func TestGroupHiddenAbsentFromRootHelp(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &gRootCmdResult{OK: true}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	combined := strings.ToLower(stdout.String() + stderr.String())
	if strings.Contains(combined, "hidden") {
		t.Errorf("hidden group should be absent from root help; got: %s", combined)
	}

	// Still invocable.
	stdout.Reset()
	stderr.Reset()
	err := app.RunWithIO(context.Background(), []string{"hidden", "secret"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("hidden group should still be invocable: %v", err)
	}
	if _, ok := ex.gotCmd.(*gRootCmd); !ok {
		t.Fatalf("executor received %T, want *gRootCmd", ex.gotCmd)
	}
}

// TestGroupRootCommandUnaffected verifies T6: a root command still dispatches
// when groups are present.
func TestGroupRootCommandUnaffected(t *testing.T) {
	t.Parallel()

	ex := &recordingExecutor{result: &gRootCmdResult{OK: true}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"root"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := ex.gotCmd.(*gRootCmd); !ok {
		t.Fatalf("executor received %T, want *gRootCmd", ex.gotCmd)
	}
}
