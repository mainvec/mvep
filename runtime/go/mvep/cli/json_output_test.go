package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestMvepOutputJSONResult verifies T5: --mvep-output json renders the result
// as parseable JSON on stdout.
func TestMvepOutputJSONResult(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--mvep-output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; got: %s", err, stdout.String())
	}
	if got["Out"] != "echoed" {
		t.Errorf("Out = %v, want echoed", got["Out"])
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty in JSON mode; got: %s", stderr.String())
	}
}

// TestMvepOutputTextMatchesDefault verifies T5: --mvep-output text matches
// today's output.
func TestMvepOutputTextMatchesDefault(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}

	// Default app (no flag).
	appDefault := New(&t8Desc, ex)
	var outDefault, _ bytes.Buffer
	if err := appDefault.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &outDefault, &bytes.Buffer{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// App with explicit --mvep-output text.
	appText := New(&t8Desc, &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}})
	var outText, _ bytes.Buffer
	if err := appText.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--mvep-output", "text"}, &outText, &bytes.Buffer{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outText.String() != outDefault.String() {
		t.Errorf("text output differs from default:\n default: %q\n text:    %q", outDefault.String(), outText.String())
	}
}

// TestMvepOutputJSONError verifies T5: a failing command under --mvep-output
// json emits {"error":...} on stdout, nothing on stderr, and still returns the
// error.
func TestMvepOutputJSONError(t *testing.T) {
	// Command with a required field that will be missing.
	type reqCmd struct{ Name string }
	desc := mvepDesc("errpkg", "ReqCmd", func() any { return &reqCmd{} }, []mvep.FieldDesc{
		{Name: "name", Fnum: 1, Type: mvep.FieldString, Required: true, Ptr: func(c any) any { return &c.(*reqCmd).Name }},
	})
	app := New(&desc, &recordingExecutor{})

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"req_cmd", "--mvep-output", "json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if jerr := json.Unmarshal(stdout.Bytes(), &got); jerr != nil {
		t.Fatalf("stdout is not valid JSON: %v; got: %s", jerr, stdout.String())
	}
	if got.Error.Message == "" {
		t.Error("error message should be present")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty in JSON mode; got: %s", stderr.String())
	}
}

// TestMvepOutputJSONFlagParseError verifies T5: a flag-parse failure combined
// with --mvep-output json still produces JSON (exercises the arg pre-scan).
func TestMvepOutputJSONFlagParseError(t *testing.T) {
	app := New(&t8Desc, &recordingExecutor{})

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--bogus-flag", "x", "--mvep-output", "json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected flag-parse error, got nil")
	}
	var got map[string]any
	if jerr := json.Unmarshal(stdout.Bytes(), &got); jerr != nil {
		t.Fatalf("stdout is not valid JSON after flag-parse error: %v; got: %s", jerr, stdout.String())
	}
	if _, ok := got["error"]; !ok {
		t.Errorf("expected an error object; got: %s", stdout.String())
	}
}

// TestMvepOutputNamespaceRenamed verifies T5: renaming the namespace renames
// the output flag.
func TestMvepOutputNamespaceRenamed(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex, WithNamespace("acme"))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--acme-output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; got: %s", err, stdout.String())
	}
}

// TestMvepOutputDoesNotCollideWithOutput verifies T5: the implementor's own
// --output flag coexists with --mvep-output (renderer_test.go registers one).
func TestMvepOutputDoesNotCollideWithOutput(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex)
	// Implementor adds its own --output persistent flag (as renderer_test does).
	var output string
	app.Root().PersistentFlags().StringVar(&output, "output", "text", "implementor flag")

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Implementor's --output didn't trip over the namespace's flag; result still
	// renders via the default text renderer.
	if !strings.Contains(stdout.String(), "echoed") {
		t.Errorf("expected echoed in output; got: %s", stdout.String())
	}
}
