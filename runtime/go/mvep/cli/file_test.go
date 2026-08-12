package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// writeFile writes content to path in t.TempDir and returns the path.
func tempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestTopLevelMapFile verifies T8: --<name>-file loads a map field from a JSON
// file.
func TestTopLevelMapFile(t *testing.T) {
	path := tempFile(t, `{"key":"value","foo":"bar"}`)
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--headers-file", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if len(cmd.Headers) != 2 || cmd.Headers["key"] != "value" || cmd.Headers["foo"] != "bar" {
		t.Errorf("Headers = %v, want {key:value foo:bar}", cmd.Headers)
	}
}

// TestTopLevelRepeatedNonStringFile verifies T8: --<name>-file loads a repeated
// non-string field from a JSON array file.
func TestTopLevelRepeatedNonStringFile(t *testing.T) {
	type repCmd struct {
		Counts []int32 `json:"counts"`
	}
	desc := mvepDesc("repfile", "RepCmd", func() any { return &repCmd{} }, []mvep.FieldDesc{
		{Name: "counts", Fnum: 1, Type: mvep.FieldInt32, Repeated: true,
			Ptr: func(c any) any { return &c.(*repCmd).Counts }},
	})
	path := tempFile(t, `[1,2,3]`)
	ex := &recordingExecutor{}
	app := New(&desc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"rep_cmd", "--counts-file", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*repCmd)
	if len(cmd.Counts) != 3 || cmd.Counts[0] != 1 || cmd.Counts[2] != 3 {
		t.Errorf("Counts = %v, want [1 2 3]", cmd.Counts)
	}
}

// TestSubFieldRepeatedNonStringFile verifies T8: --<record>-<field>-file loads a
// repeated non-string sub-field from a JSON array file.
func TestSubFieldRepeatedNonStringFile(t *testing.T) {
	path := tempFile(t, `[1,2,3]`)
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--addr-scores-file", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil || len(cmd.Addr.Scores) != 3 || cmd.Addr.Scores[0] != 1 || cmd.Addr.Scores[2] != 3 {
		t.Errorf("Addr.Scores = %v, want [1 2 3]", cmd.Addr.Scores)
	}
}

// TestFileSupersedesJSON verifies T8: -file supersedes -json when both are
// given (within-field precedence -file > -json).
func TestFileSupersedesJSON(t *testing.T) {
	path := tempFile(t, `[9,8,7]`)
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--addr-scores-json", "[1,2]", "--addr-scores-file", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil || len(cmd.Addr.Scores) != 3 || cmd.Addr.Scores[0] != 9 {
		t.Errorf("Addr.Scores = %v, want [9 8 7] (file supersedes json)", cmd.Addr.Scores)
	}
}

// TestSubFieldRecRefFile verifies T8: a repeated recRef sub-field loads from
// --<record>-<field>-file.
func TestSubFieldRecRefFile(t *testing.T) {
	path := tempFile(t, `[{"street":"1 A St","city":"X"},{"street":"2 B St","city":"Y"}]`)
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)
	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"all_types_cmd", "--addr-refs-file", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := ex.gotCmd.(*t9AllTypesCmd)
	if cmd.Addr == nil || len(cmd.Addr.Refs) != 2 {
		t.Fatalf("Addr.Refs length = %d, want 2", len(cmd.Addr.Refs))
	}
	if cmd.Addr.Refs[0].Street != "1 A St" || cmd.Addr.Refs[1].Street != "2 B St" {
		t.Errorf("Addr.Refs = %+v, want two addresses", cmd.Addr.Refs)
	}
}
