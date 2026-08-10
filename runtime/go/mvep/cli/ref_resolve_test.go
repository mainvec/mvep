package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestRecordFlatteningFromGeneratedDescriptor verifies #28: when a descriptor
// emits Ref name-only (as codegen does — Ref.Fields is empty, the full fields
// live in PackageDesc.Records), the CLI resolves the record via
// PackageDesc.Record(name) and registers --<name>-<field> flags instead of
// falling back to --<name>-json.
//
// This test uses a descriptor shape matching what codegen actually emits (Ref
// name-only, fields in Records), not the hand-written Ref-with-Fields shape
// that TestFlagBindingRecord uses (which is unreachable from generated code).
func TestRecordFlatteningFromGeneratedDescriptor(t *testing.T) {
	// This descriptor mirrors the codegen output for a recRef field: Ref
	// carries only the record name; the full fields live in Records.
	type genAddress struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type genCmd struct {
		Addr *genAddress `json:"addr"`
	}

	desc := mvep.PackageDesc{
		Name: "gentest",
		Commands: []mvep.CommandDesc{
			{
				Name: "GenCmd",
				New:  func() any { return &genCmd{} },
				Fields: []mvep.FieldDesc{
					{
						Name: "addr", Fnum: 1, Type: mvep.FieldRecord,
						Ptr: func(c any) any { return &c.(*genCmd).Addr },
						// Name-only Ref — this is what codegen emits.
						Ref: &mvep.RecordDesc{Name: "Address"},
					},
				},
				Result: &mvep.ResultDesc{Name: "GenCmdResult", New: func() any { return &struct{}{} }},
			},
		},
		// The full record fields live here, resolvable via Record("Address").
		Records: []mvep.RecordDesc{
			{
				Name: "Address",
				Fields: []mvep.FieldDesc{
					{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*genAddress).Street }},
					{Name: "city", Fnum: 2, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*genAddress).City }},
				},
			},
		},
	}

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	// --addr-street and --addr-city must exist (depth-1 flattening from Records).
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"gen_cmd", "--addr-street", "123 Main St", "--addr-city", "Springfield"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*genCmd)
	if cmd.Addr == nil {
		t.Fatal("Addr is nil, want constructed record from --addr-* flags")
	}
	if cmd.Addr.Street != "123 Main St" {
		t.Errorf("Street = %q, want %q", cmd.Addr.Street, "123 Main St")
	}
	if cmd.Addr.City != "Springfield" {
		t.Errorf("City = %q, want %q", cmd.Addr.City, "Springfield")
	}
}

// TestRecordFlatteningHelpShowsFields verifies #28: the --help output for a
// command with a name-only Ref lists --<name>-<field> flags, not --<name>-json.
func TestRecordFlatteningHelpShowsFields(t *testing.T) {
	type genAddress struct {
		Street string `json:"street"`
	}
	type genCmd struct {
		Addr *genAddress `json:"addr"`
	}

	desc := mvep.PackageDesc{
		Name: "gentest",
		Commands: []mvep.CommandDesc{
			{
				Name: "GenCmd",
				New:  func() any { return &genCmd{} },
				Fields: []mvep.FieldDesc{
					{
						Name: "addr", Fnum: 1, Type: mvep.FieldRecord,
						Ptr: func(c any) any { return &c.(*genCmd).Addr },
						Ref: &mvep.RecordDesc{Name: "Address"},
					},
				},
				Result: &mvep.ResultDesc{Name: "GenCmdResult", New: func() any { return &struct{}{} }},
			},
		},
		Records: []mvep.RecordDesc{
			{Name: "Address", Fields: []mvep.FieldDesc{
				{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*genAddress).Street }},
			}},
		},
	}

	ex := &recordingExecutor{}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"gen_cmd", "--help"}, &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "-addr-street") {
		t.Errorf("help should list -addr-street flag (depth-1 flattening from Records); got:\n%s", out)
	}
}
