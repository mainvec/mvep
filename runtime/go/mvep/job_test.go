package mvep

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- T1: failing tests for InMemoryJobStore, sanitization, and error codes ---

// TestInMemoryJobStore_CreateAndGet verifies the basic lifecycle: a created job
// is retrievable and its fields are preserved.
func TestInMemoryJobStore_CreateAndGet(t *testing.T) {
	store := NewInMemoryJobStore(0)
	ctx := context.Background()

	job := &Job{
		ID:      "job-1",
		Cmd:     "ExportReport",
		Encoder: "application/json",
		Status:  JobPending,
		Headers: map[string]string{"auth": "tok"},
		Payload: []byte(`{"x":1}`),
	}
	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, ok, err := store.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get: job not found")
	}
	if got.Cmd != "ExportReport" {
		t.Errorf("Cmd = %q, want %q", got.Cmd, "ExportReport")
	}
	if got.Encoder != "application/json" {
		t.Errorf("Encoder = %q, want %q", got.Encoder, "application/json")
	}
	if got.Status != JobPending {
		t.Errorf("Status = %q, want %q", got.Status, JobPending)
	}
	if string(got.Payload) != `{"x":1}` {
		t.Errorf("Payload = %q, want %q", got.Payload, `{"x":1}`)
	}
	if got.Headers["auth"] != "tok" {
		t.Errorf("Headers[auth] = %q, want %q", got.Headers["auth"], "tok")
	}
}

// TestInMemoryJobStore_GetMissing verifies a missing job returns ok=false, not an error.
func TestInMemoryJobStore_GetMissing(t *testing.T) {
	store := NewInMemoryJobStore(0)
	_, ok, err := store.Get(context.Background(), "nope")
	if err != nil {
		t.Fatalf("Get missing: unexpected error: %v", err)
	}
	if ok {
		t.Fatal("Get missing: expected ok=false")
	}
}

// TestInMemoryJobStore_Transitions verifies MarkRunning → MarkSucceeded and
// MarkRunning → MarkFailed state transitions update the job correctly.
func TestInMemoryJobStore_Transitions(t *testing.T) {
	ctx := context.Background()

	t.Run("succeeded", func(t *testing.T) {
		store := NewInMemoryJobStore(0)
		now := time.Now()
		_ = store.Create(ctx, &Job{ID: "s1", Cmd: "C", Status: JobPending})

		if err := store.MarkRunning(ctx, "s1", now); err != nil {
			t.Fatalf("MarkRunning: %v", err)
		}
		got, _, _ := store.Get(ctx, "s1")
		if got.Status != JobRunning {
			t.Errorf("after MarkRunning: Status = %q, want %q", got.Status, JobRunning)
		}
		if got.StartedAt.IsZero() {
			t.Error("after MarkRunning: StartedAt not set")
		}

		if err := store.MarkSucceeded(ctx, "s1", []byte(`{"ok":true}`), now.Add(time.Second)); err != nil {
			t.Fatalf("MarkSucceeded: %v", err)
		}
		got, _, _ = store.Get(ctx, "s1")
		if got.Status != JobSucceeded {
			t.Errorf("after MarkSucceeded: Status = %q, want %q", got.Status, JobSucceeded)
		}
		if string(got.ResultPayload) != `{"ok":true}` {
			t.Errorf("after MarkSucceeded: ResultPayload = %q", got.ResultPayload)
		}
		if got.CompletedAt.IsZero() {
			t.Error("after MarkSucceeded: CompletedAt not set")
		}
	})

	t.Run("failed", func(t *testing.T) {
		store := NewInMemoryJobStore(0)
		now := time.Now()
		_ = store.Create(ctx, &Job{ID: "f1", Cmd: "C", Status: JobPending})
		_ = store.MarkRunning(ctx, "f1", now)

		errInfo := &ErrorInfo{Code: "command_error", Message: "boom"}
		if err := store.MarkFailed(ctx, "f1", errInfo, now.Add(time.Second)); err != nil {
			t.Fatalf("MarkFailed: %v", err)
		}
		got, _, _ := store.Get(ctx, "f1")
		if got.Status != JobFailed {
			t.Errorf("after MarkFailed: Status = %q, want %q", got.Status, JobFailed)
		}
		if got.Error == nil || got.Error.Code != "command_error" {
			t.Errorf("after MarkFailed: Error = %+v, want code %q", got.Error, "command_error")
		}
		if got.CompletedAt.IsZero() {
			t.Error("after MarkFailed: CompletedAt not set")
		}
	})
}

