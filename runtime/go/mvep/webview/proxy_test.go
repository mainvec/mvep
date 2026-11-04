package webview

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// unixDaemon serves h on a Unix socket and returns its unix:// address. The
// socket lives under os.TempDir with a short name, since macOS limits socket
// paths to 104 bytes and t.TempDir paths can exceed that.
func unixDaemon(t *testing.T, h http.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "wv")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return "unix://" + path
}

func startProxy(t *testing.T, opts ProxyOptions) *Proxy {
	t.Helper()
	if opts.AllowOrigin == nil {
		opts.AllowOrigin = WailsOrigin
	}
	p, err := Start(opts)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func proxyRequest(t *testing.T, ctx context.Context, p *Proxy, path string, header map[string]string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL+path, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("Authorization", "Bearer "+p.Token)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	return req
}

// recordingDaemon records the headers of the last request it served and
// answers with an upstream CORS header the proxy must strip.
type recordingDaemon struct {
	got  chan http.Header
	path chan string
}

func newRecordingDaemon() *recordingDaemon {
	return &recordingDaemon{got: make(chan http.Header, 1), path: make(chan string, 1)}
}

func (d *recordingDaemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.got <- r.Header.Clone()
	d.path <- r.URL.Path
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("x-mvep-job-id", "j1")
	_, _ = io.WriteString(w, "daemon-ok")
}

func TestProxyRewritesCredentialsAndOrigin(t *testing.T) {
	for name, mk := range map[string]func(t *testing.T, h http.Handler) string{
		"unix": unixDaemon,
		"tcp": func(t *testing.T, h http.Handler) string {
			s := httptest.NewServer(h)
			t.Cleanup(s.Close)
			return s.URL
		},
		"bare host:port": func(t *testing.T, h http.Handler) string {
			s := httptest.NewServer(h)
			t.Cleanup(s.Close)
			return strings.TrimPrefix(s.URL, "http://")
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := newRecordingDaemon()
			p := startProxy(t, ProxyOptions{Daemon: mk(t, d), DaemonToken: "daemon-secret"})

			req := proxyRequest(t, context.Background(), p, "/api/pkg/cmd", map[string]string{
				"Referer":       "wails://wails/index.html",
				"Cookie":        "a=b",
				"x-mainvec-cmd": "EchoCmd",
			})
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != 200 || string(body) != "daemon-ok" {
				t.Fatalf("status %d body %q, want 200 daemon-ok", resp.StatusCode, body)
			}
			h := <-d.got
			if got := <-d.path; got != "/api/pkg/cmd" {
				t.Errorf("daemon path = %q, want /api/pkg/cmd", got)
			}
			for _, k := range []string{"Origin", "Referer", "Cookie", "Authorization"} {
				if v := h.Get(k); v != "" {
					t.Errorf("daemon saw %s = %q, want it dropped", k, v)
				}
			}
			if got := h.Get("x-mvep-auth"); got != "daemon-secret" {
				t.Errorf("daemon saw x-mvep-auth %q, want the daemon token", got)
			}
			if got := h.Get("x-mainvec-cmd"); got != "EchoCmd" {
				t.Errorf("daemon saw x-mainvec-cmd %q, want EchoCmd", got)
			}
			if got := resp.Header.Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != testOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want only %q", got, testOrigin)
			}
			if got := resp.Header.Get("Access-Control-Expose-Headers"); got != "x-mvep-job-id" {
				t.Errorf("Access-Control-Expose-Headers = %q, want x-mvep-job-id", got)
			}
		})
	}
}

