package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// t9AllTypesCmd is a command struct carrying every spec field type, mirroring
// the generated AllTypesCmd from fixture 11.
type t9AllTypesCmd struct {
	Str       string
	Flag      bool
	I32       int32
	I64       int64
	U32       uint32
	S32       int32
	F32       float32
	F64       float64
	Byts      []byte
	Ts        time.Time
	Dur       time.Duration
	Repeateds []string
	Headers   map[string]string
	Addr      *t9Address
}

type t9Address struct {
	Street string
	City   string
	Tags   []string // repeated string sub-field
	Scores []int32  // repeated non-string sub-field
}

// t9AllTypesResult is the matching result.
type t9AllTypesResult struct {
	Summary string
}

// t9AllTypesDesc is a descriptor with one command carrying every scalar type
// plus repeated string, a map, and a $ref record.
var t9AllTypesDesc = mvep.PackageDesc{
	Name:        "t9pkg",
	SpecVersion: "0.2",
	Commands: []mvep.CommandDesc{
		{
			Name: "AllTypesCmd",
			New:  func() any { return &t9AllTypesCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "str", Fnum: 1, Type: mvep.FieldString, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Str }},
				{Name: "flag", Fnum: 2, Type: mvep.FieldBool, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Flag }},
				{Name: "i32", Fnum: 3, Type: mvep.FieldInt32, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).I32 }},
				{Name: "i64", Fnum: 4, Type: mvep.FieldInt64, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).I64 }},
				{Name: "u32", Fnum: 5, Type: mvep.FieldUint32, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).U32 }},
				{Name: "s32", Fnum: 6, Type: mvep.FieldSint32, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).S32 }},
				{Name: "f32", Fnum: 7, Type: mvep.FieldFloat, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).F32 }},
				{Name: "f64", Fnum: 8, Type: mvep.FieldDouble, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).F64 }},
				{Name: "byts", Fnum: 9, Type: mvep.FieldBytes, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Byts }},
				{Name: "ts", Fnum: 10, Type: mvep.FieldTimestamp, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Ts }},
				{Name: "dur", Fnum: 11, Type: mvep.FieldDuration, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Dur }},
				{Name: "repeateds", Fnum: 13, Type: mvep.FieldString, Repeated: true, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Repeateds }},
				{Name: "headers", Fnum: 14, Type: mvep.FieldMap, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Headers }},
				{Name: "addr", Fnum: 15, Type: mvep.FieldRecord, Ptr: func(c any) any { return &c.(*t9AllTypesCmd).Addr },
					Ref: &mvep.RecordDesc{Name: "Address", Fields: []mvep.FieldDesc{
						{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).Street }},
						{Name: "city", Fnum: 2, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).City }},
						{Name: "tags", Fnum: 3, Type: mvep.FieldString, Repeated: true, Ptr: func(r any) any { return &r.(*t9Address).Tags }},
						{Name: "scores", Fnum: 4, Type: mvep.FieldInt32, Repeated: true, Ptr: func(r any) any { return &r.(*t9Address).Scores }},
					}}},
			},
			Result: &mvep.ResultDesc{
				Name: "AllTypesCmdResult",
				New:  func() any { return &t9AllTypesResult{} },
			},
		},
	},
	Records: []mvep.RecordDesc{
		{Name: "Address", Fields: []mvep.FieldDesc{
			{Name: "street", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).Street }},
			{Name: "city", Fnum: 2, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*t9Address).City }},
			{Name: "tags", Fnum: 3, Type: mvep.FieldString, Repeated: true, Ptr: func(r any) any { return &r.(*t9Address).Tags }},
			{Name: "scores", Fnum: 4, Type: mvep.FieldInt32, Repeated: true, Ptr: func(r any) any { return &r.(*t9Address).Scores }},
		}},
	},
}

// TestFlagBindingAllScalarTypes verifies T9: every scalar spec type binds from
// a flag and arrives in the command struct. One subtest per type.
func TestFlagBindingAllScalarTypes(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		check func(*t9AllTypesCmd) bool
	}{
		{"string", []string{"all_types_cmd", "--str", "hello"}, func(c *t9AllTypesCmd) bool { return c.Str == "hello" }},
		{"bool", []string{"all_types_cmd", "--flag"}, func(c *t9AllTypesCmd) bool { return c.Flag == true }},
		{"int32", []string{"all_types_cmd", "--i32", "42"}, func(c *t9AllTypesCmd) bool { return c.I32 == 42 }},
		{"int64", []string{"all_types_cmd", "--i64", "9000000000"}, func(c *t9AllTypesCmd) bool { return c.I64 == 9000000000 }},
		{"uint32", []string{"all_types_cmd", "--u32", "4294967295"}, func(c *t9AllTypesCmd) bool { return c.U32 == 4294967295 }},
		{"sint32", []string{"all_types_cmd", "--s32", "-99"}, func(c *t9AllTypesCmd) bool { return c.S32 == -99 }},
		{"float32", []string{"all_types_cmd", "--f32", "3.14"}, func(c *t9AllTypesCmd) bool { return c.F32 == 3.14 }},
		{"float64", []string{"all_types_cmd", "--f64", "2.71828"}, func(c *t9AllTypesCmd) bool { return c.F64 == 2.71828 }},
		{"bytes", []string{"all_types_cmd", "--byts", "aGVsbG8="}, func(c *t9AllTypesCmd) bool { return string(c.Byts) == "hello" }},
		{"timestamp", []string{"all_types_cmd", "--ts", "2026-01-15T10:30:00Z"}, func(c *t9AllTypesCmd) bool {
			return !c.Ts.IsZero() && c.Ts.Year() == 2026
		}},
		{"duration", []string{"all_types_cmd", "--dur", "5m"}, func(c *t9AllTypesCmd) bool { return c.Dur == 5*time.Minute }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ex := &recordingExecutor{}
			app := New(&t9AllTypesDesc, ex)
			var stdout, stderr bytes.Buffer
			if err := app.RunWithIO(context.Background(), tc.args, &stdout, &stderr); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cmd, ok := ex.gotCmd.(*t9AllTypesCmd)
			if !ok {
				t.Fatalf("executor received %T, want *t9AllTypesCmd", ex.gotCmd)
			}
			if !tc.check(cmd) {
				t.Errorf("field %q not bound correctly; got %+v", tc.name, cmd)
			}
		})
	}
}

