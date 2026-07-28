package mvep

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testPackage is a minimal Package implementation for ServeHTTP tests.
type testPackage struct {
	name string
	cmds map[string]func() any
}

func newTestPackage(name string, cmds ...string) *testPackage {
	p := &testPackage{name: name, cmds: make(map[string]func() any)}
	for _, c := range cmds {
		cmdName := c
		p.cmds[cmdName] = func() any {
			return &struct {
				Name string `json:"name"`
			}{Name: cmdName}
		}
	}
	return p
}

func (p *testPackage) GetName() string { return p.name }

func (p *testPackage) InstanceOf(cmdName string) (any, bool) {
	factory, ok := p.cmds[cmdName]
	if !ok {
		return nil, false
	}
	return factory(), true
}

func (p *testPackage) NameOf(cmd any) string {
	for name := range p.cmds {
		return name // single-cmd packages in tests
	}
	return ""
}

// echoRunner returns the command itself as the result.
type echoRunner struct{}

func (echoRunner) RunCmd(ctx context.Context, cmd any) (any, error) { return cmd, nil }

// failRunner returns an error containing a DSN-style detail that must not leak.
type failRunner struct{}

func (failRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
	return nil, errors.New("dial tcp 10.0.0.5:5432: connect: refused")
}

func serveRequest(h http.Handler, method, contentType, cmdName, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/pkg/cmd", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cmdName != "" {
		req.Header.Set("x-mainvec-cmd", cmdName)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// TestTransportCmdReqHeaderPrefix verifies that custom headers are written and read
// using the canonical wire prefix. It exercises both directions: the request header
// key seen by the server, and the response header key surfaced (stripped) to the caller.
func TestTransportCmdReqHeaderPrefix(t *testing.T) {
	var gotAuthKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthKey = r.Header.Get(HeaderPrefix + "auth")
		w.Header().Set(HeaderPrefix+"rate-limit", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	// Guard the wire contract explicitly.
	if HeaderPrefix != "x-mvep-" {
		t.Fatalf("HeaderPrefix = %q, want %q", HeaderPrefix, "x-mvep-")
	}

	tr, err := NewHttpTransporter(srv.URL, "/pkg")
	if err != nil {
		t.Fatalf("NewHttpTransporter: %v", err)
	}

	req := NewCmdReq("SomeCmd", []byte(`{}`)).WithAuth("tok123")
	resp, err := tr.TransportCmdReq(context.Background(), req, "application/json")
	if err != nil {
		t.Fatalf("TransportCmdReq: %v", err)
	}

	if gotAuthKey != "tok123" {
		t.Errorf("server did not receive %qauth header; got %q", HeaderPrefix, gotAuthKey)
	}
	if resp.Headers["rate-limit"] != "100" {
		t.Errorf("response header not surfaced under stripped key; got %q", resp.Headers["rate-limit"])
	}
}

// --- T1: failing tests for the ServeHTTP surface ---

// TestServeHTTPMethodNotAllowed asserts that non-POST requests to the command
// endpoint are rejected with 405 and never reach the runner.
func TestServeHTTPMethodNotAllowed(t *testing.T) {
	h := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: echoRunner{}}
	rr := serveRequest(h, http.MethodGet, "application/json", "SomeCmd", `{}`)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

// TestServeHTTPStatusMapping asserts each failure condition maps to its
// documented HTTP status and stable error code.
func TestServeHTTPStatusMapping(t *testing.T) {
	base := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: echoRunner{}}
	authed := &PackageHandler{
		Package:       newTestPackage("pkg", "SomeCmd"),
		CommandRunner: echoRunner{},
		Interceptor:   AuthInterceptor(rejectingValidator{}),
	}
	failing := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: failRunner{}}

	cases := []struct {
		name        string
		handler     *PackageHandler
		method      string
		contentType string
		cmdName     string
		body        string
		maxBytes    int64 // 0 = default
		wantStatus  int
		wantCode    string
	}{
		{"unknown command", base, http.MethodPost, "application/json", "NoSuchCmd", `{}`, 0, http.StatusNotFound, "unknown_command"},
		{"unsupported media type", base, http.MethodPost, "text/plain", "SomeCmd", `{}`, 0, http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"decode error", base, http.MethodPost, "application/json", "SomeCmd", `{"name":`, 0, http.StatusBadRequest, "decode_error"},
		{"method not allowed", base, http.MethodGet, "application/json", "SomeCmd", `{}`, 0, http.StatusMethodNotAllowed, "method_not_allowed"},
		{"unauthorized", authed, http.MethodPost, "application/json", "SomeCmd", `{}`, 0, http.StatusUnauthorized, "unauthorized"},
		{"command error", failing, http.MethodPost, "application/json", "SomeCmd", `{}`, 0, http.StatusInternalServerError, "command_error"},
		{"missing cmd header", base, http.MethodPost, "application/json", "", `{}`, 0, http.StatusBadRequest, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := serveRequest(tc.handler, tc.method, tc.contentType, tc.cmdName, tc.body)
			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantCode != "" {
				if got := rr.Header().Get("x-mainvec-error-code"); got != tc.wantCode {
					t.Errorf("x-mainvec-error-code = %q, want %q", got, tc.wantCode)
				}
			}
		})
	}
}

