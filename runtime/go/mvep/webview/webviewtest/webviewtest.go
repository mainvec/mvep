// Package webviewtest checks that a loopback listener meant for an app's own
// webview refuses everyone else. An app runs it in CI against its own
// listener, whether that is a webview.Start proxy or an in-process
// webview.Guard, or any hand-written equivalent.
//
//	p, _ := webview.Start(opts)
//	defer p.Close()
//	webviewtest.Check(t, webviewtest.Config{URL: p.URL, Token: p.Token, Origin: "wails://wails"})
package webviewtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// ForeignOrigin is the origin used to stand in for a hostile web page.
const ForeignOrigin = "https://attacker.example"

// ForeignHost is the Host used to stand in for a domain rebound to loopback.
const ForeignHost = "attacker.example"

// Config describes the listener under test.
type Config struct {
	// URL is the listener's base URL, for example "http://127.0.0.1:53124".
	URL string
	// Token is the session token the app's page sends.
	Token string
	// Origin is the app's own webview origin, which must be admitted.
	Origin string
	// Path is the request path probed on URL. Default "/". Any response that
	// is not a refusal counts as admitted, so a daemon answering the probe with
	// 400 or 404 is fine.
	Path string
	// Client sends the probes. Default: a client with a 10s timeout.
	Client *http.Client
}

// rule is one conformance rule.
type rule struct {
	name  string
	check func(ctx context.Context, p *prober) error
}

// rules returns the conformance rules in the order they are checked.
func rules() []rule {
	return []rule{
		{"no token is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodPost})
		}},
		{"wrong token is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodPost, token: p.cfg.Token + "x"})
		}},
		{"wrong x-mvep-auth token is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodPost, mvepAuth: p.cfg.Token + "x"})
		}},
		{"foreign Origin is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodPost, origin: ForeignOrigin, token: p.cfg.Token})
		}},
		{"foreign Host is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodPost, host: p.foreignHost(), origin: p.cfg.Origin, token: p.cfg.Token})
		}},
		{"foreign preflight is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodOptions, origin: ForeignOrigin, preflight: true})
		}},
		{"preflight under foreign Host is refused", func(ctx context.Context, p *prober) error {
			return p.expectRefused(ctx, probe{method: http.MethodOptions, host: p.foreignHost(), origin: p.cfg.Origin, preflight: true})
		}},
		{"own preflight is admitted", func(ctx context.Context, p *prober) error {
			return p.expectAdmitted(ctx, probe{method: http.MethodOptions, origin: p.cfg.Origin, preflight: true})
		}},
		{"own origin with Bearer token is admitted", func(ctx context.Context, p *prober) error {
			return p.expectAdmitted(ctx, probe{method: http.MethodPost, origin: p.cfg.Origin, token: p.cfg.Token})
		}},
		{"own origin with x-mvep-auth token is admitted", func(ctx context.Context, p *prober) error {
			return p.expectAdmitted(ctx, probe{method: http.MethodPost, origin: p.cfg.Origin, mvepAuth: p.cfg.Token})
		}},
	}
}

// Check runs every rule as a subtest of t.
func Check(t *testing.T, cfg Config) {
	t.Helper()
	p, err := newProber(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rules() {
		t.Run(r.name, func(t *testing.T) {
			if err := r.check(t.Context(), p); err != nil {
				t.Error(err)
			}
		})
	}
}

// Verify runs every rule and returns the failures joined, or nil.
func Verify(ctx context.Context, cfg Config) error {
	p, err := newProber(cfg)
	if err != nil {
		return err
	}
	var errs []error
	for _, r := range rules() {
		if err := r.check(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, err))
		}
	}
	return errors.Join(errs...)
}

type prober struct {
	cfg    Config
	target *url.URL
	client *http.Client
}

func newProber(cfg Config) (*prober, error) {
	if cfg.URL == "" || cfg.Token == "" || cfg.Origin == "" {
		return nil, errors.New("webviewtest: Config.URL, Token and Origin are required")
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("webviewtest: invalid URL %q", cfg.URL)
	}
	path := cfg.Path
	if path == "" {
		path = "/"
	}
	u = u.JoinPath(path)
	c := cfg.Client
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	return &prober{cfg: cfg, target: u, client: c}, nil
}

// foreignHost keeps the listener's port so only the name differs.
func (p *prober) foreignHost() string {
	if port := p.target.Port(); port != "" {
		return ForeignHost + ":" + port
	}
	return ForeignHost
}

type probe struct {
	method    string
	host      string
	origin    string
	token     string // sent as Authorization: Bearer
	mvepAuth  string // sent as x-mvep-auth
	preflight bool
}

type result struct {
	status int
	header http.Header
}

func (p *prober) send(ctx context.Context, pr probe) (*result, error) {
	var body io.Reader
	if pr.method == http.MethodPost {
		body = strings.NewReader("{}")
	}
	req, err := http.NewRequestWithContext(ctx, pr.method, p.target.String(), body)
	if err != nil {
		return nil, err
	}
	if pr.host != "" {
		req.Host = pr.host
	}
	if pr.origin != "" {
		req.Header.Set("Origin", pr.origin)
	}
	if pr.preflight {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "authorization, content-type, x-mainvec-cmd")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	if pr.token != "" {
		req.Header.Set("Authorization", "Bearer "+pr.token)
	}
	if pr.mvepAuth != "" {
		req.Header.Set("x-mvep-auth", pr.mvepAuth)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return &result{status: resp.StatusCode, header: resp.Header}, nil
}

// refused reports whether status is a refusal.
func refused(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusMisdirectedRequest
}

func (p *prober) expectRefused(ctx context.Context, pr probe) error {
	res, err := p.send(ctx, pr)
	if err != nil {
		return err
	}
	if !refused(res.status) {
		return fmt.Errorf("got status %d, want 401, 403 or 421", res.status)
	}
	if acao := res.header.Get("Access-Control-Allow-Origin"); acao == "*" ||
		(acao != "" && pr.origin != p.cfg.Origin) {
		return fmt.Errorf("refusal is readable by the requesting page: Access-Control-Allow-Origin %q", acao)
	}
	if pr.host != "" {
		if acao := res.header.Get("Access-Control-Allow-Origin"); acao != "" {
			return fmt.Errorf("refusal under a foreign Host carries Access-Control-Allow-Origin %q", acao)
		}
	}
	return nil
}

func (p *prober) expectAdmitted(ctx context.Context, pr probe) error {
	res, err := p.send(ctx, pr)
	if err != nil {
		return err
	}
	if refused(res.status) {
		return fmt.Errorf("got status %d, want the request admitted", res.status)
	}
	if acao := res.header.Get("Access-Control-Allow-Origin"); acao != pr.origin {
		return fmt.Errorf("Access-Control-Allow-Origin = %q, want exactly %q", acao, pr.origin)
	}
	if !varyHasOrigin(res.header) {
		return errors.New("response lacks Vary: Origin")
	}
	return nil
}

func varyHasOrigin(h http.Header) bool {
	for _, v := range h.Values("Vary") {
		for _, f := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(f), "Origin") {
				return true
			}
		}
	}
	return false
}
