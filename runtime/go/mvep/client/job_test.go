package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/mvep/runtime/go/mvep/server"
)

// --- T7: Failing tests for SendEnvelope and job helpers ---

// clientTestPackage is a minimal Package for client tests.
type clientTestPackage struct {
	name string
	cmds map[string]struct{}
}

func newClientTestPackage(name string, cmds ...string) *clientTestPackage {
	m := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		m[c] = struct{}{}
	}
	return &clientTestPackage{name: name, cmds: m}
}

func (p *clientTestPackage) GetName() string { return p.name }

func (p *clientTestPackage) InstanceOf(cmdName string) (any, bool) {
	if _, ok := p.cmds[cmdName]; !ok {
		// Support <Cmd>Result lookups for WaitForJob.
		if strings.HasSuffix(cmdName, "Result") {
			base := strings.TrimSuffix(cmdName, "Result")
			if _, ok := p.cmds[base]; ok {
				return &struct {
					Name string `json:"name"`
				}{}, true
			}
		}
		return nil, false
	}
	return &struct {
		Name string `json:"name"`
	}{}, true
}

func (p *clientTestPackage) NameOf(cmd any) string {
	if s, ok := cmd.(*struct {
		Name string `json:"name"`
	}); ok {
		return s.Name
	}
	return ""
}

// funcRunner adapts a function to CommandRunner.
type funcRunner func(ctx context.Context, cmd any) (any, error)

func (f funcRunner) RunCmd(ctx context.Context, cmd any) (any, error) { return f(ctx, cmd) }