func TestProxyWithoutDaemonTokenSendsNoCredential(t *testing.T) {
	d := newRecordingDaemon()
	p := startProxy(t, ProxyOptions{Daemon: unixDaemon(t, d)})
	req := proxyRequest(t, context.Background(), p, "/", map[string]string{"x-mvep-auth": "ignored"})
	req.Header.Del("Authorization")
	req.Header.Set("x-mvep-auth", p.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := (<-d.got).Get("x-mvep-auth"); got != "" {
		t.Errorf("daemon saw x-mvep-auth %q, want none (the session token must not leak)", got)
	}
}

func TestProxyStreams(t *testing.T) {
	release := make(chan struct{})
	daemon := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "first\n")
		_ = http.NewResponseController(w).Flush()
		<-release
		_, _ = io.WriteString(w, "second\n")
	})
	p := startProxy(t, ProxyOptions{Daemon: unixDaemon(t, daemon)})

	resp, err := http.DefaultClient.Do(proxyRequest(t, context.Background(), p, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	lines := make(chan string, 2)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	select {
	case l := <-lines:
		if l != "first" {
			t.Fatalf("first line = %q", l)
		}
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("first chunk did not arrive before the daemon finished: the proxy buffers")
	}
	close(release)
	if l := <-lines; l != "second" {
		t.Errorf("second line = %q", l)
	}
}

func TestProxyPropagatesCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	daemon := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// net/http only watches for a closed connection once the body has
		// been read to EOF, as PackageHandler.ServeHTTP does.
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		select {
		case <-r.Context().Done():
			close(cancelled)
		case <-time.After(10 * time.Second):
		}
	})
	p := startProxy(t, ProxyOptions{Daemon: unixDaemon(t, daemon)})

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(proxyRequest(t, ctx, p, "/", nil))
		if err == nil {
			resp.Body.Close()
		}
		errc <- err
	}()

	<-started
	cancel()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("daemon request context was not cancelled")
	}
	if err := <-errc; !errors.Is(err, context.Canceled) {
		t.Errorf("client error = %v, want context.Canceled", err)
	}
}

func TestProxyDeadDaemon(t *testing.T) {
	p := startProxy(t, ProxyOptions{Daemon: "unix://" + filepath.Join(os.TempDir(), "wv-missing.sock")})
	resp, err := http.DefaultClient.Do(proxyRequest(t, context.Background(), p, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
	if got := resp.Header.Get("x-mainvec-error-code"); got != "daemon_unavailable" {
		t.Errorf("x-mainvec-error-code = %q, want daemon_unavailable", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want the page to be able to read the 502", got)
	}
}

func TestProxyClose(t *testing.T) {
	p, err := Start(ProxyOptions{Daemon: unixDaemon(t, newRecordingDaemon()), AllowOrigin: WailsOrigin})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := http.DefaultClient.Do(proxyRequest(t, context.Background(), p, "/", nil)); err == nil {
		t.Error("request after Close succeeded")
	}
}

func TestStartValidatesOptions(t *testing.T) {
	ok := ProxyOptions{Daemon: "http://127.0.0.1:7000", AllowOrigin: WailsOrigin}
	cases := map[string]func(o *ProxyOptions){
		"missing AllowOrigin":   func(o *ProxyOptions) { o.AllowOrigin = nil },
		"missing Daemon":        func(o *ProxyOptions) { o.Daemon = "" },
		"empty unix path":       func(o *ProxyOptions) { o.Daemon = "unix://" },
		"unsupported scheme":    func(o *ProxyOptions) { o.Daemon = "ftp://127.0.0.1:21" },
		"wildcard listen":       func(o *ProxyOptions) { o.Addr = "0.0.0.0:0" },
		"hostname listen":       func(o *ProxyOptions) { o.Addr = "localhost:0" },
		"listen without a port": func(o *ProxyOptions) { o.Addr = "127.0.0.1" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			o := ok
			mutate(&o)
			if p, err := Start(o); err == nil {
				_ = p.Close()
				t.Error("Start succeeded, want an error")
			}
		})
	}

	p, err := Start(ProxyOptions{Daemon: ok.Daemon, AllowOrigin: WailsOrigin, Addr: "[::1]:0"})
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer p.Close()
	if !strings.HasPrefix(p.URL, "http://[::1]:") || p.Token == "" {
		t.Errorf("URL %q, Token %q", p.URL, p.Token)
	}
}