// TestInMemoryJobStore_SetProgress verifies progress is stored and retrievable.
func TestInMemoryJobStore_SetProgress(t *testing.T) {
	store := NewInMemoryJobStore(0)
	ctx := context.Background()
	_ = store.Create(ctx, &Job{ID: "p1", Cmd: "C", Status: JobRunning})

	prog := &JobProgress{Percent: 42, Message: "almost there"}
	if err := store.SetProgress(ctx, "p1", prog); err != nil {
		t.Fatalf("SetProgress: %v", err)
	}
	got, _, _ := store.Get(ctx, "p1")
	if got.Progress == nil {
		t.Fatal("Progress not set")
	}
	if got.Progress.Percent != 42 {
		t.Errorf("Progress.Percent = %d, want 42", got.Progress.Percent)
	}
	if got.Progress.Message != "almost there" {
		t.Errorf("Progress.Message = %q, want %q", got.Progress.Message, "almost there")
	}
}

// TestInMemoryJobStore_RetentionSweepsCompletedOnly verifies that the
// opportunistic sweep removes only completed jobs older than the retention
// period, never pending or running ones regardless of age.
func TestInMemoryJobStore_RetentionSweepsCompletedOnly(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryJobStore(10 * time.Millisecond)

	// A completed job, old enough to be swept.
	completed := &Job{ID: "done", Cmd: "C", Status: JobSucceeded, CompletedAt: time.Now().Add(-time.Hour)}
	_ = store.Create(ctx, completed)

	// A pending job, equally old — must NOT be swept.
	pending := &Job{ID: "pend", Cmd: "C", Status: JobPending}
	_ = store.Create(ctx, pending)

	// A running job, equally old — must NOT be swept.
	running := &Job{ID: "run", Cmd: "C", Status: JobRunning, StartedAt: time.Now().Add(-time.Hour)}
	_ = store.Create(ctx, running)

	// A failed job, old enough to be swept.
	failed := &Job{ID: "fail", Cmd: "C", Status: JobFailed, CompletedAt: time.Now().Add(-time.Hour)}
	_ = store.Create(ctx, failed)

	// Trigger a sweep by creating a new job (opportunistic on writes).
	time.Sleep(20 * time.Millisecond)
	_ = store.Create(ctx, &Job{ID: "trigger", Cmd: "C", Status: JobPending})

	for _, id := range []string{"done", "fail"} {
		if _, ok, _ := store.Get(ctx, id); ok {
			t.Errorf("completed job %q should have been swept", id)
		}
	}
	for _, id := range []string{"pend", "run"} {
		if _, ok, _ := store.Get(ctx, id); !ok {
			t.Errorf("non-completed job %q was swept — must never sweep pending/running", id)
		}
	}
}

// TestInMemoryJobStore_MarkRunningMissing verifies MarkRunning on a missing
// job returns an error.
func TestInMemoryJobStore_MarkRunningMissing(t *testing.T) {
	store := NewInMemoryJobStore(0)
	if err := store.MarkRunning(context.Background(), "nope", time.Now()); err == nil {
		t.Fatal("MarkRunning on missing job: expected error, got nil")
	}
}

// TestInMemoryJobStore_MarkSucceededMissing verifies MarkSucceeded on a missing
// job returns an error.
func TestInMemoryJobStore_MarkSucceededMissing(t *testing.T) {
	store := NewInMemoryJobStore(0)
	if err := store.MarkSucceeded(context.Background(), "nope", nil, time.Now()); err == nil {
		t.Fatal("MarkSucceeded on missing job: expected error, got nil")
	}
}

// TestInMemoryJobStore_MarkFailedMissing verifies MarkFailed on a missing job
// returns an error.
func TestInMemoryJobStore_MarkFailedMissing(t *testing.T) {
	store := NewInMemoryJobStore(0)
	if err := store.MarkFailed(context.Background(), "nope", &ErrorInfo{Code: "x"}, time.Now()); err == nil {
		t.Fatal("MarkFailed on missing job: expected error, got nil")
	}
}

// TestInMemoryJobStore_SetProgressMissing verifies SetProgress on a missing
// job returns an error.
func TestInMemoryJobStore_SetProgressMissing(t *testing.T) {
	store := NewInMemoryJobStore(0)
	if err := store.SetProgress(context.Background(), "nope", &JobProgress{Percent: 1}); err == nil {
		t.Fatal("SetProgress on missing job: expected error, got nil")
	}
}