// TestFlagBindingRepeatedString verifies T9: a repeated string field
// accumulates multiple --flag occurrences into a []string.
func TestFlagBindingRepeatedString(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--repeateds", "a", "--repeateds", "b", "--repeateds", "c"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if len(cmd.Repeateds) != 3 || cmd.Repeateds[0] != "a" || cmd.Repeateds[1] != "b" || cmd.Repeateds[2] != "c" {
		t.Errorf("Repeateds = %v, want [a b c]", cmd.Repeateds)
	}
}

// TestFlagBindingMap verifies T9: a map[string]string field binds from a
// --x-json flag (JSON object) into the struct's map field.
func TestFlagBindingMap(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--headers-json", `{"key":"value","foo":"bar"}`}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if len(cmd.Headers) != 2 || cmd.Headers["key"] != "value" || cmd.Headers["foo"] != "bar" {
		t.Errorf("Headers = %v, want {key:value foo:bar}", cmd.Headers)
	}
}

// TestFlagBindingRecord verifies T9: a $ref record field flattens to depth 1,
// binding record fields as --record-field flags and constructing the nested
// struct.
func TestFlagBindingRecord(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--addr-street", "123 Main St", "--addr-city", "Springfield"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil {
		t.Fatal("Addr is nil, want constructed record")
	}
	if cmd.Addr.Street != "123 Main St" || cmd.Addr.City != "Springfield" {
		t.Errorf("Addr = %+v, want {Street:123 Main St City:Springfield}", cmd.Addr)
	}
}

// TestFlagBindingRepeatedStringSubField verifies T1: a repeated string record
// sub-field binds from repeatable --record-field flags into a []string. This is
// the zirafa argsTemplate case.
func TestFlagBindingRepeatedStringSubField(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{
		"all_types_cmd",
		"--addr-tags", "a",
		"--addr-tags", "b",
		"--addr-tags", "c",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil {
		t.Fatal("Addr is nil, want constructed record")
	}
	if len(cmd.Addr.Tags) != 3 || cmd.Addr.Tags[0] != "a" || cmd.Addr.Tags[1] != "b" || cmd.Addr.Tags[2] != "c" {
		t.Errorf("Addr.Tags = %v, want [a b c]", cmd.Addr.Tags)
	}
}

// TestFlagBindingRepeatedNonStringSubField verifies T1: a repeated non-string
// record sub-field binds from --record-field-json as a JSON array.
func TestFlagBindingRepeatedNonStringSubField(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{
		"all_types_cmd",
		"--addr-scores-json", "[1,2,3]",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil {
		t.Fatal("Addr is nil, want constructed record")
	}
	if len(cmd.Addr.Scores) != 3 || cmd.Addr.Scores[0] != 1 || cmd.Addr.Scores[1] != 2 || cmd.Addr.Scores[2] != 3 {
		t.Errorf("Addr.Scores = %v, want [1 2 3]", cmd.Addr.Scores)
	}
}

// TestFlagBindingRepeatedSubFieldJSONError verifies T1: malformed JSON passed
// to a repeated non-string --record-field-json flag errors naming the sub-field
// flag, not the parent record flag.
func TestFlagBindingRepeatedSubFieldJSONError(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{
		"all_types_cmd",
		"--addr-scores-json", "[1,2",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for malformed -json, got nil")
	}
	if !strings.Contains(err.Error(), "--addr-scores-json") {
		t.Errorf("error should name the sub-field flag; got: %v", err)
	}
}

// TestFlagBindingMixedRepeatedAndScalarSubFields verifies T1: a record with both
// repeated and scalar sub-fields set together assembles into one correct struct.
func TestFlagBindingMixedRepeatedAndScalarSubFields(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{
		"all_types_cmd",
		"--addr-street", "123 Main St",
		"--addr-city", "Springfield",
		"--addr-tags", "a",
		"--addr-tags", "b",
		"--addr-scores-json", "[5,6]",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil {
		t.Fatal("Addr is nil, want constructed record")
	}
	if cmd.Addr.Street != "123 Main St" || cmd.Addr.City != "Springfield" {
		t.Errorf("Addr scalar = %+v, want {Street:123 Main St City:Springfield}", cmd.Addr)
	}
	if len(cmd.Addr.Tags) != 2 || cmd.Addr.Tags[0] != "a" || cmd.Addr.Tags[1] != "b" {
		t.Errorf("Addr.Tags = %v, want [a b]", cmd.Addr.Tags)
	}
	if len(cmd.Addr.Scores) != 2 || cmd.Addr.Scores[0] != 5 || cmd.Addr.Scores[1] != 6 {
		t.Errorf("Addr.Scores = %v, want [5 6]", cmd.Addr.Scores)
	}
}

