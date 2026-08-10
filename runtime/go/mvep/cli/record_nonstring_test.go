package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestRecordFlatteningNonStringFields verifies #36: a $ref record with
// non-string sub-fields (uint32, bool) binds correctly via depth-1 flattening.
// The bug was that all sub-field values were encoded as JSON strings in a
// map[string]string, so {"port":"8080"} failed to unmarshal into uint32.
func TestRecordFlatteningNonStringFields(t *testing.T) {
	type addrWithPort struct {
		Street string `json:"street"`
		Port   uint32 `json:"port"`
		Active bool   `json:"active"`
	}
	type cmd struct {
		Addr *addrWithPort `json:"addr"`
	}

	desc := mvep.PackageDesc{
		Name: "nonstr",
		Commands: []mvep.CommandDesc{
			{
				Name: "NonStrCmd",
				New:  func() any { return &cmd{} },
				Fields: []mvep.FieldDesc{
					{Name: "addr", Fnum: 1, Type: mvep.FieldRecord,
						Ptr: func(c any) any { return &c.(*cmd).Addr },
						Ref: &mvep.RecordDesc{Name: "AddrWithPort"},
					},
				},
				Result: &mvep.ResultDesc{Name: "NonStrCmdResult", New: func() any { return &struct{}{} }},
			},
		},
		Records: []mvep.RecordDesc{
			{Name: "AddrWithPort", Fields: []mvep.FieldDesc{
				{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*addrWithPort).Street }},
				{Name: "port", Fnum: 2, Type: mvep.FieldUint32, Ptr: func(r any) any { return &r.(*addrWithPort).Port }},
				{Name: "active", Fnum: 3, Type: mvep.FieldBool, Ptr: func(r any) any { return &r.(*addrWithPort).Active }},
			}},
		},
	}

	ex := &recordingExecutor{result: &struct{}{}}
	app := New(&desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(),
		[]string{"non_str_cmd", "--addr-street", "Main St", "--addr-port", "8080", "--addr-active"},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := ex.gotCmd.(*cmd)
	if c.Addr == nil {
		t.Fatal("Addr is nil")
	}
	if c.Addr.Street != "Main St" {
		t.Errorf("Street = %q, want %q", c.Addr.Street, "Main St")
	}
	if c.Addr.Port != 8080 {
		t.Errorf("Port = %d, want 8080", c.Addr.Port)
	}
	if !c.Addr.Active {
		t.Error("Active = false, want true")
	}
}