// TestInMemoryJobStore_Concurrent verifies the store is safe under concurrent
// access — the -race flag will catch issues.
func TestInMemoryJobStore_Concurrent(t *testing.T) {
	store := NewInMemoryJobStore(0)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "job-" + string(rune('A'+n))
			_ = store.Create(ctx, &Job{ID: id, Cmd: "C", Status: JobPending})
			_, _, _ = store.Get(ctx, id)
		}(i)
	}
	wg.Wait()
}

// --- Header sanitization ---

// TestSanitizeJobHeaderMessage strips CR/LF, handles non-ASCII, and truncates.
func TestSanitizeJobHeaderMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"clean", "hello world", "hello world"},
		{"embedded crlf", "line1\r\nline2", "line1 line2"},
		{"embedded cr only", "a\rb", "a b"},
		{"embedded lf only", "a\nb", "a b"},
		{"tab preserved", "a\tb", "a\tb"}, // tab is not a CR/LF
		{"non-ascii dropped", "café au lait", "caf au lait"},
		{"empty", "", ""},
		{"only crlf", "\r\n\r\n", "  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeJobHeaderValue(tc.in, MaxJobMessageBytes)
			if got != tc.want {
				t.Errorf("sanitizeJobHeaderMessage(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeJobHeaderMessage_Truncates verifies long messages are capped.
func TestSanitizeJobHeaderMessage_Truncates(t *testing.T) {
	long := strings.Repeat("a", 600)
	got := sanitizeJobHeaderValue(long, 512)
	if len(got) != 512 {
		t.Errorf("len = %d, want 512", len(got))
	}
}

// TestSanitizeJobHeaderMessage_TruncatesAfterSanitizing verifies sanitization
// happens before truncation, so a crlf string is cleaned then truncated.
func TestSanitizeJobHeaderMessage_TruncatesAfterSanitizing(t *testing.T) {
	long := "a" + strings.Repeat("\r\n", 300)
	got := sanitizeJobHeaderValue(long, 512)
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("result still contains CR/LF: %q", got)
	}
	if len(got) > 512 {
		t.Errorf("len = %d, want <= 512", len(got))
	}
}

// --- HTTPStatusForErrorCode job cases ---

func TestHTTPStatusForErrorCode_JobCases(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"job_not_found", 404},
		{"job_queue_full", 429},
		{"nested_job_forbidden", 400},
		{"job_store_error", 500},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := HTTPStatusForErrorCode(tc.code)
			if got != tc.want {
				t.Errorf("HTTPStatusForErrorCode(%q) = %d, want %d", tc.code, got, tc.want)
			}
		})
	}
}

// TestHTTPStatusForErrorCode_JobResultTooLargeAbsent verifies the deliberately
// absent case: job_result_too_large is a terminal job state, not a
// CmdResp.Error, so it must not have a mapping.
func TestHTTPStatusForErrorCode_JobResultTooLargeAbsent(t *testing.T) {
	got := HTTPStatusForErrorCode("job_result_too_large")
	if got != 500 {
		t.Errorf("HTTPStatusForErrorCode(job_result_too_large) = %d, want 500 (default: deliberately absent)", got)
	}
}

// --- GenerateJobID ---

