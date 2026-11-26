package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

type testAddress struct {
	City string `json:"city,omitempty"`
}

type greetCmd struct {
	Name    string            `json:"name,omitempty"`
	Count   int32             `json:"count,omitempty"`
	Timeout time.Duration     `json:"timeout,omitempty"`
	Address *testAddress      `json:"address,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

type greetCmdResult struct {
	Greeting string `json:"greeting,omitempty"`
	City     string `json:"city,omitempty"`
}

type failCmd struct{}
type failCmdResult struct{}

type panicCmd struct{}
type panicCmdResult struct{}

type testRunner struct{}

func (r *testRunner) RunCmd(_ context.Context, cmd any) (any, error) {
	switch c := cmd.(type) {
	case *greetCmd:
		city := ""
		if c.Address != nil {
			city = c.Address.City
		}
		return &greetCmdResult{
			Greeting: "hello " + c.Name,
			City:     city,
		}, nil
	case *failCmd:
		return nil, errors.New("boom from handler")
	case *panicCmd:
		panic("unexpected handler crash")
	default:
		return nil, errors.New("unknown command")
	}
}

func newTestPackage() (*mvep.PackageDesc, *mvep.PackageHandler) {
	desc := &mvep.PackageDesc{
		Name:        "greeter",
		SpecVersion: "v1.2.3",
		Records: []mvep.RecordDesc{
			{
				Name: "Address",
				Fields: []mvep.FieldDesc{
					{Name: "city", Type: mvep.FieldString, Required: true},
				},
			},
		},
		Commands: []mvep.CommandDesc{
			{
				Name:  "GreetCmd",
				Alias: "greet",
				Group: "social",
				Desc:  "Greet a user",
				New:   func() any { return &greetCmd{} },
				Fields: []mvep.FieldDesc{
					{Name: "name", Type: mvep.FieldString, Required: true, Desc: "User name"},
					{Name: "count", Type: mvep.FieldInt32},
					{Name: "timeout", Type: mvep.FieldDuration},
					{Name: "address", Type: mvep.FieldRecord, Ref: &mvep.RecordDesc{Name: "Address"}},
					{Name: "meta", Type: mvep.FieldMap},
				},
				Result: &mvep.ResultDesc{
					Name: "GreetCmdResult",
					New:  func() any { return &greetCmdResult{} },
				},
			},
			{
				Name:  "FailCmd",
				Alias: "fail",
				Desc:  "Always returns an error",
				New:   func() any { return &failCmd{} },
				Result: &mvep.ResultDesc{
					Name: "FailCmdResult",
					New:  func() any { return &failCmdResult{} },
				},
			},
			{
				Name:  "PanicCmd",
				Alias: "boom",
				Desc:  "Panics inside handler",
				New:   func() any { return &panicCmd{} },
				Result: &mvep.ResultDesc{
					Name: "PanicCmdResult",
					New:  func() any { return &panicCmdResult{} },
				},
			},
		},
	}

	handler := &mvep.PackageHandler{
		Package:       mvep.NewPackageFromDesc(desc),
		CommandRunner: &testRunner{},
		Interceptor:   mvep.Chain(mvep.RecoveryInterceptor()),
	}
	return desc, handler
}

func TestServer_PipeRoundTrip(t *testing.T) {
	desc, handler := newTestPackage()
	srv, err := NewServer(desc, handler)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	clientToSrvR, clientToSrvW := io.Pipe()
	srvToClientR, srvToClientW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(ctx, NewStreamTransport(clientToSrvR, srvToClientW))
	}()

	reader := bufio.NewReader(srvToClientR)
	send := func(msg string) map[string]any {
		t.Helper()
		if _, err := io.WriteString(clientToSrvW, msg+"\n"); err != nil {
			t.Fatalf("write: %v", err)
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read response: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(line, &out); err != nil {
			t.Fatalf("unmarshal response %q: %v", line, err)
		}
		return out
	}

	// 1. initialize
	initResp := send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)
	initRes := initResp["result"].(map[string]any)
	if initRes["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v", initRes["protocolVersion"])
	}
	info := initRes["serverInfo"].(map[string]any)
	if info["name"] != "greeter" || info["version"] != "v1.2.3" {
		t.Errorf("serverInfo = %v", info)
	}

	// 2. notification must emit no response; verify by immediately sending ping.
	if _, err := io.WriteString(clientToSrvW, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"); err != nil {
		t.Fatalf("write notification: %v", err)
	}
	pingResp := send(`{"jsonrpc":"2.0","id":"p1","method":"ping"}`)
	if pingResp["id"] != "p1" {
		t.Errorf("expected ping response id=p1 (notification should produce no reply), got %v", pingResp)
	}

	// 3. tools/list
	listResp := send(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	if tools[0].(map[string]any)["name"] != "social_greet" {
		t.Errorf("first tool = %v, want social_greet", tools[0])
	}

	// 4. tools/call happy path with nested record, map, and duration.
	callResp := send(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"social_greet","arguments":{"name":"Alice","count":2,"timeout":5000000000,"address":{"city":"Yerevan"},"meta":{"k":"v"}}}}`)
	callRes := callResp["result"].(map[string]any)
	if callRes["isError"] == true {
		t.Fatalf("unexpected tool error: %v", callRes)
	}
	contentText := callRes["content"].([]any)[0].(map[string]any)["text"].(string)
	var decoded greetCmdResult
	if err := json.Unmarshal([]byte(contentText), &decoded); err != nil {
		t.Fatalf("unmarshal tool content %q: %v", contentText, err)
	}
	if decoded.Greeting != "hello Alice" || decoded.City != "Yerevan" {
		t.Errorf("decoded = %+v", decoded)
	}

	// 5. panicking handler must be recovered by PackageHandler's RecoveryInterceptor
	// and surfaced as an MCP tool error without killing the stdio loop.
	panicResp := send(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"boom","arguments":{}}}`)
	panicRes := panicResp["result"].(map[string]any)
	if panicRes["isError"] != true {
		t.Fatalf("expected isError=true for panic, got %v", panicRes)
	}
	panicText := panicRes["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(panicText, "[internal_error]") {
		t.Errorf("expected [internal_error] error text, got %q", panicText)
	}

	// 6. Server is still alive after panic; handler error surfaces as isError=true.
	failResp := send(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"fail"}}`)
	failRes := failResp["result"].(map[string]any)
	if failRes["isError"] != true {
		t.Fatalf("expected isError=true for handler error, got %v", failRes)
	}
	failText := failRes["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(failText, "boom from handler") {
		t.Errorf("expected handler error message, got %q", failText)
	}

	// Close client write side -> clean EOF return from Serve.
	_ = clientToSrvW.Close()
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve returned unexpected error on EOF: %v", err)
	}
}

