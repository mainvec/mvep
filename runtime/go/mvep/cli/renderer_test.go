package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestExitCodeSuccess verifies T14: a successful command returns exit code 0.
func TestExitCodeSuccess(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "hi"}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ExitCode(err); got != 0 {
		t.Errorf("ExitCode = %d, want 0", got)
	}
}

// TestExitCodeMissingRequiredFlag verifies T14: a missing required flag yields
// exit code 2 (usage error).
func TestExitCodeMissingRequiredFlag(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t10ReqDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"req_cmd"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing required flag")
	}
	if got := ExitCode(err); got != 2 {
		t.Errorf("ExitCode = %d, want 2 (usage)", got)
	}
}

// TestExitCodeExecutionError verifies T14: a generic execution error yields
// exit code 1.
func TestExitCodeExecutionError(t *testing.T) {
	ex := &recordingExecutor{err: errors.New("something broke")}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected execution error")
	}
	if got := ExitCode(err); got != 1 {
		t.Errorf("ExitCode = %d, want 1 (other)", got)
	}
}

// TestExitCodeUnknownCommandRemote verifies T14: an ErrorCode with http_404
// (unknown command from remote) yields exit code 3 (not-found).
func TestExitCodeUnknownCommandRemote(t *testing.T) {
	err := &ErrorCode{Code: "http_404", Message: "unknown command"}
	if got := ExitCode(err); got != 3 {
		t.Errorf("ExitCode = %d, want 3 (not-found)", got)
	}
}

// TestExitCodeAuthError verifies T14: an ErrorCode with http_401 or http_403
// yields exit code 4 (auth).
func TestExitCodeAuthError(t *testing.T) {
	for _, code := range []string{"http_401", "http_403"} {
		err := &ErrorCode{Code: code, Message: "auth failed"}
		if got := ExitCode(err); got != 4 {
			t.Errorf("ExitCode(%s) = %d, want 4 (auth)", code, got)
		}
	}
}

// TestExitCodeServerError verifies T14: an ErrorCode with http_500 yields
// exit code 1 (other execution error).
func TestExitCodeServerError(t *testing.T) {
	err := &ErrorCode{Code: "http_500", Message: "command failed"}
	if got := ExitCode(err); got != 1 {
		t.Errorf("ExitCode = %d, want 1 (other)", got)
	}
}

// TestDefaultRenderer verifies T14: the default (text) renderer prints the
// result to stdout.
func TestDefaultRenderer(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "echoed") {
		t.Errorf("stdout should contain the result; got: %s", stdout.String())
	}
}

// TestJSONRenderer verifies T14: --output=json renders the result as JSON.
// The --output flag is a persistent flag the implementor adds; the App
// supports a JSON renderer via SetRenderer.
func TestJSONRenderer(t *testing.T) {
	ex := &recordingExecutor{result: &t8EchoCmdResult{Out: "echoed"}}
	app := New(&t8Desc, ex)

	// The implementor adds a --output persistent flag and wires it to a
	// renderer that switches on the flag value.
	var output string
	app.Root().PersistentFlags().StringVar(&output, "output", "text", "output format (text|json)")
	app.SetRenderer(func(w io.Writer, result any) {
		switch output {
		case "json":
			b, _ := json.Marshal(result)
			w.Write(b)
		default:
			fmt.Fprintln(w, result)
		}
	})

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"echo_cmd", "--in", "hello", "--output", "json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	// JSON output should contain the result as a JSON object.
	if !strings.Contains(out, `"Out":"echoed"`) {
		t.Errorf("JSON output should contain the result as JSON; got: %s", out)
	}
}