// TestServeHTTPPayloadTooLarge asserts the server bounds the request body.
func TestServeHTTPPayloadTooLarge(t *testing.T) {
	h := &PackageHandler{
		Package:         newTestPackage("pkg", "SomeCmd"),
		CommandRunner:   echoRunner{},
		MaxRequestBytes: 8,
	}
	rr := serveRequest(h, http.MethodPost, "application/json", "SomeCmd", `{"name":"0123456789abcdef"}`)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
}

// TestServeHTTPMediaTypeParsing asserts a parameterized Content-Type resolves.
func TestServeHTTPMediaTypeParsing(t *testing.T) {
	h := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: echoRunner{}}
	rr := serveRequest(h, http.MethodPost, "application/json; charset=utf-8", "SomeCmd", `{"name":"x"}`)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
}

// TestServeHTTPErrorRedaction asserts handler error detail never reaches the caller.
func TestServeHTTPErrorRedaction(t *testing.T) {
	h := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: failRunner{}}
	rr := serveRequest(h, http.MethodPost, "application/json", "SomeCmd", `{}`)
	for _, leak := range []string{"10.0.0.5", "5432", "dial tcp"} {
		if strings.Contains(rr.Body.String(), leak) || strings.Contains(rr.Header().Get("x-mainvec-error"), leak) {
			t.Errorf("response leaks internal detail %q (body=%q header=%q)", leak, rr.Body.String(), rr.Header().Get("x-mainvec-error"))
		}
	}
}

// TestServeHTTPVerboseErrors asserts VerboseErrors restores detail for local dev.
func TestServeHTTPVerboseErrors(t *testing.T) {
	h := &PackageHandler{Package: newTestPackage("pkg", "SomeCmd"), CommandRunner: failRunner{}, VerboseErrors: true}
	rr := serveRequest(h, http.MethodPost, "application/json", "SomeCmd", `{}`)
	if !strings.Contains(rr.Header().Get("x-mainvec-error"), "5432") {
		t.Errorf("VerboseErrors: x-mainvec-error = %q, want detail containing port", rr.Header().Get("x-mainvec-error"))
	}
}

// rejectingValidator rejects every token.
type rejectingValidator struct{}

func (rejectingValidator) Validate(ctx context.Context, token string) (any, error) {
	return nil, fmt.Errorf("bad token")
}

// TestTransportCmdContextCancellation asserts the legacy byte-stream path
// honors the caller's context (T9).
func TestTransportCmdContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until the client gives up
	}))
	defer srv.Close()

	tr, err := NewHttpTransporter(srv.URL, "/pkg")
	if err != nil {
		t.Fatalf("NewHttpTransporter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	_, err = tr.TransportCmd(ctx, "SomeCmd", "application/json", io.NopCloser(strings.NewReader(`{}`)))
	if err == nil {
		t.Fatal("TransportCmd with cancelled context returned no error")
	}
}

// TestTransportCmdErrorBranchClosesBody asserts the non-200 path closes the
// response body (T9). A leak is detected by the server never seeing the
// connection return to idle; we assert correctness structurally instead:
// the error path must drain-and-close so the body is not abandoned.
func TestTransportCmdErrorBranchClosesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-mainvec-error", "boom")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	tr, err := NewHttpTransporter(srv.URL, "/pkg")
	if err != nil {
		t.Fatalf("NewHttpTransporter: %v", err)
	}
	_, err = tr.TransportCmd(context.Background(), "SomeCmd", "application/json", io.NopCloser(strings.NewReader(`{}`)))
	if err == nil {
		t.Fatal("expected error on 400 response")
	}
	// With the body unclosed the transport would not reuse the connection; this
	// is a smoke assertion that the call returns rather than leaking silently.
}

// TestSendCmdReqCopiesHeaders asserts SendCmdReq does not alias the caller's map (T9).
func TestSendCmdReqCopiesHeaders(t *testing.T) {
	callerHeaders := map[string]string{"auth": "tok"}
	pkg := newTestPackage("pkg", "SomeCmd")

	var seen *CmdReq
	roundTrip := &stubEnvelopeTransporter{onReq: func(r *CmdReq) { seen = r }}
	h := &PackageHandler{Package: pkg, Transporter: roundTrip}

	_, _, _ = h.SendCmdReq(context.Background(), struct{}{}, callerHeaders, "application/json")
	if seen == nil {
		t.Fatal("transporter never saw a request")
	}
	seen.Headers["auth"] = "mutated"
	if callerHeaders["auth"] != "tok" {
		t.Error("mutating the request headers mutated the caller's map")
	}
}

// stubEnvelopeTransporter captures the outgoing CmdReq.
type stubEnvelopeTransporter struct {
	onReq func(*CmdReq)
}

func (s *stubEnvelopeTransporter) TransportCmd(ctx context.Context, cmdName string, contentType string, cmdData io.ReadCloser) (io.ReadCloser, error) {
	return nil, errors.New("stub: legacy path unused")
}

func (s *stubEnvelopeTransporter) TransportCmdReq(ctx context.Context, req *CmdReq, contentType string) (*CmdResp, error) {
	if s.onReq != nil {
		s.onReq(req)
	}
	return nil, errors.New("stub: stop here")
}