func TestServer_ValidationAndProtocolErrors(t *testing.T) {
	desc, handler := newTestPackage()
	srv, err := NewServer(desc, handler)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	call := func(req string) map[string]any {
		t.Helper()
		raw, err := srv.HandleMessage(context.Background(), []byte(req))
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal %q: %v", raw, err)
		}
		return out
	}

	// Unknown top-level argument.
	r := call(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"social_greet","arguments":{"name":"Bob","bogus":1}}}`)
	res := r["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), `unknown argument "bogus"`) {
		t.Errorf("expected unknown argument error, got %v", r)
	}

	// Unknown nested record argument.
	r = call(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"social_greet","arguments":{"name":"Bob","address":{"city":"NY","extra":true}}}}`)
	res = r["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), `unknown argument "extra" in address`) {
		t.Errorf("expected nested unknown argument error, got %v", r)
	}

	// Missing required argument.
	r = call(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"social_greet","arguments":{}}}`)
	res = r["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), `missing required argument "name"`) {
		t.Errorf("expected missing required argument error, got %v", r)
	}

	// Unknown tool -> JSON-RPC -32602.
	r = call(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"does_not_exist"}}`)
	if int(r["error"].(map[string]any)["code"].(float64)) != codeInvalidParams {
		t.Errorf("expected -32602 for unknown tool, got %v", r)
	}

	// Parse error -> -32700.
	r = call(`{not json`)
	if int(r["error"].(map[string]any)["code"].(float64)) != codeParseError {
		t.Errorf("expected -32700 for parse error, got %v", r)
	}

	// Unknown method -> -32601.
	r = call(`{"jsonrpc":"2.0","id":5,"method":"resources/list"}`)
	if int(r["error"].(map[string]any)["code"].(float64)) != codeMethodNotFound {
		t.Errorf("expected -32601 for unknown method, got %v", r)
	}
}

type denyAllValidator struct{}

func (denyAllValidator) Validate(_ context.Context, _ string) (any, error) {
	return nil, errors.New("denied")
}

func TestServer_InterceptorApplies(t *testing.T) {
	desc, handler := newTestPackage()
	handler.Interceptor = mvep.Chain(
		mvep.RecoveryInterceptor(),
		mvep.AuthInterceptor(denyAllValidator{}),
	)

	srv, err := NewServer(desc, handler)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	raw, err := srv.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"social_greet","arguments":{"name":"Alice"}}}`))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	res := out["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("expected auth rejection as isError=true, got %v", res)
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "[unauthorized]") {
		t.Errorf("expected [unauthorized], got %q", text)
	}
}

func TestStreamTransport_FramingGuards(t *testing.T) {
	var out bytes.Buffer
	tr := NewStreamTransport(strings.NewReader("{\"a\":1}\n"), &out)
	tr.maxMessageBytes = 16

	msg, err := tr.Read(context.Background())
	if err != nil {
		t.Fatalf("read small message: %v", err)
	}
	if string(msg) != `{"a":1}` {
		t.Errorf("msg = %q", msg)
	}

	// Oversized inbound line fails.
	trBig := NewStreamTransport(strings.NewReader(strings.Repeat("x", 32)+"\n"), &out)
	trBig.maxMessageBytes = 16
	if _, err := trBig.Read(context.Background()); err == nil {
		t.Error("expected error for oversized line")
	}

	// Outbound message containing newline is rejected.
	if err := tr.Write(context.Background(), []byte("{\"a\":1}\n{\"b\":2}")); err == nil {
		t.Error("expected error for outbound message with newline")
	}
}
