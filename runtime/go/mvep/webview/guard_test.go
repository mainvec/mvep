package webview

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const (
	testToken  = "session-token"
	testOrigin = "wails://wails"
)

// guardServer serves Guard over a real loopback listener, so Host and the
// connection's local address are what a browser would produce. next records
// whether it ran and the x-mvep-auth it saw.
type guardServer struct {
	*httptest.Server
	reached  bool
	sawAuth  string
	setNextH func(http.Header)
}

func newGuardServer(t *testing.T, opts GuardOptions) *guardServer {
	t.Helper()
	gs := &guardServer{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gs.reached = true
		gs.sawAuth = r.Header.Get("x-mvep-auth")
		if gs.setNextH != nil {
			gs.setNextH(w.Header())
		}
		_, _ = io.WriteString(w, "ok")
	})
	gs.Server = httptest.NewServer(Guard(next, opts))
	t.Cleanup(gs.Close)
	return gs
}

type reqSpec struct {
	method  string
	host    string
	headers map[string]string
}

func (gs *guardServer) do(t *testing.T, rs reqSpec) *http.Response {
	t.Helper()
	gs.reached, gs.sawAuth = false, ""
	method := rs.method
	if method == "" {
		method = http.MethodPost
	}
	req, err := http.NewRequest(method, gs.URL+"/pkg/cmd", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if rs.host != "" {
		req.Host = rs.host
	}
	for k, v := range rs.headers {
		req.Header.Set(k, v)
	}
	resp, err := gs.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp
}

func TestGuardRules(t *testing.T) {
	gs := newGuardServer(t, GuardOptions{Token: testToken, AllowOrigin: WailsOrigin})
	port := gs.Listener.Addr().(*net.TCPAddr).Port

	tests := []struct {
		name       string
		req        reqSpec
		wantStatus int
		wantACAO   string
		wantCode   string
	}{
		{"bearer token, no origin", reqSpec{headers: map[string]string{"Authorization": "Bearer " + testToken}}, 200, "", ""},
		{"x-mvep-auth token, no origin", reqSpec{headers: map[string]string{"x-mvep-auth": testToken}}, 200, "", ""},
		{"lowercase bearer scheme", reqSpec{headers: map[string]string{"Authorization": "bearer " + testToken}}, 200, "", ""},
		{"own origin with token", reqSpec{headers: map[string]string{"Origin": testOrigin, "Authorization": "Bearer " + testToken}}, 200, testOrigin, ""},
		{"no token", reqSpec{}, 401, "", "unauthorized"},
		{"wrong token", reqSpec{headers: map[string]string{"Authorization": "Bearer nope"}}, 401, "", "unauthorized"},
		{"token prefix only", reqSpec{headers: map[string]string{"x-mvep-auth": testToken[:4]}}, 401, "", "unauthorized"},
		{"non-bearer authorization", reqSpec{headers: map[string]string{"Authorization": "Basic " + testToken}}, 401, "", "unauthorized"},
		{"bad authorization shadows good x-mvep-auth", reqSpec{headers: map[string]string{"Authorization": "Bearer nope", "x-mvep-auth": testToken}}, 401, "", "unauthorized"},
		{"own origin, no token: readable 401", reqSpec{headers: map[string]string{"Origin": testOrigin}}, 401, testOrigin, "unauthorized"},
		{"foreign origin with token", reqSpec{headers: map[string]string{"Origin": "https://attacker.example", "Authorization": "Bearer " + testToken}}, 403, "", "forbidden"},
		{"opaque null origin", reqSpec{headers: map[string]string{"Origin": "null", "Authorization": "Bearer " + testToken}}, 403, "", "forbidden"},
		{"rebound host with token", reqSpec{host: "attacker.example:" + strconv.Itoa(port), headers: map[string]string{"Authorization": "Bearer " + testToken}}, 403, "", "forbidden"},
		{"localhost name is not the listener", reqSpec{host: "localhost:" + strconv.Itoa(port), headers: map[string]string{"Authorization": "Bearer " + testToken}}, 403, "", "forbidden"},
		{"wrong port", reqSpec{host: "127.0.0.1:1", headers: map[string]string{"Authorization": "Bearer " + testToken}}, 403, "", "forbidden"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := gs.do(t, tc.req)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if got := resp.Header.Get("Access-Control-Allow-Origin"); got != tc.wantACAO {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantACAO)
			}
			if got := resp.Header.Get("x-mainvec-error-code"); got != tc.wantCode {
				t.Errorf("x-mainvec-error-code = %q, want %q", got, tc.wantCode)
			}
			if !strings.Contains(strings.Join(resp.Header.Values("Vary"), ","), "Origin") {
				t.Errorf("Vary = %q, want it to include Origin", resp.Header.Values("Vary"))
			}
			if gs.reached != (tc.wantStatus == 200) {
				t.Errorf("next reached = %v, want %v", gs.reached, tc.wantStatus == 200)
			}
		})
	}
}