// newJobTestClientServer builds a started server with async jobs enabled and
// returns a connected client. The server is cleaned up automatically.
func newJobTestClientServer(t *testing.T, pkg mvep.Package, runner mvep.CommandRunner) (*server.Server, *Client, *PackageClient) {
	t.Helper()

	svr, err := server.NewServer(&server.ServerConfig{
		Listeners:         []server.ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		EnableAsyncJobs:   true,
		MaxConcurrentJobs: 10,
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

	ln := svr.GetListener()
	baseURL := "http://" + ln.Addr().String()

	client, err := NewClient(ClientConfig{
		BaseURL:  baseURL,
		BasePath: "/api",
		Encoder:  "application/json",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	pkgClient, err := client.RegisterPackage(pkg)
	if err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}

	return svr, client, pkgClient
}

// TestSubmitJob_RoundTrip verifies the full submit → poll → wait cycle via the
// Go client helpers.
func TestSubmitJob_RoundTrip(t *testing.T) {
	pkg := newClientTestPackage("testpkg", "EchoCmd")
	_, _, pkgClient := newJobTestClientServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobID, err := pkgClient.SubmitJob(ctx, &struct {
		Name string `json:"name"`
	}{Name: "EchoCmd"}, nil)
	if err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}
	if jobID == "" {
		t.Fatal("SubmitJob returned empty job ID")
	}

	// Poll for status.
	status, err := pkgClient.GetJobStatus(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if status == nil {
		t.Fatal("GetJobStatus returned nil")
	}
	if status.Status != string(mvep.JobSucceeded) && status.Status != string(mvep.JobRunning) && status.Status != string(mvep.JobPending) {
		t.Errorf("initial status = %q", status.Status)
	}

	// WaitForJob should return the result.
	result, err := pkgClient.WaitForJob(ctx, jobID, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForJob: %v", err)
	}
	if result == nil {
		t.Fatal("WaitForJob returned nil result")
	}
}

// TestGetJobStatus_FailedJobReturnsResultNotError verifies that polling a
// failed job returns a populated JobStatusResult with a nil Go error.
func TestGetJobStatus_FailedJobReturnsResultNotError(t *testing.T) {
	pkg := newClientTestPackage("testpkg", "FailCmd")
	_, _, pkgClient := newJobTestClientServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return nil, errors.New("command failed")
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobID, err := pkgClient.SubmitJob(ctx, &struct {
		Name string `json:"name"`
	}{Name: "FailCmd"}, nil)
	if err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}

	// Wait for the job to fail.
	deadline := time.Now().Add(3 * time.Second)
	var status *mvep.JobStatusResult
	for time.Now().Before(deadline) {
		status, err = pkgClient.GetJobStatus(ctx, jobID)
		if err != nil {
			t.Fatalf("GetJobStatus: %v", err)
		}
		if status.Status == string(mvep.JobFailed) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status == nil || status.Status != string(mvep.JobFailed) {
		t.Fatalf("job did not reach failed state; last status = %v", status)
	}
	// A failed job returns a non-nil result and a nil error — the query succeeded.
	if status.Error == nil {
		t.Error("failed job should have a non-nil Error in JobStatusResult")
	}
	if status.Error.Code == "" {
		t.Error("failed job Error.Code should be non-empty")
	}
}

// TestSendEnvelope_RunsClientInterceptors verifies that SendEnvelope goes
// through the client interceptor chain, so job submissions get the same
// interceptor coverage as regular commands.
func TestSendEnvelope_RunsClientInterceptors(t *testing.T) {
	pkg := newClientTestPackage("testpkg", "RealCmd")

	svr, err := server.NewServer(&server.ServerConfig{
		Listeners:         []server.ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		EnableAsyncJobs:   true,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := svr.RegisterPackage(pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	})); err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}
	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })

	var interceptorCalls int64
	clientInterceptor := func(ctx context.Context, req *mvep.CmdReq, invoker mvep.ClientInvoker) (*mvep.CmdResp, error) {
		atomic.AddInt64(&interceptorCalls, 1)
		return invoker(ctx, req)
	}

	ln := svr.GetListener()
	baseURL := "http://" + ln.Addr().String()
	client, err := NewClient(ClientConfig{
		BaseURL:     baseURL,
		BasePath:    "/api",
		Encoder:     "application/json",
		Interceptor: clientInterceptor,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	pkgClient, err := client.RegisterPackage(pkg)
	if err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}

	req := mvep.NewCmdReq(mvep.SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pkgClient.SendEnvelope(ctx, req)
	if err != nil {
		t.Fatalf("SendEnvelope: %v", err)
	}
	if resp == nil {
		t.Fatal("SendEnvelope returned nil response")
	}
	if atomic.LoadInt64(&interceptorCalls) == 0 {
		t.Error("client interceptor was not called — SendEnvelope skips the interceptor chain")
	}
}

// TestSendEnvelope_NoInterceptor verifies SendEnvelope works without any
// client interceptor configured.
func TestSendEnvelope_NoInterceptor(t *testing.T) {
	pkg := newClientTestPackage("testpkg", "RealCmd")
	_, _, pkgClient := newJobTestClientServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := mvep.NewCmdReq(mvep.SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"

	resp, err := pkgClient.SendEnvelope(ctx, req)
	if err != nil {
		t.Fatalf("SendEnvelope: %v", err)
	}
	if resp == nil {
		t.Fatal("SendEnvelope returned nil response")
	}
	if resp.HasError() {
		t.Fatalf("SendEnvelope error: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]
	if jobID == "" {
		t.Fatal("SendEnvelope did not return job-id")
	}
}

// TestGetJobStatus_NotFound verifies a missing job returns an error, not a
// result with a failed status.
func TestGetJobStatus_NotFound(t *testing.T) {
	pkg := newClientTestPackage("testpkg", "RealCmd")
	_, _, pkgClient := newJobTestClientServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pkgClient.GetJobStatus(ctx, "nonexistent")
	if err == nil {
		t.Fatal("GetJobStatus on missing job: expected error, got nil")
	}
}

// Ensure unused imports don't cause errors.
var _ = httptest.NewRecorder
var _ = http.MethodGet
var _ = strings.NewReader
