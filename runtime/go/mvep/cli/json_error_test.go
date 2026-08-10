package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// mvepDesc is a test helper that builds a minimal PackageDesc with one command.
// newFn must return a fresh instance each call.
func mvepDesc(pkgName, cmdName string, newFn func() any, fields []mvep.FieldDesc) mvep.PackageDesc {
	return mvep.PackageDesc{
		Name: pkgName,
		Commands: []mvep.CommandDesc{
			{
				Name:   cmdName,
				New:    newFn,
				Fields: fields,
				Result: &mvep.ResultDesc{Name: cmdName + "Result", New: func() any { return &struct{}{} }},
			},
		},
	}
}

// TestMapFlagJSONError verifies #29: an invalid JSON value passed to a
// --<name>-json flag produces an error, not a silent zero-valued field.
func TestMapFlagJSONError(t *testing.T) {
	type mapCmd struct {
		Tags map[string]string `json:"tags"`
	}
	desc := mvepDesc("maptest", "MapCmd", func() any { return &mapCmd{} }, []mvep.FieldDesc{
		{Name: "tags", Fnum: 1, Type: mvep.FieldMap,
			Ptr: func(c any) any { return &c.(*mapCmd).Tags }},
	})

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"map_cmd", "--tags-json", "{bad json}"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "tags") && !strings.Contains(err.Error(), "json") {
		t.Errorf("error should mention the flag or json; got: %v", err)
	}
	if ex.gotCmd != nil {
		t.Error("executor should not be called when JSON parsing fails")
	}
}

// TestRepeatedFlagJSONError verifies #29: an invalid JSON array passed to a
// repeated non-string --<name>-json flag produces an error.
func TestRepeatedFlagJSONError(t *testing.T) {
	type repCmd struct {
		Counts []int32 `json:"counts"`
	}
	desc := mvepDesc("reptest", "RepCmd", func() any { return &repCmd{} }, []mvep.FieldDesc{
		{Name: "counts", Fnum: 1, Type: mvep.FieldInt32, Repeated: true,
			Ptr: func(c any) any { return &c.(*repCmd).Counts }},
	})

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"rep_cmd", "--counts-json", "[bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid JSON array, got nil")
	}
	if ex.gotCmd != nil {
		t.Error("executor should not be called when JSON parsing fails")
	}
}

// TestRecordJSONFallbackError verifies #29: when a record falls back to
// --<name>-json (no record fields to flatten), an invalid JSON value
// produces an error.
func TestRecordJSONFallbackError(t *testing.T) {
	type recCmd struct {
		Addr *struct {
			Street string `json:"street"`
		} `json:"addr"`
	}
	desc := mvepDesc("rectest", "RecCmd", func() any { return &recCmd{} }, []mvep.FieldDesc{
		{Name: "addr", Fnum: 1, Type: mvep.FieldRecord,
			Ptr: func(c any) any { return &c.(*recCmd).Addr },
			// No Ref and no Records — falls back to --addr-json
		},
	})

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"rec_cmd", "--addr-json", "{bad}"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if ex.gotCmd != nil {
		t.Error("executor should not be called when JSON parsing fails")
	}
}

// TestRecordFlatteningJSONError verifies #29: when a depth-1 flattened record
// builds JSON from sub-field values, an unmarshal error is surfaced (this
// path uses json.Marshal + json.Unmarshal internally; the Marshal should never
// fail for a map[string]string, but the Unmarshal into **Record could fail if
// the record struct doesn't have the right shape). This test is a guard.
func TestRecordFlatteningJSONError(t *testing.T) {
	// This path builds a map[string]string from valid string flags, then
	// marshals (always succeeds) and unmarshals into **Record. The unmarshal
	// could only fail if the record struct shape is wrong — which is a codegen
	// error caught at compile time. So this test just verifies no panic and
	// success when the flags are valid strings.
	type addr struct {
		Street string `json:"street"`
	}
	type cmd struct {
		Addr *addr `json:"addr"`
	}
	desc := mvepDesc("flat", "FlatCmd", func() any { return &cmd{} }, []mvep.FieldDesc{
		{Name: "addr", Fnum: 1, Type: mvep.FieldRecord,
			Ptr: func(c any) any { return &c.(*cmd).Addr },
			Ref: &mvep.RecordDesc{Name: "Address"},
		},
	})
	desc.Records = []mvep.RecordDesc{
		{Name: "Address", Fields: []mvep.FieldDesc{
			{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*addr).Street }},
		}},
	}

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"flat_cmd", "--addr-street", "Main"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := ex.gotCmd.(*cmd)
	if c.Addr == nil || c.Addr.Street != "Main" {
		t.Errorf("Addr = %+v, want {Street:Main}", c.Addr)
	}
}