func TestGuardBearerIsVisibleAsMvepAuth(t *testing.T) {
	gs := newGuardServer(t, GuardOptions{Token: testToken, AllowOrigin: WailsOrigin})
	gs.do(t, reqSpec{headers: map[string]string{"Authorization": "Bearer " + testToken}})
	if gs.sawAuth != testToken {
		t.Errorf("next saw x-mvep-auth %q, want the session token", gs.sawAuth)
	}
}

func TestGuardNilAllowOriginRefusesAnyOrigin(t *testing.T) {
	gs := newGuardServer(t, GuardOptions{Token: testToken})
	if resp := gs.do(t, reqSpec{headers: map[string]string{"Origin": testOrigin, "x-mvep-auth": testToken}}); resp.StatusCode != 403 {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
	if resp := gs.do(t, reqSpec{headers: map[string]string{"x-mvep-auth": testToken}}); resp.StatusCode != 200 {
		t.Errorf("no-Origin status = %d, want 200", resp.StatusCode)
	}
}

func TestGuardPreflight(t *testing.T) {
	gs := newGuardServer(t, GuardOptions{Token: testToken, AllowOrigin: WailsOrigin})

	t.Run("own origin", func(t *testing.T) {
		resp := gs.do(t, reqSpec{method: http.MethodOptions, headers: map[string]string{
			"Origin":                                 testOrigin,
			"Access-Control-Request-Method":          "POST",
			"Access-Control-Request-Headers":         "Content-Type, Authorization, x-mainvec-cmd, x-mvep-request-id, x-evil",
			"Access-Control-Request-Private-Network": "true",
		}})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gs.reached {
			t.Error("preflight reached next")
		}
		want := map[string]string{
			"Access-Control-Allow-Origin":          testOrigin,
			"Access-Control-Allow-Methods":         "GET, POST",
			"Access-Control-Allow-Headers":         "content-type, authorization, x-mainvec-cmd, x-mvep-request-id",
			"Access-Control-Max-Age":               "600",
			"Access-Control-Allow-Private-Network": "true",
		}
		for k, v := range want {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
	})

	t.Run("foreign origin", func(t *testing.T) {
		resp := gs.do(t, reqSpec{method: http.MethodOptions, headers: map[string]string{
			"Origin": "https://attacker.example", "Access-Control-Request-Method": "POST",
		}})
		if resp.StatusCode != 403 || resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("status = %d, ACAO = %q; want 403 and none", resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("plain OPTIONS needs the token", func(t *testing.T) {
		resp := gs.do(t, reqSpec{method: http.MethodOptions, headers: map[string]string{"Origin": testOrigin}})
		if resp.StatusCode != 401 {
			t.Errorf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("no PNA header unless asked", func(t *testing.T) {
		resp := gs.do(t, reqSpec{method: http.MethodOptions, headers: map[string]string{
			"Origin": testOrigin, "Access-Control-Request-Method": "POST",
		}})
		if got := resp.Header.Get("Access-Control-Allow-Private-Network"); got != "" {
			t.Errorf("Access-Control-Allow-Private-Network = %q, want unset", got)
		}
	})
}

func TestGuardExposesMvepHeaders(t *testing.T) {
	gs := newGuardServer(t, GuardOptions{Token: testToken, AllowOrigin: WailsOrigin})
	gs.setNextH = func(h http.Header) {
		h.Set("x-mvep-job-id", "j1")
		h.Set("x-mainvec-error-code", "decode_error")
		h.Set("X-Other", "x")
	}
	resp := gs.do(t, reqSpec{headers: map[string]string{"Origin": testOrigin, "x-mvep-auth": testToken}})
	if got, want := resp.Header.Get("Access-Control-Expose-Headers"), "x-mainvec-error-code, x-mvep-job-id"; got != want {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", got, want)
	}

	resp = gs.do(t, reqSpec{headers: map[string]string{"x-mvep-auth": testToken}})
	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("without Origin, Access-Control-Expose-Headers = %q, want unset", got)
	}

	gs.setNextH = nil
	resp = gs.do(t, reqSpec{headers: map[string]string{"Origin": testOrigin}})
	if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "x-mainvec-error-code" {
		t.Errorf("on a 401, Access-Control-Expose-Headers = %q, want x-mainvec-error-code", got)
	}
}

func TestGuardFailsClosedWithoutLoopbackListener(t *testing.T) {
	h := Guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next reached")
	}), GuardOptions{Token: testToken, AllowOrigin: WailsOrigin})

	cases := map[string]net.Addr{
		"no local address": nil,
		"non-loopback TCP": &net.TCPAddr{IP: net.IPv4(192, 168, 1, 10), Port: 8080},
		"unix socket":      &net.UnixAddr{Name: "/tmp/x.sock", Net: "unix"},
	}
	for name, addr := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if addr != nil {
				req.Host = addr.String()
				req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, addr))
			}
			req.Header.Set("x-mvep-auth", testToken)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 403 {
				t.Errorf("status = %d, want 403", rec.Code)
			}
		})
	}
}

func TestGuardPanicsOnEmptyToken(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Guard with empty Token did not panic")
		}
	}()
	Guard(http.NotFoundHandler(), GuardOptions{AllowOrigin: WailsOrigin})
}
