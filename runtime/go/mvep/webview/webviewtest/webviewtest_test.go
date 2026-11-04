package webviewtest_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/mvep/runtime/go/mvep/client"
	"github.com/mainvec/mvep/runtime/go/mvep/server"
	"github.com/mainvec/mvep/runtime/go/mvep/webview"
	"github.com/mainvec/mvep/runtime/go/mvep/webview/webviewtest"
)

const appOrigin = "wails://wails"

// echoPackage is a one-command mvep package: EchoCmd{Msg} → EchoCmdResult{Echo}.
type echoPackage struct{}

type EchoCmd struct {
	Msg string `json:"msg"`
}

type EchoCmdResult struct {
	Echo string `json:"echo"`
}

func (echoPackage) GetName() string { return "echo" }

func (echoPackage) InstanceOf(name string) (any, bool) {
	switch name {
	case "EchoCmd":
		return &EchoCmd{}, true
	case "EchoCmdResult":
		return &EchoCmdResult{}, true
	}
	return nil, false
}

func (echoPackage) NameOf(cmd any) string {
	if _, ok := cmd.(*EchoCmd); ok {
		return "EchoCmd"
	}
	return ""
}

type echoRunner struct{}

func (echoRunner) RunCmd(_ context.Context, cmd any) (any, error) {
	return &EchoCmdResult{Echo: cmd.(*EchoCmd).Msg}, nil
}

// tokenValidator accepts exactly one token.
type tokenValidator string

func (v tokenValidator) Validate(_ context.Context, token string) (any, error) {
	if token != string(v) {
		return nil, errors.New("bad token")
	}
	return "user", nil
}

// startDaemon starts an mvep server for echoPackage that requires daemonToken.
func startDaemon(t *testing.T, ln server.ListenerConfig, daemonToken string) *server.Server {
	t.Helper()
	svr, err := server.NewServer(&server.ServerConfig{
		Listeners:   []server.ListenerConfig{ln},
		BasePath:    "/api",
		Interceptor: mvep.AuthInterceptor(tokenValidator(daemonToken)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svr.RegisterPackage(echoPackage{}, echoRunner{}); err != nil {
		t.Fatal(err)
	}
	if err := svr.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })
	return svr
}

// echo calls EchoCmd at baseURL with token as x-mvep-auth.
func echo(t *testing.T, baseURL, token string) (string, error) {
	t.Helper()
	c, err := client.NewClient(client.ClientConfig{
		BaseURL:     baseURL,
		BasePath:    "/api",
		Encoder:     "application/json",
		Interceptor: mvep.StaticAuthHeaderInterceptor(token),
	})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := c.RegisterPackage(echoPackage{})
	if err != nil {
		t.Fatal(err)
	}
	res, resp, err := pc.SendCmdReq(context.Background(), &EchoCmd{Msg: "hi"}, nil)
	if err != nil {
		return "", err
	}
	if resp != nil && resp.HasError() {
		return "", errors.New(resp.Error.Code + ": " + resp.Error.Message)
	}
	return res.(*EchoCmdResult).Echo, nil
}

func TestProxySetup(t *testing.T) {
	const daemonToken = "daemon-secret"
	daemon := startDaemon(t, server.ListenerConfig{Address: "127.0.0.1:0"}, daemonToken)

	p, err := webview.Start(webview.ProxyOptions{
		Daemon:      "http://" + daemon.GetListener().Addr().String(),
		DaemonToken: daemonToken,
		AllowOrigin: webview.WailsOrigin,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })

	webviewtest.Check(t, webviewtest.Config{URL: p.URL, Token: p.Token, Origin: appOrigin, Path: "/api/echo/cmd"})

	if got, err := echo(t, p.URL, p.Token); err != nil || got != "hi" {
		t.Errorf("echo through proxy with session token = %q, %v; want hi", got, err)
	}
	if _, err := echo(t, p.URL, daemonToken); err == nil {
		t.Error("the daemon token was accepted by the proxy; only the session token may be")
	}
}

func TestInProcessSetup(t *testing.T) {
	token := webview.NewToken()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	startDaemon(t, server.ListenerConfig{
		Listener: ln,
		Middleware: func(h http.Handler) http.Handler {
			return webview.Guard(h, webview.GuardOptions{Token: token, AllowOrigin: webview.WailsOrigin})
		},
	}, token)
	url := "http://" + ln.Addr().String()

	webviewtest.Check(t, webviewtest.Config{URL: url, Token: token, Origin: appOrigin, Path: "/api/echo/cmd"})

	if got, err := echo(t, url, token); err != nil || got != "hi" {
		t.Errorf("echo through in-process guard = %q, %v; want hi", got, err)
	}
}

func TestVerifyFailsOnUnguardedListener(t *testing.T) {
	open := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
	}))
	defer open.Close()

	err := webviewtest.Verify(context.Background(), webviewtest.Config{URL: open.URL, Token: "t", Origin: appOrigin})
	if err == nil {
		t.Fatal("Verify passed an unguarded listener")
	}
	for _, rule := range []string{"no token is refused", "foreign Origin is refused", "foreign Host is refused", "own origin with Bearer token is admitted"} {
		if !strings.Contains(err.Error(), rule) {
			t.Errorf("Verify error does not report %q:\n%v", rule, err)
		}
	}
}

func TestVerifyRequiresConfig(t *testing.T) {
	if err := webviewtest.Verify(context.Background(), webviewtest.Config{URL: "http://127.0.0.1:1"}); err == nil {
		t.Error("Verify accepted a Config without Token and Origin")
	}
}
