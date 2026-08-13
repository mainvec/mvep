package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// Name-identity tests (issue #60).
//
// The mvep machine surface (exec/send/describe/list) keys commands by the
// descriptor Name, which is unique and wire-consistent with the server. The
// gDesc fixture defines grouped commands (StartServerCmd, CreateKeyCmd) plus a
// root command (RootCmd). Their names must be emitted and resolved
// unambiguously — bare aliases that repeat across groups must not collide.

// TestMvepListNames verifies mvep list emits the descriptor Name and the
// command description for grouped and root commands.
func TestMvepListNames(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "list"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"StartServerCmd", "CreateKeyCmd", "SecretCmd", "RootCmd"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("list should contain %q; got:\n%s", want, stdout.String())
		}
	}
	// The description should be present too (StartServerCmd's desc).
	if !strings.Contains(stdout.String(), "Start an LLM server") {
		t.Errorf("list should include the command description; got:\n%s", stdout.String())
	}
}

// TestMvepListJSONNames verifies mvep list --mvep-output json emits the Name and
// description as {name, description} objects.
func TestMvepListJSONNames(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "list", "--mvep-output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var entries []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil {
		t.Fatalf("list output not valid JSON array: %v; got: %s", err, stdout.String())
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e["name"]
	}
	joined := strings.Join(names, " ")
	for _, want := range []string{"StartServerCmd", "CreateKeyCmd", "SecretCmd", "RootCmd"} {
		if !strings.Contains(joined, want) {
			t.Errorf("names %v should contain %q", names, want)
		}
	}
}

// TestMvepDescribeName verifies mvep describe resolves the descriptor Name.
func TestMvepDescribeName(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe", "StartServerCmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("describe output not valid JSON: %v; got: %s", err, stdout.String())
	}
	if got["name"] != "StartServerCmd" {
		t.Errorf("name = %v, want StartServerCmd", got["name"])
	}
}

// TestMvepDescribeUnknownName verifies describe of an unknown name errors.
func TestMvepDescribeUnknownName(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe", "start"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for bare alias start, got nil")
	}
	if !strings.Contains(err.Error(), "start") {
		t.Errorf("error should name the unknown name; got: %v", err)
	}
}

// TestMvepExecName verifies mvep exec resolves the descriptor Name and reaches
// the command.
func TestMvepExecName(t *testing.T) {
	ex := &recordingExecutor{result: &gServerCmdResult{PID: 42}}
	app := New(&gDesc, ex, WithStdin(strings.NewReader(`{"model":"llama3"}`)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "StartServerCmd"}, &stdout, &stderr)
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

// TestMvepSendName verifies mvep send resolves a Cmd.Cmd carrying the
// descriptor Name (wire-compatible with the server).
func TestMvepSendName(t *testing.T) {
	ex := &recordingExecutor{result: &gKeyCmdResult{Key: "k"}}
	app := New(&gDesc, ex, WithStdin(strings.NewReader(
		sendReqJSON("CreateKeyCmd", map[string]any{"name": "prod"})+"\n",
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send"}, &stdout, &stderr)
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

// TestMvepListNoAliasCollision verifies two commands with the same bare alias in
// different groups do not collide: both Names appear and exec resolves either.
func TestMvepListNoAliasCollision(t *testing.T) {
	amb := mvep.PackageDesc{
		Name: "amb",
		Commands: []mvep.CommandDesc{
			{Name: "ListReposCmd", Alias: "list", Group: "repo", New: func() any { return &gRootCmd{} },
				Result: &mvep.ResultDesc{Name: "ListReposCmdResult", New: func() any { return &gRootCmdResult{} }}},
			{Name: "ListServersCmd", Alias: "list", Group: "server", New: func() any { return &gRootCmd{} },
				Result: &mvep.ResultDesc{Name: "ListServersCmdResult", New: func() any { return &gRootCmdResult{} }}},
		},
		Groups: []mvep.GroupDesc{
			{Path: "repo", Name: "repo"},
			{Path: "server", Name: "server"},
		},
	}
	ex := &recordingExecutor{}
	app := New(&amb, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "list"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"ListReposCmd", "ListServersCmd"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("list should contain both %q; got:\n%s", want, stdout.String())
		}
	}
	// Executing either Name resolves unambiguously.
	ex2 := &recordingExecutor{}
	app2 := New(&amb, ex2, WithStdin(strings.NewReader(`{}`)))
	var out, _ bytes.Buffer
	if err := app2.RunWithIO(context.Background(), []string{"mvep", "exec", "ListServersCmd"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("exec ListServersCmd unexpected error: %v", err)
	}
}