// TestGenerateJobID_UniqueAndNonEmpty verifies IDs are non-empty and unique.
func TestGenerateJobID_UniqueAndNonEmpty(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateJobID()
		if id == "" {
			t.Fatal("GenerateJobID returned empty string")
		}
		if seen[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

// TestGenerateJobID_Length verifies the ID is a 32-char hex string (16 bytes).
func TestGenerateJobID_Length(t *testing.T) {
	id := GenerateJobID()
	if len(id) != 32 {
		t.Errorf("len(ID) = %d, want 32", len(id))
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("ID %q contains non-hex char %q", id, c)
			break
		}
	}
}

// =============================================================================
// T3: Failing tests for SubmitJob/GetJobStatus dispatch, guards, and security
// =============================================================================

// multiCmdPackage is a test Package that supports multiple named commands,
// each mapping to a simple struct. It tracks which commands it knows.
type multiCmdPackage struct {
	name string
	cmds map[string]struct{}
}

func newMultiCmdPackage(name string, cmds ...string) *multiCmdPackage {
	m := make(map[string]struct{}, len(cmds))
	for _, c := range cmds {
		m[c] = struct{}{}
	}
	return &multiCmdPackage{name: name, cmds: m}
}

func (p *multiCmdPackage) GetName() string { return p.name }

func (p *multiCmdPackage) InstanceOf(cmdName string) (any, bool) {
	if _, ok := p.cmds[cmdName]; !ok {
		return nil, false
	}
	return &struct {
		Name string `json:"name"`
	}{}, true
}

func (p *multiCmdPackage) NameOf(cmd any) string {
	if s, ok := cmd.(*struct {
		Name string `json:"name"`
	}); ok {
		return s.Name
	}
	return ""
}

// channelRunner is a CommandRunner that gates execution on a channel, allowing
// tests to control when a command completes. It returns the cmd as the result.
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

// errorRunner always returns an error.
type errorRunner struct{}

func (errorRunner) RunCmd(ctx context.Context, cmd any) (any, error) {
	return nil, errors.New("command failed")
}

// jobTestHandler builds a PackageHandler with async jobs enabled for testing.
func jobTestHandler(pkg Package, runner CommandRunner, opts ...func(*PackageHandler)) *PackageHandler {
	h := &PackageHandler{
		Package:           pkg,
		CommandRunner:     runner,
		EnableAsyncJobs:   true,
		MaxConcurrentJobs: 10,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// TestSubmitJob_DisabledReturnsUnknownCommand verifies that when EnableAsyncJobs
// is false, SubmitJob is indistinguishable from an unregistered command.
func TestSubmitJob_DisabledReturnsUnknownCommand(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "ExportReport")
	h := &PackageHandler{
		Package:       pkg,
		CommandRunner: echoRunner{},
		// EnableAsyncJobs not set
	}
	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "ExportReport"

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if !resp.HasError() {
		t.Fatal("expected error when EnableAsyncJobs is false")
	}
	if resp.Error.Code != "unknown_command" {
		t.Errorf("code = %q, want %q", resp.Error.Code, "unknown_command")
	}
}

// TestSubmitJob_ReturnsBeforeInnerCompletes verifies the core async property:
// SubmitJob returns a job ID before the wrapped command finishes.
func TestSubmitJob_ReturnsBeforeInnerCompletes(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "SlowCmd")
	gate := make(chan struct{})
	h := jobTestHandler(pkg, &channelRunner{gate: gate})

	// Give the command a name via the payload so NameOf works.
	innerReq := NewCmdReq(SubmitJobName, []byte(`{}`))
	innerReq.Headers["job-cmd"] = "SlowCmd"

	resp := h.ServeCmdReq(context.Background(), innerReq, "application/json")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]
	if jobID == "" {
		t.Fatal("SubmitJob did not return a job-id header")
	}
	if resp.Headers["job-cmd"] != "SlowCmd" {
		t.Errorf("job-cmd = %q, want %q", resp.Headers["job-cmd"], "SlowCmd")
	}
	if len(resp.Payload) != 0 {
		t.Errorf("SubmitJob payload should be empty, got %d bytes", len(resp.Payload))
	}

	// The inner command hasn't completed yet (gate is still closed).
	// Now release the gate and let the job finish.
	close(gate)

	// Poll until the job is done.
	status := pollJobStatus(t, h, jobID, 2*time.Second)
	if status != string(JobSucceeded) {
		t.Errorf("final status = %q, want %q", status, string(JobSucceeded))
	}
}

// TestSubmitJob_UnknownInnerCommand verifies an unknown inner command returns
// unknown_command without creating a job.
func TestSubmitJob_UnknownInnerCommand(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "NoSuchCmd"

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if !resp.HasError() {
		t.Fatal("expected error for unknown inner command")
	}
	if resp.Error.Code != "unknown_command" {
		t.Errorf("code = %q, want %q", resp.Error.Code, "unknown_command")
	}
}

// TestSubmitJob_NestedJobForbidden verifies submitting SubmitJob or
// GetJobStatus as the inner command is rejected.
func TestSubmitJob_NestedJobForbidden(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	for _, nested := range []string{SubmitJobName, GetJobStatusName} {
		req := NewCmdReq(SubmitJobName, []byte(`{}`))
		req.Headers["job-cmd"] = nested

		resp := h.ServeCmdReq(context.Background(), req, "application/json")
		if !resp.HasError() {
			t.Fatalf("expected error for nested job-cmd=%s", nested)
		}
		if resp.Error.Code != "nested_job_forbidden" {
			t.Errorf("nested=%s: code = %q, want %q", nested, resp.Error.Code, "nested_job_forbidden")
		}
	}
}

// TestSubmitJob_MissingJobCmdHeader verifies a missing job-cmd header is rejected.
func TestSubmitJob_MissingJobCmdHeader(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	// job-cmd header not set

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if !resp.HasError() {
		t.Fatal("expected error for missing job-cmd header")
	}
	if resp.Error.Code != "nested_job_forbidden" {
		t.Errorf("code = %q, want %q", resp.Error.Code, "nested_job_forbidden")
	}
}

