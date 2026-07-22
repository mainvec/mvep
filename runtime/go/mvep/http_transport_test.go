package mvep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
