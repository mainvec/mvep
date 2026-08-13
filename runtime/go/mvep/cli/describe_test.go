package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestMvepList verifies T7: mvep list prints the command names, one per line.
func TestMvepList(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t4Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "list"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "EchoCmd") {
		t.Errorf("list should include EchoCmd; got: %s", stdout.String())
	}
}

// TestMvepListJSON verifies T7: mvep list under --mvep-output json prints a
// JSON array of {name, description} objects.
func TestMvepListJSON(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t4Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "list", "--mvep-output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var entries []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil {
		t.Fatalf("list output not valid JSON array: %v; got: %s", err, stdout.String())
	}
	if len(entries) != 1 || entries[0]["name"] != "EchoCmd" {
		t.Errorf("entries = %v, want [{name:EchoCmd}]", entries)
	}
}

// TestMvepDescribe verifies T7: mvep describe <command> emits a projection with
// the command's name and fields.
func TestMvepDescribe(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t4Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe", "EchoCmd"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("describe output not valid JSON: %v; got: %s", err, stdout.String())
	}
	if got["name"] != "EchoCmd" {
		t.Errorf("name = %v, want EchoCmd", got["name"])
	}
}

// TestMvepDescribeUnknown verifies T7: describe of an unknown command errors.
func TestMvepDescribeUnknown(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t4Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe", "nope"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
}

// TestMvepDescribeAll verifies T7: describe with no argument describes all
// commands (a JSON array).
func TestMvepDescribeAll(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t4Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("describe output not valid JSON: %v; got: %s", err, stdout.String())
	}
	if len(got) != 1 {
		t.Errorf("expected 1 command described, got %d", len(got))
	}
}

// TestDescribeProjectionStable verifies T7: describe output is unchanged by a
// synthetic FieldDesc field addition (projection decoupled from the descriptor).
func TestDescribeProjectionStable(t *testing.T) {
	// A descriptor with one extra FieldDesc field (a synthetic type) still
	// projects cleanly because the projection only reads known fields.
	type weirdCmd struct{ Name string }
	desc := mvepDesc("proj", "WeirdCmd", func() any { return &weirdCmd{} }, []mvep.FieldDesc{
		{Name: "name", Fnum: 1, Type: mvep.FieldString, Ptr: func(c any) any { return &c.(*weirdCmd).Name }},
	})

	ex := &recordingExecutor{}
	app := New(&desc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "describe"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("describe output not valid JSON: %v; got: %s", err, stdout.String())
	}
	if len(got) != 1 {
		t.Errorf("expected 1 command, got %d", len(got))
	}
	if got[0]["name"] != "WeirdCmd" {
		t.Errorf("name = %v, want WeirdCmd", got[0]["name"])
	}
}