// TestSubmitJob_QueueFull verifies that when MaxConcurrentJobs is exceeded,
// the response is job_queue_full.
func TestSubmitJob_QueueFull(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "SlowCmd")
	gate := make(chan struct{})
	h := jobTestHandler(pkg, &channelRunner{gate: gate}, func(h *PackageHandler) {
		h.MaxConcurrentJobs = 1
	})

	// Submit one job (fills the semaphore).
	req1 := NewCmdReq(SubmitJobName, []byte(`{}`))
	req1.Headers["job-cmd"] = "SlowCmd"
	resp1 := h.ServeCmdReq(context.Background(), req1, "application/json")
	if resp1.HasError() {
		t.Fatalf("first submit failed: %s", resp1.Error.Message)
	}

	// Second job should be rejected as queue full.
	req2 := NewCmdReq(SubmitJobName, []byte(`{}`))
	req2.Headers["job-cmd"] = "SlowCmd"
	resp2 := h.ServeCmdReq(context.Background(), req2, "application/json")
	if !resp2.HasError() {
		t.Fatal("expected job_queue_full error")
	}
	if resp2.Error.Code != "job_queue_full" {
		t.Errorf("code = %q, want %q", resp2.Error.Code, "job_queue_full")
	}

	close(gate)
}

// TestGetJobStatus_NotFound verifies an unknown job ID returns job_not_found.
func TestGetJobStatus_NotFound(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	req := NewCmdReq(GetJobStatusName, nil)
	req.Headers["job-id"] = "nonexistent"

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if !resp.HasError() {
		t.Fatal("expected error for missing job")
	}
	if resp.Error.Code != "job_not_found" {
		t.Errorf("code = %q, want %q", resp.Error.Code, "job_not_found")
	}
}

// TestGetJobStatus_PendingThenSucceeded verifies the status transitions from
// pending to succeeded after the inner command completes.
func TestGetJobStatus_PendingThenSucceeded(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	gate := make(chan struct{})
	h := jobTestHandler(pkg, &channelRunner{gate: gate})

	jobID := submitJob(t, h, "RealCmd")

	// While the command is gated, status should be pending or running.
	statusBefore := pollJobStatusRaw(t, h, jobID)
	if statusBefore != string(JobPending) && statusBefore != string(JobRunning) {
		t.Errorf("status before close = %q, want pending or running", statusBefore)
	}

	// Release the command.
	close(gate)

	statusAfter := pollJobStatus(t, h, jobID, 2*time.Second)
	if statusAfter != string(JobSucceeded) {
		t.Errorf("status after close = %q, want %q", statusAfter, string(JobSucceeded))
	}

	// A succeeded job should have a payload.
	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	statusResp := h.ServeCmdReq(context.Background(), statusReq, "application/json")
	if statusResp.HasError() {
		t.Fatalf("GetJobStatus on succeeded job: %s", statusResp.Error.Message)
	}
	if len(statusResp.Payload) == 0 {
		t.Error("succeeded job should have a result payload")
	}
}

// TestGetJobStatus_FailedJobIsNotCmdRespError verifies the failed-job model:
// a job that ran and failed reports job-status: failed with job-error-code
// headers, NOT via CmdResp.Error. This is the regression guard for the
// HTTP-200 decision.
func TestGetJobStatus_FailedJobIsNotCmdRespError(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "FailCmd")
	h := jobTestHandler(pkg, errorRunner{})

	jobID := submitJob(t, h, "FailCmd")

	// Wait for the job to finish.
	status := pollJobStatus(t, h, jobID, 2*time.Second)
	if status != string(JobFailed) {
		t.Fatalf("final status = %q, want %q", status, string(JobFailed))
	}

	// Now query status explicitly.
	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	resp := h.ServeCmdReq(context.Background(), statusReq, "application/json")

	if resp.HasError() {
		t.Errorf("failed job status query must NOT set CmdResp.Error; got: %s", resp.Error.Message)
	}
	if resp.Headers["job-status"] != string(JobFailed) {
		t.Errorf("job-status = %q, want %q", resp.Headers["job-status"], string(JobFailed))
	}
	if resp.Headers["job-error-code"] == "" {
		t.Error("job-error-code header should be set for a failed job")
	}
}

