package webview

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// ProxyOptions configures Start.
type ProxyOptions struct {
	// Daemon is the mvep daemon to forward to: "unix:///path/to.sock",
	// "http://127.0.0.1:7000", or a bare "127.0.0.1:7000". Required.
	Daemon string
	// DaemonToken is sent to the daemon as x-mvep-auth in place of the session
	// token. Leave it empty for a daemon that trusts its Unix socket (see
	// mvep.LocalTrustMiddleware).
	DaemonToken string
	// AllowOrigin reports whether an Origin is the app's own webview, for
	// example WailsOrigin or CapacitorOrigin. Required.
	AllowOrigin func(origin string) bool
	// Addr is the address the proxy listens on. It must be a loopback IP
	// literal with a port. The default "127.0.0.1:0" picks a free port.
	Addr string
}

// Proxy is a running guarded proxy. Hand URL and Token to the app's page
// through its own bridge (a Wails binding, a Capacitor plugin); the page sends
// Token as "Authorization: Bearer <Token>" or "x-mvep-auth: <Token>".
type Proxy struct {
	// URL is the proxy's base URL, for example "http://127.0.0.1:53124".
	URL string
	// Token is this launch's random session token.
	Token string

	srv       *http.Server
	transport *http.Transport
}

// Close stops the proxy, closing its listener and every open connection.
func (p *Proxy) Close() error {
	err := p.srv.Close()
	p.transport.CloseIdleConnections()
	return err
}

const (
	proxyReadHeaderTimeout = 10 * time.Second
	proxyMaxHeaderBytes    = 1 << 20
)

// Start runs Guard in front of a streaming reverse proxy to a separate daemon,
// on a new loopback listener with a fresh session token.
//
// The proxy replaces the session token with opts.DaemonToken and drops Origin,
// Referer, Cookie and Authorization, so the page never holds the daemon token
// and the daemon never sees a browser origin. CORS headers from the daemon are
// dropped; only Guard's apply. Responses stream as they arrive, and a request
// the page cancels is cancelled at the daemon.
func Start(opts ProxyOptions) (*Proxy, error) {
	if opts.AllowOrigin == nil {
		return nil, errors.New("webview: ProxyOptions.AllowOrigin is required")
	}
	target, transport, err := daemonTarget(opts.Daemon)
	if err != nil {
		return nil, err
	}
	addr := opts.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	if err := checkLoopbackAddr(addr); err != nil {
		return nil, err
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			h := pr.Out.Header
			for _, k := range []string{"Origin", "Referer", "Cookie", "Authorization", mvep.HeaderAuth} {
				h.Del(k)
			}
			if opts.DaemonToken != "" {
				h.Set(mvep.HeaderAuth, opts.DaemonToken)
			}
		},
		Transport:     transport,
		FlushInterval: -1,
		ModifyResponse: func(resp *http.Response) error {
			for k := range resp.Header {
				if strings.HasPrefix(strings.ToLower(k), "access-control-") {
					resp.Header.Del(k)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if r.Context().Err() != nil {
				return // the page went away; nothing to answer
			}
			slog.Warn("webview: daemon request failed", "error", err)
			w.Header().Set(mvep.HeaderErrorCode, "daemon_unavailable")
			http.Error(w, "daemon unavailable", http.StatusBadGateway)
		},
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("webview: listen on %s: %w", addr, err)
	}
	token := NewToken()
	srv := &http.Server{
		Handler:           Guard(rp, GuardOptions{Token: token, AllowOrigin: opts.AllowOrigin}),
		ReadHeaderTimeout: proxyReadHeaderTimeout,
		MaxHeaderBytes:    proxyMaxHeaderBytes,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("webview: proxy stopped", "error", err)
		}
	}()

	return &Proxy{
		URL:       "http://" + ln.Addr().String(),
		Token:     token,
		srv:       srv,
		transport: transport,
	}, nil
}

// daemonTarget parses ProxyOptions.Daemon into the URL to forward to and a
// transport that reaches it.
func daemonTarget(daemon string) (*url.URL, *http.Transport, error) {
	if daemon == "" {
		return nil, nil, errors.New("webview: ProxyOptions.Daemon is required")
	}
	if path, ok := strings.CutPrefix(daemon, "unix://"); ok {
		if path == "" {
			return nil, nil, errors.New("webview: unix daemon address has no socket path")
		}
		var d net.Dialer
		t := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return d.DialContext(ctx, "unix", path)
			},
			MaxIdleConns:    16,
			IdleConnTimeout: 90 * time.Second,
		}
		// The host is a placeholder; DialContext ignores it.
		return &url.URL{Scheme: "http", Host: "localhost"}, t, nil
	}

	if !strings.Contains(daemon, "://") {
		daemon = "http://" + daemon
	}
	u, err := url.Parse(daemon)
	if err != nil {
		return nil, nil, fmt.Errorf("webview: invalid daemon address %q: %w", daemon, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, nil, fmt.Errorf("webview: daemon address %q must be unix://, http:// or https://", daemon)
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	return u, t, nil
}

// checkLoopbackAddr requires addr to be a loopback IP literal with a port. A
// host name such as "localhost" is refused because it may resolve elsewhere.
func checkLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("webview: invalid listen address %q: %w", addr, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("webview: listen address %q must be a loopback IP such as 127.0.0.1", addr)
	}
	return nil
}
