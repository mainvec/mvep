package mvep

import (
	"context"
	"strings"
	"sync"
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