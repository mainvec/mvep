package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// t10ReqCmd has one required and one optional string field.
type t10ReqCmd struct {
	Required string
	Optional string
}

var t10ReqDesc = mvep.PackageDesc{
	Name: "t10pkg",
	Commands: []mvep.CommandDesc{
		{
			Name: "ReqCmd",
			New:  func() any { return &t10ReqCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "required", Fnum: 1, Type: mvep.FieldString, Required: true,
					Ptr: func(c any) any { return &c.(*t10ReqCmd).Required }},
				{Name: "optional", Fnum: 2, Type: mvep.FieldString,
					Ptr: func(c any) any { return &c.(*t10ReqCmd).Optional }},
			},
			Result: &mvep.ResultDesc{Name: "ReqCmdResult", New: func() any { return &struct{}{} }},
		},
	},
}

// TestRequiredFlagMissing verifies T10: omitting a required flag produces an
// error that names the flag, so the caller can exit with code 2.
func TestRequiredFlagMissing(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t10ReqDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"req_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing required flag, got nil")
	}
	// The error must name the missing flag.
	if !strings.Contains(strings.ToLower(err.Error()), "required") {
		t.Errorf("error should mention the required flag; got: %v", err)
	}
	// The executor must NOT have been called.
	if ex.gotCmd != nil {
		t.Error("executor should not be called when a required flag is missing")
	}
}

// TestRequiredFlagProvided verifies T10: providing the required flag succeeds.
func TestRequiredFlagProvided(t *testing.T) {
	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&t10ReqDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"req_cmd", "--required", "value"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := ex.gotCmd.(*t10ReqCmd)
	if !ok {
		t.Fatalf("executor received %T, want *t10ReqCmd", ex.gotCmd)
	}
	if cmd.Required != "value" {
		t.Errorf("Required = %q, want %q", cmd.Required, "value")
	}
}

// TestRequiredFlagInt32 verifies T10: a non-string required field (int32) is
// enforced — zero value after parsing is treated as missing.
func TestRequiredFlagInt32(t *testing.T) {
	type intReqCmd struct {
		Count int32
	}
	desc := mvep.PackageDesc{
		Name: "t10int",
		Commands: []mvep.CommandDesc{
			{
				Name: "IntReqCmd",
				New:  func() any { return &intReqCmd{} },
				Fields: []mvep.FieldDesc{
					{Name: "count", Fnum: 1, Type: mvep.FieldInt32, Required: true,
						Ptr: func(c any) any { return &c.(*intReqCmd).Count }},
				},
				Result: &mvep.ResultDesc{Name: "IntReqCmdResult", New: func() any { return &struct{}{} }},
			},
		},
	}

	t.Run("missing", func(t *testing.T) {
		ex := &recordingExecutor{}
		app := New(&desc, ex)
		var stdout, stderr bytes.Buffer
		err := app.RunWithIO(context.Background(), []string{"int_req_cmd"}, &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for missing required int32 flag")
		}
		if ex.gotCmd != nil {
			t.Error("executor should not be called")
		}
	})

	t.Run("provided", func(t *testing.T) {
		ex := &recordingExecutor{result: &struct{}{}}
		app := New(&desc, ex)
		var stdout, stderr bytes.Buffer
		err := app.RunWithIO(context.Background(), []string{"int_req_cmd", "--count", "5"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.gotCmd == nil {
			t.Fatal("executor should have been called")
		}
	})
}
