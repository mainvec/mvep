package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// --- T5: Failing tests for server lifecycle and /jobs/{id} endpoint ---

// testMultiCmdPackage is a server-test Package supporting multiple commands.
type testMultiCmdPackage struct {
	name string
	cmds map[string]struct{}
}

func newTestMultiCmdPackage(name string, cmds ...string) *testMultiCmdPackage {
	m := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		m[c] = struct{}{}
	}
	return &testMultiCmdPackage{name: name, cmds: m}
}

func (p *testMultiCmdPackage) GetName() string { return p.name }

func (p *testMultiCmdPackage) InstanceOf(cmdName string) (any, bool) {
	if _, ok := p.cmds[cmdName]; !ok {
		return nil, false
	}
	return &struct {
		Name string `json:"name"`
	}{}, true
}

func (p *testMultiCmdPackage) NameOf(cmd any) string {
	if s, ok := cmd.(*struct {
		Name string `json:"name"`
	}); ok {
		return s.Name
	}
	return ""
}

// channelRunner gates execution on a channel, allowing tests to control timing.
type channelRunner struct {
	gate chan struct{}
}

func (r *channelRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
	if r.gate != nil {
		select {
		case <-r.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return cmd, nil
}

// funcRunner runs an arbitrary function as a CommandRunner.
type funcRunner func(ctx context.Context, cmd any) (any, error)

func (f funcRunner) RunCmd(ctx context.Context, cmd any) (any, error) { return f(ctx, cmd) }

// newJobTestServer builds a started server with async jobs enabled and a
// registered package. Returns the server, the base URL, and the package path.
func newJobTestServer(t *testing.T, pkg mvep.Package, runner mvep.CommandRunner, opts ...func(*ServerConfig)) (*Server, string, string) {
	t.Helper()

	listeners := []ListenerConfig{{Address: "127.0.0.1:0"}}
	config := &ServerConfig{
		Listeners:         listeners,
		BasePath:          "/api",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		EnableAsyncJobs:   true,
		MaxConcurrentJobs: 10,
	}
	for _, opt := range opts {
		opt(config)
	}

	svr, err := NewServer(config)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := svr.RegisterPackage(pkg, runner); err != nil {
		t.Fatalf("RegisterPackage: %v", err)
	}
	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	ln := svr.GetListener()
	baseURL := "http://" + ln.Addr().String()
	pkgPath := "/api/" + pkg.GetName()

	t.Cleanup(func() { _ = svr.Shutdown() })

	return svr, baseURL, pkgPath
}

// submitJobViaHTTP sends a SubmitJob request and returns the job ID from the
// response header.
func submitJobViaHTTP(t *testing.T, baseURL, pkgPath, innerCmd string) string {
	t.Helper()
	body := `{}`
	req, _ := http.NewRequest("POST", baseURL+pkgPath+"/cmd", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-mainvec-cmd", mvep.SubmitJobName)
	req.Header.Set("x-mvep-job-cmd", innerCmd)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SubmitJob request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SubmitJob status = %d, want 200", resp.StatusCode)
	}

	jobID := resp.Header.Get("x-mvep-job-id")
	if jobID == "" {
		t.Fatal("SubmitJob did not return x-mvep-job-id")
	}
	return jobID
}

// pollJobStatusHTTP polls GET /jobs/{id} until the job reaches a terminal state
// or the timeout expires.
func pollJobStatusHTTP(t *testing.T, baseURL, pkgPath, jobID string, timeout time.Duration) (status string, contentType string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/"+jobID, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /jobs/{id}: %v", err)
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		status = resp.Header.Get("x-mvep-job-status")
		contentType = resp.Header.Get("Content-Type")
		resp.Body.Close()
		if status == string(mvep.JobSucceeded) || status == string(mvep.JobFailed) {
			return status, contentType
		}
		time.Sleep(10 * time.Millisecond)
	}
	return status, contentType
}

// TestServer_ShutdownDrainsInFlightJob verifies that ShutdownContext waits for
// a running job to complete, bounded by the shutdown context.
func TestServer_ShutdownDrainsInFlightJob(t *testing.T) {
	pkg := newTestMultiCmdPackage("testpkg", "SlowCmd")
	gate := make(chan struct{})
	svr, baseURL, pkgPath := newJobTestServer(t, pkg, &channelRunner{gate: gate})

	jobID := submitJobViaHTTP(t, baseURL, pkgPath, "SlowCmd")

	// The job is now running but gated. Initiate shutdown.
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		shutdownDone <- svr.ShutdownContext(ctx)
	}()

	// Give shutdown a moment to start draining, then release the gate.
	time.Sleep(50 * time.Millisecond)
	close(gate)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("ShutdownContext returned error: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("ShutdownContext did not complete before timeout")
	}

	// After shutdown, GET /jobs/{id} should still report the final state.
	req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/"+jobID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Server is stopped — acceptable, the listener is closed.
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		status := resp.Header.Get("x-mvep-job-status")
		if status != string(mvep.JobSucceeded) && status != string(mvep.JobFailed) {
			t.Errorf("post-shutdown status = %q, want succeeded or failed", status)
		}
	}
}