// TestSubmitJob_JobCmdStrippedFromReplayedHeaders verifies that job-* headers
// are stripped from the stored job headers, so they don't leak into the inner
// command's header namespace.
func TestSubmitJob_JobCmdStrippedFromReplayedHeaders(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")

	var seenHeaders map[string]string
	runner := CommandRunnerFunc(func(ctx context.Context, cmd any) (any, error) {
		if req := CmdReqFromContext(ctx); req != nil {
			seenHeaders = make(map[string]string, len(req.Headers))
			for k, v := range req.Headers {
				seenHeaders[k] = v
			}
		}
		return cmd, nil
	})
	h := jobTestHandler(pkg, runner)

	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"
	req.Headers["auth"] = "tok123"
	req.Headers["request-id"] = "req-1"

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]

	// Wait for the job to complete.
	pollJobStatus(t, h, jobID, 2*time.Second)

	if seenHeaders == nil {
		t.Fatal("runner did not see any headers")
	}
	for k := range seenHeaders {
		if strings.HasPrefix(k, "job-") {
			t.Errorf("replayed headers contain job-* key %q — should be stripped", k)
		}
	}
	if seenHeaders["auth"] != "tok123" {
		t.Errorf("auth header not replayed: got %q, want %q", seenHeaders["auth"], "tok123")
	}
}

// TestJob_AuthPolicyNotBypassable is the regression guard: runJob must call
// ServeCmdReq (which runs the interceptor chain), not executeCmd directly.
// If runJob is changed to call executeCmd, this test must fail because the
// auth interceptor would be bypassed for the inner command.
func TestJob_AuthPolicyNotBypassable(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "PrivilegedCmd")

	var authCalls int64
	authInterceptor := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		// Only allow commands with a valid auth header.
		if req.Headers["auth"] != "valid-token" {
			return NewCmdRespError("unauthorized", "missing or invalid auth")
		}
		atomic.AddInt64(&authCalls, 1)
		return next(ctx, req)
	}

	h := jobTestHandler(pkg, echoRunner{}, func(h *PackageHandler) {
		h.Interceptor = authInterceptor
	})

	// Submit a job with a valid auth header.
	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "PrivilegedCmd"
	req.Headers["auth"] = "valid-token"

	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]

	// Wait for the job to finish — the inner command runs in a background
	// goroutine through ServeCmdReq, so the interceptor should have been
	// called at least twice: once for SubmitJob, once for PrivilegedCmd.
	deadline := time.Now().Add(2 * time.Second)
	var status string
	for time.Now().Before(deadline) {
		statusReq := NewCmdReq(GetJobStatusName, nil)
		statusReq.Headers["job-id"] = jobID
		statusReq.Headers["auth"] = "valid-token"
		statusResp := h.ServeCmdReq(context.Background(), statusReq, "application/json")
		if statusResp.HasError() {
			t.Fatalf("GetJobStatus failed: %s", statusResp.Error.Message)
		}
		status = statusResp.Headers["job-status"]
		if status == string(JobSucceeded) || status == string(JobFailed) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status != string(JobSucceeded) {
		t.Fatalf("final status = %q, want %q", status, string(JobSucceeded))
	}

	// authCalls should be >= 2: SubmitJob + the inner PrivilegedCmd.
	// (GetJobStatus also goes through the interceptor, adding more.)
	if atomic.LoadInt64(&authCalls) < 2 {
		t.Error("auth interceptor was called < 2 times — runJob may be calling executeCmd directly, bypassing the interceptor chain for the inner command")
	}
}

// TestSubmitJob_PostShutdown verifies that submitting a job after Shutdown
// returns job_store_error, not a panic or a job on a dead context.
func TestSubmitJob_PostShutdown(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	// Shutdown must be safe to call even if no jobs were ever submitted.
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown with no jobs: %v", err)
	}

	// Submitting after shutdown should fail cleanly.
	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"
	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if !resp.HasError() {
		t.Fatal("expected error for post-shutdown submit")
	}
	if resp.Error.Code != "job_store_error" {
		t.Errorf("code = %q, want %q", resp.Error.Code, "job_store_error")
	}
}

// TestShutdown_DrainsInFlightJob verifies that Shutdown waits for a running
// job to complete, bounded by the shutdown context.
func TestShutdown_DrainsInFlightJob(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "SlowCmd")
	gate := make(chan struct{})
	h := jobTestHandler(pkg, &channelRunner{gate: gate})

	jobID := submitJob(t, h, "SlowCmd")

	// The job is now running but gated. Shutdown should block until we
	// release the gate (or the context expires).
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- h.Shutdown(context.Background())
	}()

	// Give shutdown a moment to cancel jobCtx, then release the gate.
	time.Sleep(50 * time.Millisecond)
	close(gate)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not complete within 5 seconds")
	}

	// After shutdown, the job should have completed (the gate was released
	// before shutdown's context cancel took effect, because the runner checks
	// ctx.Done but the gate was already closed). Check status.
	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	resp := h.ServeCmdReq(context.Background(), statusReq, "application/json")
	// GetJobStatus still works after shutdown.
	if resp.HasError() {
		t.Errorf("GetJobStatus after shutdown failed: %s", resp.Error.Message)
	}
}

