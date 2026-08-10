package cliclient

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/mvep/runtime/go/mvep/cli"
	"github.com/mainvec/mvep/runtime/go/mvep/client"
	"github.com/mainvec/mvep/runtime/go/mvep/server"
)

// remoteTestPackage is a minimal mvep.Package for RemoteExecutor tests. Its
// commands are structs carrying a Name field so NameOf can identify them.
type remoteTestPackage struct {
	name string
	cmds map[string]struct{}
}

func newRemoteTestPackage(name string, cmds ...string) *remoteTestPackage {
	m := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		m[c] = struct{}{}
	}
	return &remoteTestPackage{name: name, cmds: m}
}

func (p *remoteTestPackage) GetName() string { return p.name }

func (p *remoteTestPackage) InstanceOf(cmdName string) (any, bool) {
	if _, ok := p.cmds[cmdName]; ok {
		return &remoteCmd{}, true
	}
	if strings.HasSuffix(cmdName, "Result") {
		base := strings.TrimSuffix(cmdName, "Result")
		if _, ok := p.cmds[base]; ok {
			return &remoteResult{}, true
		}
	}
	return nil, false
}

func (p *remoteTestPackage) NameOf(cmd any) string {
	if s, ok := cmd.(*remoteCmd); ok {
		return s.Name
	}
	return ""
}

type remoteCmd struct {
	Name string `json:"name"`
	Msg  string `json:"msg"`
}

type remoteResult struct {
	Echo string `json:"echo"`
}

// funcRunner adapts a function to mvep.CommandRunner for the test server.
type funcRunner func(ctx context.Context, cmd any) (any, error)

func (f funcRunner) RunCmd(ctx context.Context, cmd any) (any, error) { return f(ctx, cmd) }

// newRemoteTestServer builds a started server + connected PackageClient for
// RemoteExecutor tests. The server is cleaned up automatically.
func newRemoteTestServer(t *testing.T, pkg mvep.Package, runner mvep.CommandRunner) *client.PackageClient {
	t.Helper()

	svr, err := server.NewServer(&server.ServerConfig{
		Listeners:         []server.ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := svr.RegisterPackage(pkg, runner); err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}
	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })

	baseURL := "http://" + svr.GetListener().Addr().String()
	c, err := client.NewClient(client.ClientConfig{
		BaseURL:  baseURL,
		BasePath: "/api",
		Encoder:  "application/json",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	pkgClient, err := c.RegisterPackage(pkg)
	if err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}
	return pkgClient
}

// TestRemoteExecutorSatisfiesExecutor is a compile-time assertion that
// RemoteExecutor satisfies cli.Executor.
func TestRemoteExecutorSatisfiesExecutor(t *testing.T) {
	t.Parallel()
	var _ cli.Executor = (*RemoteExecutor)(nil)
}

// TestRemoteExecutorForwardsCommand verifies T7: RemoteExecutor forwards a
// command to the server via PackageClient.SendCmdReq and returns the result.
func TestRemoteExecutorForwardsCommand(t *testing.T) {
	pkg := newRemoteTestPackage("remotetest", "EchoCmd")
	pkgClient := newRemoteTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		c := cmd.(*remoteCmd)
		return &remoteResult{Echo: c.Msg}, nil
	}))

	ex := NewRemoteExecutor(pkgClient)
	got, err := ex.Run(context.Background(), &remoteCmd{Name: "EchoCmd", Msg: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := got.(*remoteResult)
	if !ok {
		t.Fatalf("result type = %T, want *remoteResult", got)
	}
	if r.Echo != "hello" {
		t.Errorf("Echo = %q, want %q", r.Echo, "hello")
	}
}

// TestRemoteExecutorPropagatesServerError verifies T7: a server-side error is
// returned as a typed error carrying the CmdResp.Error.Code so T14's
// exit-code classification can key on the code without string parsing.
//
// Note: the HTTP transport (http_transport.go) sets CmdResp.Error.Code to
// "http_<status>" (e.g. http_500), not the semantic code (command_error) the
// server sets in the x-mainvec-error-code header. The semantic code is
// recoverable from the header; T14 wires that up. T7 asserts the typed
// *cli.ErrorCode wrapping the wire code, whatever it is.
func TestRemoteExecutorPropagatesServerError(t *testing.T) {
	pkg := newRemoteTestPackage("remotetest", "FailCmd")
	pkgClient := newRemoteTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return nil, errors.New("deliberate failure")
	}))

	ex := NewRemoteExecutor(pkgClient)
	_, err := ex.Run(context.Background(), &remoteCmd{Name: "FailCmd"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// The error must be a *cli.ErrorCode wrapping the wire code so T14 can classify it.
	var ce *cli.ErrorCode
	if !errors.As(err, &ce) {
		t.Fatalf("error should be *cli.ErrorCode, got %T: %v", err, err)
	}
	// The wire code is http_500 (runner failures map to 500). T14 will recover the
	// semantic "command_error" from the x-mainvec-error-code header.
	if ce.Code != "http_500" {
		t.Errorf("Code = %q, want %q", ce.Code, "http_500")
	}
	if ce.Message == "" {
		t.Error("Message should be non-empty")
	}
}

// TestRemoteExecutorUnknownCommand verifies T7: an unknown command name yields
// an error carrying the server's error code, not a generic error. The server
// returns http_404 for unknown commands; the wire code is http_404.
func TestRemoteExecutorUnknownCommand(t *testing.T) {
	pkg := newRemoteTestPackage("remotetest", "RealCmd")
	pkgClient := newRemoteTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return &remoteResult{}, nil
	}))

	ex := NewRemoteExecutor(pkgClient)
	// Send a command whose Name is not registered on the package. The server
	// receives it (NameOf returns the Name field), cannot find it, and returns 404.
	_, err := ex.Run(context.Background(), &remoteCmd{Name: "GhostCmd"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ce *cli.ErrorCode
	if !errors.As(err, &ce) {
		t.Fatalf("error should be *cli.ErrorCode, got %T: %v", err, err)
	}
	if ce.Code != "http_404" {
		t.Errorf("Code = %q, want %q", ce.Code, "http_404")
	}
}

// Ensure httptest import is used (server setup may reference it indirectly).
var _ = httptest.NewServer