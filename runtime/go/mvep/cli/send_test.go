package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// headerExecutor records the request header it saw and sets a response header.
type headerExecutor struct {
	gotHeader string
}

func (e *headerExecutor) Run(ctx context.Context, cmd any) (any, error) {
	e.gotHeader = mvep.GetRequestHeader(ctx, "trace-id")
	mvep.SetResponseHeader(ctx, "resp-trace", "abc")
	return &t4EchoResult{Out: "ok"}, nil
}

// sendReqJSON builds a CmdReq line for a command name and raw payload object.
// CmdReq.Payload is []byte, so it serializes as base64 — the wire format that
// send consumes.
func sendReqJSON(cmd string, payloadObj any) string {
	b, _ := json.Marshal(payloadObj)
	req := mvep.NewCmdReq(cmd, b)
	out, _ := json.Marshal(req)
	return string(out)
}

// TestSendNDJSON verifies T6: NDJSON input streams produce one CmdResp per
// record.
func TestSendNDJSON(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(
		sendReqJSON("echo_cmd", map[string]any{"in": "a", "count": 1})+"\n"+
			sendReqJSON("echo_cmd", map[string]any{"in": "b", "count": 2})+"\n",
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 CmdResp lines, got %d: %s", len(lines), stdout.String())
	}
	var resp mvep.CmdResp
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("line 0 not valid JSON: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error on first record: %+v", resp.Error)
	}
}

// TestSendConcatenated verifies T6: concatenated-object input (no newlines)
// gives the same result as NDJSON.
func TestSendConcatenated(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(
		sendReqJSON("echo_cmd", map[string]any{"in": "a", "count": 1})+
			sendReqJSON("echo_cmd", map[string]any{"in": "b", "count": 2}),
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 CmdResp lines, got %d: %s", len(lines), stdout.String())
	}
}

// TestSendMalformedContinues verifies T6: one malformed record yields a
// CmdResp.Error and the stream continues (default continue-on-error).
func TestSendMalformedContinues(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(
		sendReqJSON("echo_cmd", map[string]any{"in": "a", "count": 1})+"\n"+
			"NOT JSON\n"+
			sendReqJSON("echo_cmd", map[string]any{"in": "b", "count": 2})+"\n",
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected non-zero error because a record failed, got nil")
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CmdResp lines (2 ok + 1 error), got %d: %s", len(lines), stdout.String())
	}
	var resp mvep.CmdResp
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil {
		t.Fatalf("middle line not valid JSON: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "decode_error" {
		t.Errorf("middle record should be a decode_error CmdResp; got: %+v", resp.Error)
	}
}

// TestSendFailFast verifies T6: --fail-fast halts at the first failure.
func TestSendFailFast(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(
		sendReqJSON("echo_cmd", map[string]any{"in": "a", "count": 1})+"\n"+
			"NOT JSON\n"+
			sendReqJSON("echo_cmd", map[string]any{"in": "b", "count": 2})+"\n",
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send", "--fail-fast"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error with --fail-fast, got nil")
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 CmdResp lines (halt after first error), got %d: %s", len(lines), stdout.String())
	}
}

// TestSendResponseHeaders verifies T6: a header set via SetResponseHeader
// appears on the emitted CmdResp, and a request header is readable.
func TestSendResponseHeaders(t *testing.T) {
	ex := &headerExecutor{}
	app := New(&t4Desc, ex, WithStdin(strings.NewReader(
		`{"cmd":"echo_cmd","headers":{"trace-id":"t1"},"payload":"`+base64.StdEncoding.EncodeToString([]byte(`{"in":"a","count":1}`))+`"}`+"\n",
	)))

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"mvep", "send"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex.gotHeader != "t1" {
		t.Errorf("executor read request header = %q, want t1", ex.gotHeader)
	}
	var resp mvep.CmdResp
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if resp.Headers["resp-trace"] != "abc" {
		t.Errorf("response header resp-trace = %q, want abc; got headers %v", resp.Headers["resp-trace"], resp.Headers)
	}
}

// TestSendFlushesPerRecord verifies #54: each CmdResp is written and flushed as
// its record is processed, so a response is readable before the input closes.
func TestSendFlushesPerRecord(t *testing.T) {
	ex := &recordingExecutor{result: &t4EchoResult{Out: "ok"}}

	// Wire send's stdin to a pipe we control, and its stdout to a pipe we read
	// from incrementally.
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	app := New(&t4Desc, ex, WithStdin(inR))

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = app.RunWithIO(context.Background(), []string{"mvep", "send"}, outW, io.Discard)
	}()

	// Write one request (a single line with a trailing newline).
	req := `{"cmd":"echo_cmd","payload":"` + base64.StdEncoding.EncodeToString([]byte(`{"in":"a","count":1}`)) + `"}` + "\n"
	if _, err := io.WriteString(inW, req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	// Read a response BEFORE closing stdin, with a timeout. If send buffers to
	// EOF, no data arrives and this times out.
	type readResult struct {
		n   int
		err error
	}
	readCh := make(chan readResult, 1)
	buf := make([]byte, 4096)
	go func() {
		n, err := outR.Read(buf)
		readCh <- readResult{n: n, err: err}
	}()

	select {
	case rr := <-readCh:
		if rr.err != nil {
			t.Fatalf("no response before input closed (send buffers to EOF?): %v", rr.err)
		}
		var resp mvep.CmdResp
		if err := json.Unmarshal(buf[:rr.n], &resp); err != nil {
			t.Fatalf("response not valid JSON: %v; got %q", err, buf[:rr.n])
		}
		if resp.Error != nil {
			t.Errorf("unexpected error on record: %+v", resp.Error)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a response before input closed — send buffers to EOF")
	}

	// Close stdin and stdout so the send loop ends and the writer unblocks.
	inW.Close()
	outW.Close()
	<-done
}