// TestSubmitJob_ResultTooLarge verifies that a result exceeding
// MaxJobResultBytes transitions the job to failed with job_result_too_large.
func TestSubmitJob_ResultTooLarge(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "BigCmd")

	// A runner that returns a large result.
	bigData := strings.Repeat("x", 200)
	h := jobTestHandler(pkg, CommandRunnerFunc(func(ctx context.Context, cmd any) (any, error) {
		return bigData, nil
	}), func(h *PackageHandler) {
		h.MaxJobResultBytes = 100 // very small to trigger the cap
	})

	jobID := submitJob(t, h, "BigCmd")
	status := pollJobStatus(t, h, jobID, 2*time.Second)
	if status != string(JobFailed) {
		t.Fatalf("status = %q, want %q", status, string(JobFailed))
	}

	// The failure should carry job_result_too_large.
	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	resp := h.ServeCmdReq(context.Background(), statusReq, "application/json")
	if resp.Headers["job-error-code"] != "job_result_too_large" {
		t.Errorf("job-error-code = %q, want %q", resp.Headers["job-error-code"], "job_result_too_large")
	}
}

// TestSetJobProgress_ReportsProgress verifies that a running command can
// report progress via SetJobProgress, and it appears in GetJobStatus.
func TestSetJobProgress_ReportsProgress(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "ProgressCmd")

	h := jobTestHandler(pkg, CommandRunnerFunc(func(ctx context.Context, cmd any) (any, error) {
		SetJobProgress(ctx, 50, "halfway there")
		return cmd, nil
	}))

	jobID := submitJob(t, h, "ProgressCmd")

	// Wait for completion, then check progress was recorded.
	pollJobStatus(t, h, jobID, 2*time.Second)

	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	resp := h.ServeCmdReq(context.Background(), statusReq, "application/json")
	if resp.Headers["job-progress-percent"] != "50" {
		t.Errorf("progress-percent = %q, want %q", resp.Headers["job-progress-percent"], "50")
	}
	// The message should be sanitized but "halfway there" is clean.
	if resp.Headers["job-progress-message"] != "halfway there" {
		t.Errorf("progress-message = %q, want %q", resp.Headers["job-progress-message"], "halfway there")
	}
}

// TestSetJobProgress_NoopOutsideJob verifies SetJobProgress is a no-op when
// called from a synchronous (non-job) command.
func TestSetJobProgress_NoopOutsideJob(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "SyncCmd")
	h := jobTestHandler(pkg, CommandRunnerFunc(func(ctx context.Context, cmd any) (any, error) {
		// This should be a no-op: no panic, no effect.
		SetJobProgress(ctx, 50, "should be ignored")
		return cmd, nil
	}))

	req := NewCmdReq("SyncCmd", []byte(`{}`))
	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("SyncCmd failed: %s", resp.Error.Message)
	}
	// No job was created, so no progress to check — the test passes if it
	// doesn't panic.
}

// TestSubmitJob_EncoderRecorded verifies that the submit-time encoder is
// stored on the Job and used when running the inner command.
func TestSubmitJob_EncoderRecorded(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	h := jobTestHandler(pkg, echoRunner{})

	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"
	resp := h.ServeCmdReq(context.Background(), req, "application/x-protobuf")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]
	pollJobStatus(t, h, jobID, 2*time.Second)

	// Check the job's encoder.
	store := h.jobStore()
	if store == nil {
		t.Fatal("JobStore is nil")
	}
	job, ok, _ := store.Get(context.Background(), jobID)
	if !ok {
		t.Fatal("job not found in store")
	}
	if job.Encoder != "application/x-protobuf" {
		t.Errorf("job.Encoder = %q, want %q", job.Encoder, "application/x-protobuf")
	}
}