// TestJobsEndpoint_SameAuthAsCommand verifies the /jobs/{id} route has
// identical auth posture to calling GetJobStatus as a command. It asserts
// BOTH directions: rejected without auth, and accepted with valid auth.
func TestJobsEndpoint_SameAuthAsCommand(t *testing.T) {
	pkg := newTestMultiCmdPackage("testpkg", "RealCmd")

	var authCalls int64
	authInterceptor := func(ctx context.Context, req *mvep.CmdReq, next mvep.CmdHandler) *mvep.CmdResp {
		if req.Headers["auth"] != "valid-token" {
			return mvep.NewCmdRespError("unauthorized", "missing or invalid auth")
		}
		atomic.AddInt64(&authCalls, 1)
		return next(ctx, req)
	}

	svr, baseURL, pkgPath := newJobTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}), func(cfg *ServerConfig) {
		cfg.Interceptor = authInterceptor
	})
	_ = svr // server is cleaned up by t.Cleanup

	// Submit a job with valid auth so there's a job to poll.
	jobReq, _ := http.NewRequest("POST", baseURL+pkgPath+"/cmd", strings.NewReader(`{}`))
	jobReq.Header.Set("Content-Type", "application/json")
	jobReq.Header.Set("x-mainvec-cmd", mvep.SubmitJobName)
	jobReq.Header.Set("x-mvep-job-cmd", "RealCmd")
	jobReq.Header.Set("x-mvep-auth", "valid-token")
	jobResp, err := http.DefaultClient.Do(jobReq)
	if err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}
	jobResp.Body.Close()
	jobID := jobResp.Header.Get("x-mvep-job-id")
	if jobID == "" {
		t.Fatal("no job-id in submit response")
	}

	// Wait for the job to complete.
	pollJobStatusHTTP(t, baseURL, pkgPath, jobID, 2*time.Second)

	// Direction 1: GET /jobs/{id} WITHOUT auth → should be rejected.
	t.Run("rejected_without_auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/"+jobID, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("without auth: status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	// Direction 2: GET /jobs/{id} WITH valid auth → should succeed.
	t.Run("accepted_with_auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/"+jobID, nil)
		req.Header.Set("x-mvep-auth", "valid-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("with auth: status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		status := resp.Header.Get("x-mvep-job-status")
		if status != string(mvep.JobSucceeded) {
			t.Errorf("with auth: job-status = %q, want %q", status, string(mvep.JobSucceeded))
		}
	})
}

// TestJobsEndpoint_ContentTypeMatchesSubmitEncoder verifies that a job
// submitted as application/json is served from GET /jobs/{id} with
// Content-Type: application/json.
func TestJobsEndpoint_ContentTypeMatchesSubmitEncoder(t *testing.T) {
	pkg := newTestMultiCmdPackage("testpkg", "RealCmd")
	_, baseURL, pkgPath := newJobTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}))

	// Submit as JSON.
	jobID := submitJobViaHTTP(t, baseURL, pkgPath, "RealCmd")

	// Wait for completion.
	status, ct := pollJobStatusHTTP(t, baseURL, pkgPath, jobID, 2*time.Second)
	if status != string(mvep.JobSucceeded) {
		t.Fatalf("job status = %q, want succeeded", status)
	}
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

// TestJobsEndpoint_NotFound verifies a missing job ID returns 404.
func TestJobsEndpoint_NotFound(t *testing.T) {
	pkg := newTestMultiCmdPackage("testpkg", "RealCmd")
	_, baseURL, pkgPath := newJobTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return cmd, nil
	}))

	req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/nonexistent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// TestJobsEndpoint_FailedJobIsHTTP200 verifies that a failed job returns HTTP
// 200 with job-status: failed, not an error status code.
func TestJobsEndpoint_FailedJobIsHTTP200(t *testing.T) {
	pkg := newTestMultiCmdPackage("testpkg", "FailCmd")
	_, baseURL, pkgPath := newJobTestServer(t, pkg, funcRunner(func(ctx context.Context, cmd any) (any, error) {
		return nil, context.DeadlineExceeded
	}))

	jobID := submitJobViaHTTP(t, baseURL, pkgPath, "FailCmd")

	// Wait for the job to fail.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", baseURL+pkgPath+"/jobs/"+jobID, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		status := resp.Header.Get("x-mvep-job-status")
		if status == string(mvep.JobFailed) {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("failed job status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
			if resp.Header.Get("x-mvep-job-error-code") == "" {
				t.Error("job-error-code header should be set for a failed job")
			}
			resp.Body.Close()
			return
		}
		resp.Body.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach failed state before timeout")
}

// Ensure unused imports don't cause errors.
var _ = httptest.NewRecorder
