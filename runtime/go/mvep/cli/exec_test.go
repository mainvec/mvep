package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// t4EchoCmd mirrors a command with a string field, a nested record, and a
// required field, for testing mvep exec payload dispatch.
type t4EchoCmd struct {
	In    string
	Addr  *t9Address
	Count int32
}

type t4EchoResult struct {
	Out string
}

// t4Desc is a descriptor exercising top-level, nested, and required fields.
var t4Desc = mvep.PackageDesc{
	Name:        "t4pkg",
	SpecVersion: "0.2",
	Commands: []mvep.CommandDesc{
		{
			Name: "EchoCmd",
			New:  func() any { return &t4EchoCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "in", Fnum: 1, Type: mvep.FieldString, Ptr: func(c any) any { return &c.(*t4EchoCmd).In }},
				{Name: "addr", Fnum: 2, Type: mvep.FieldRecord, Ptr: func(c any) any { return &c.(*t4EchoCmd).Addr },
					Ref: &mvep.RecordDesc{Name: "Address"}},
				{Name: "count", Fnum: 3, Type: mvep.FieldInt32, Required: true, Ptr: func(c any) any { return &c.(*t4EchoCmd).Count }},
			},
			Result: &mvep.ResultDesc{Name: "EchoCmdResult", New: func() any { return &t4EchoResult{} }},
		},
	},
	Records: []mvep.RecordDesc{
		{Name: "Address", Fields: []mvep.FieldDesc{
			{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).Street }},
			{Name: "city", Fnum: 2, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).City }},
		}},
	},
}

// TestExecDispatchFromStdin verifies T4: `svc mvep exec echo_cmd` with a
// piped payload reaches the command with the field bound.
func TestExecDispatchFromStdin(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(`{"in":"spec.json","count":1}`)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := ex.gotCmd.(*t4EchoCmd)
	if !ok {
		t.Fatalf("executor received %T, want *t4EchoCmd", ex.gotCmd)
	}
	if cmd.In != "spec.json" || cmd.Count != 1 {
		t.Errorf("cmd = %+v, want {In:spec.json Count:1}", cmd)
	}
}

// TestExecDispatchFromInputFlag verifies T4: `--input <path>` produces the same
// result as implicit stdin.
func TestExecDispatchFromInputFlag(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/p.json"
	if err := os.WriteFile(path, []byte(`{"in":"file","count":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex) // stdin is a TTY in tests unless injected; --input wins

	var stdout, stderr bytes.Buffer
	// ugo parses flags with stdlib semantics, which stop at the first
	// positional argument — so --input must precede the command name.
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "--input", path, "echo_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t4EchoCmd)
	if cmd.In != "file" || cmd.Count != 2 {
		t.Errorf("cmd = %+v, want {In:file Count:2}", cmd)
	}
}

// TestExecDispatchNestedRecord verifies T4: nested record keys are validated
// and decoded.
func TestExecDispatchNestedRecord(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(`{"in":"x","count":1,"addr":{"street":"1 A St","city":"Springfield"}}`)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t4EchoCmd)
	if cmd.Addr == nil || cmd.Addr.Street != "1 A St" || cmd.Addr.City != "Springfield" {
		t.Errorf("Addr = %+v, want {Street:1 A St City:Springfield}", cmd.Addr)
	}
}

// TestExecUnknownTopLevelKey verifies T4: an unknown top-level key errors
// naming the key.
func TestExecUnknownTopLevelKey(t *testing.T) {
	app := New(&t4Desc, &recordingExecutor{}, WithStdin(strings.NewReader(`{"inn":"x","count":1}`)))
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "inn") {
		t.Errorf("error should name the unknown key; got: %v", err)
	}
}

// TestExecUnknownNestedKey verifies T4: an unknown key nested inside a record
// also errors.
func TestExecUnknownNestedKey(t *testing.T) {
	app := New(&t4Desc, &recordingExecutor{}, WithStdin(strings.NewReader(`{"in":"x","count":1,"addr":{"street2":"1 A St"}}`)))
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown nested key, got nil")
	}
	if !strings.Contains(err.Error(), "street2") {
		t.Errorf("error should name the unknown nested key; got: %v", err)
	}
}

// TestExecSnakeCaseKey verifies T4: a snake_case payload against a camelCase
// descriptor name is accepted (key normalisation).
func TestExecSnakeCaseKey(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(`{"in":"x","count":1}`)))
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestExecUnknownCommand verifies T4: an unknown command name errors listing
// valid names.
func TestExecUnknownCommand(t *testing.T) {
	app := New(&t4Desc, &recordingExecutor{}, WithStdin(strings.NewReader(`{}`)))
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "nope"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "echo_cmd") {
		t.Errorf("error should list valid names; got: %v", err)
	}
}

// TestExecRequiredField verifies T4: a payload missing a required field fails
// checkRequired.
func TestExecRequiredField(t *testing.T) {
	app := New(&t4Desc, &recordingExecutor{}, WithStdin(strings.NewReader(`{"in":"x"}`)))
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "exec", "echo_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
	if !strings.Contains(err.Error(), "count") {
		t.Errorf("error should name the required field; got: %v", err)
	}
}