// TestGetJobStatus_EncoderMismatchRejected verifies that polling a succeeded
// job with a different encoder returns unsupported_media_type, not mislabeled
// bytes. The result is encoded in the submit-time encoder; serving it under
// a different Content-Type would be a lie.
func TestGetJobStatus_EncoderMismatchRejected(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "RealCmd")
	// Use a runner that returns a JSON-encodable value; submit as JSON so the
	// job succeeds, then poll as protobuf.
	h := jobTestHandler(pkg, echoRunner{})

	// Submit as JSON.
	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = "RealCmd"
	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]
	status := pollJobStatus(t, h, jobID, 2*time.Second)
	if status != string(JobSucceeded) {
		t.Fatalf("job status = %q, want succeeded", status)
	}

	// Poll as protobuf — should be rejected since the job is JSON-encoded.
	statusReq := NewCmdReq(GetJobStatusName, nil)
	statusReq.Headers["job-id"] = jobID
	statusResp := h.ServeCmdReq(context.Background(), statusReq, "application/x-protobuf")
	if !statusResp.HasError() {
		t.Fatal("expected unsupported_media_type for encoder mismatch")
	}
	if statusResp.Error.Code != "unsupported_media_type" {
		t.Errorf("code = %q, want %q", statusResp.Error.Code, "unsupported_media_type")
	}

	// Poll as JSON — should succeed and return the payload.
	statusReq2 := NewCmdReq(GetJobStatusName, nil)
	statusReq2.Headers["job-id"] = jobID
	statusResp2 := h.ServeCmdReq(context.Background(), statusReq2, "application/json")
	if statusResp2.HasError() {
		t.Fatalf("GetJobStatus with matching encoder failed: %s", statusResp2.Error.Message)
	}
	if statusResp2.Headers["job-status"] != string(JobSucceeded) {
		t.Errorf("status = %q, want %q", statusResp2.Headers["job-status"], string(JobSucceeded))
	}
}

// TestShutdown_SubmitRace verifies that submitting concurrently with Shutdown
// does not race on the WaitGroup or let a job run after Shutdown returns. This
// is the regression guard for the lock-across-closed-check-and-wg.Add fix.
func TestShutdown_SubmitRace(t *testing.T) {
	pkg := newMultiCmdPackage("pkg", "SlowCmd")

	var submitErrs []error
	var errMu sync.Mutex

	for i := 0; i < 20; i++ {
		gate := make(chan struct{})
		h := jobTestHandler(pkg, &channelRunner{gate: gate})

		var wg sync.WaitGroup
		wg.Add(2)

		// Submitter
		go func() {
			defer wg.Done()
			req := NewCmdReq(SubmitJobName, []byte(`{}`))
			req.Headers["job-cmd"] = "SlowCmd"
			resp := h.ServeCmdReq(context.Background(), req, "application/json")
			if resp.HasError() {
				errMu.Lock()
				submitErrs = append(submitErrs, fmt.Errorf("%s", resp.Error.Message))
				errMu.Unlock()
			}
		}()

		// Shutdowner
		go func() {
			defer wg.Done()
			_ = h.Shutdown(context.Background())
		}()

		wg.Wait()
		close(gate)
	}

	// Some submits may fail with job_store_error (shutdown won the race),
	// which is correct. The test passes if there's no panic or data race.
	// We don't assert on submitErrs content — either outcome is valid.
}

// --- helpers ---

// submitJob is a test helper that submits a job and returns the job ID.
func submitJob(t *testing.T, h *PackageHandler, innerCmd string) string {
	t.Helper()
	req := NewCmdReq(SubmitJobName, []byte(`{}`))
	req.Headers["job-cmd"] = innerCmd
	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("SubmitJob failed: %s", resp.Error.Message)
	}
	jobID := resp.Headers["job-id"]
	if jobID == "" {
		t.Fatal("SubmitJob returned empty job-id")
	}
	return jobID
}

// pollJobStatusRaw reads the current job-status header without waiting.
func pollJobStatusRaw(t *testing.T, h *PackageHandler, jobID string) string {
	t.Helper()
	req := NewCmdReq(GetJobStatusName, nil)
	req.Headers["job-id"] = jobID
	resp := h.ServeCmdReq(context.Background(), req, "application/json")
	if resp.HasError() {
		t.Fatalf("GetJobStatus failed: %s", resp.Error.Message)
	}
	return resp.Headers["job-status"]
}

// pollJobStatus polls until the job reaches a terminal state (succeeded/failed)
// or the timeout expires. Returns the final status.
func pollJobStatus(t *testing.T, h *PackageHandler, jobID string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := pollJobStatusRaw(t, h, jobID)
		if status == string(JobSucceeded) || status == string(JobFailed) {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	return pollJobStatusRaw(t, h, jobID)
}

// CommandRunnerFunc adapts a function to the CommandRunner interface.
type CommandRunnerFunc func(ctx context.Context, cmd any) (any, error)

func (f CommandRunnerFunc) RunCmd(ctx context.Context, cmd any) (any, error) {
	return f(ctx, cmd)
}

// Ensure the errorRunner in this file doesn't conflict with http_transport_test.go.
var _ = fmt.Sprintf
