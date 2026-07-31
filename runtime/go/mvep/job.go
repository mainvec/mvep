package mvep

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Job status constants.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
)

// Reserved built-in command names.
const (
	SubmitJobName    = "SubmitJob"
	GetJobStatusName = "GetJobStatus"
)

// MaxJobMessageBytes bounds the length of job-progress-message and
// job-error-message header values.
const MaxJobMessageBytes = 512

// DefaultMaxJobResultBytes bounds a completed job's retained result.
const DefaultMaxJobResultBytes int64 = 4 << 20 // 4 MiB

// JobProgress holds incremental progress reported by a running command.
type JobProgress struct {
	Percent int
	Message string
}

// Job is the internal representation of a background job. It never crosses
// the wire as a typed struct.
type Job struct {
	ID            string
	Cmd           string
	Encoder       string // submit-time mime type; replayed on execute and on GET /jobs/{id}
	Status        JobStatus
	Progress      *JobProgress
	Headers       map[string]string // submitter's headers, replayed on the inner request
	Payload       []byte            // submitter's verbatim encoded inner command
	CreatedAt     time.Time
	StartedAt     time.Time
	CompletedAt   time.Time
	ResultPayload []byte // inner command's verbatim encoded result
	Error         *ErrorInfo
}

// JobStore is a pluggable interface for job persistence. Implementations must
// be safe for concurrent use.
type JobStore interface {
	Create(ctx context.Context, job *Job) error
	Get(ctx context.Context, jobID string) (*Job, bool, error)
	MarkRunning(ctx context.Context, jobID string, startedAt time.Time) error
	MarkSucceeded(ctx context.Context, jobID string, payload []byte, completedAt time.Time) error
	MarkFailed(ctx context.Context, jobID string, errInfo *ErrorInfo, completedAt time.Time) error
	SetProgress(ctx context.Context, jobID string, progress *JobProgress) error
}

// InMemoryJobStore is the default JobStore: a map guarded by a RWMutex.
// It sweeps completed entries older than retention opportunistically on writes.
type InMemoryJobStore struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	retention time.Duration
}

// NewInMemoryJobStore creates a new in-memory store with the given retention.
// retention <= 0 disables sweeping.
func NewInMemoryJobStore(retention time.Duration) *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs:      make(map[string]*Job),
		retention: retention,
	}
}

func (s *InMemoryJobStore) Create(_ context.Context, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	s.maybeSweepLocked()
	// Store a shallow copy so the caller's *Job pointer is not aliased with
	// the store's internal entry — matching the defensive-copy contract Get
	// already upholds. Headers and Payload are shared (read-only in practice),
	// but struct-level mutations by the caller won't leak into the store.
	copied := *job
	s.jobs[job.ID] = &copied
	return nil
}

func (s *InMemoryJobStore) Get(_ context.Context, jobID string) (*Job, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, false, nil
	}
	// Return a shallow copy so callers can't race with state mutations.
	copied := *job
	return &copied, true, nil
}

func (s *InMemoryJobStore) MarkRunning(_ context.Context, jobID string, startedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}
	job.Status = JobRunning
	job.StartedAt = startedAt
	return nil
}

func (s *InMemoryJobStore) MarkSucceeded(_ context.Context, jobID string, payload []byte, completedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}
	job.Status = JobSucceeded
	job.ResultPayload = payload
	job.CompletedAt = completedAt
	return nil
}

func (s *InMemoryJobStore) MarkFailed(_ context.Context, jobID string, errInfo *ErrorInfo, completedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}
	job.Status = JobFailed
	job.Error = errInfo
	job.CompletedAt = completedAt
	return nil
}

func (s *InMemoryJobStore) SetProgress(_ context.Context, jobID string, progress *JobProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}
	job.Progress = progress
	return nil
}

// maybeSweepLocked removes completed entries older than retention. Must be
// called with s.mu held for writing. Never sweeps pending or running jobs.
func (s *InMemoryJobStore) maybeSweepLocked() {
	if s.retention <= 0 {
		return
	}
	cutoff := time.Now().Add(-s.retention)
	for id, job := range s.jobs {
		if (job.Status == JobSucceeded || job.Status == JobFailed) &&
			!job.CompletedAt.IsZero() && job.CompletedAt.Before(cutoff) {
			delete(s.jobs, id)
		}
	}
}

// GenerateJobID returns a 32-char hex-encoded random ID (16 bytes of crypto/rand).
func GenerateJobID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail in practice; fall back to a panic
		// rather than returning a predictable ID.
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// sanitizeJobHeaderValue replaces each newline (\r\n pairs, lone \r, or lone
// \n) with a single space, drops other control bytes (except tab), drops
// non-ASCII, and truncates to maxLen. HTTP header values cannot contain
// newlines and must be representable as ASCII; Go's net/http silently rewrites
// newlines to spaces, which would corrupt the value invisibly.
func sanitizeJobHeaderValue(value string, maxLen int) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	runes := []rune(value)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '\r' && i+1 < len(runes) && runes[i+1] == '\n':
			// \r\n pair → single space.
			b.WriteByte(' ')
			i++ // skip the \n
		case r == '\r' || r == '\n':
			b.WriteByte(' ')
		case r == '\t':
			b.WriteRune(r)
		case r < 0x20:
			// Drop other control bytes.
		case r > 0x7F:
			// Drop non-ASCII; not representable in an HTTP header value.
		default:
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > maxLen {
		result = result[:maxLen]
	}
	return result
}

// =============================================================================
// PackageHandler job dispatch, lazy init, and Shutdown
// =============================================================================

// jobProgressSink is stored in the job's execution context so SetJobProgress
// can report progress to the JobStore.
type jobProgressSink struct {
	jobID string
	store JobStore
}

type jobProgressSinkContextKey struct{}

// jobState holds lazily-initialized async-job infrastructure, guarded by
// jobInitOnce and jobMu.
type jobState struct {
	ctx    context.Context
	cancel context.CancelFunc
	sem    chan struct{}
	wg     sync.WaitGroup
	closed bool
}

// initJobs lazily initializes the async-job infrastructure. Must be called
// before any job operation. Safe to call multiple times.
func (h *PackageHandler) initJobs() {
	h.jobInitOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		max := h.MaxConcurrentJobs
		if max <= 0 {
			max = DefaultMaxConcurrentJobs
		}
		h.jobState = &jobState{
			ctx:    ctx,
			cancel: cancel,
			sem:    make(chan struct{}, max),
		}
	})
}

// jobStore returns the configured JobStore or a lazily-created default
// InMemoryJobStore. The default is created once via sync.Once and reused for
// the handler's lifetime so submits and polls see the same store.
func (h *PackageHandler) jobStore() JobStore {
	if h.JobStore != nil {
		return h.JobStore
	}
	h.defaultStoreOnce.Do(func() {
		retention := h.JobRetention
		if retention <= 0 {
			retention = DefaultJobRetention
		}
		h.defaultStore = NewInMemoryJobStore(retention)
	})
	return h.defaultStore
}

// maxResultBytes returns the configured result-size cap or the default.
func (h *PackageHandler) maxResultBytes() int64 {
	if h.MaxJobResultBytes > 0 {
		return h.MaxJobResultBytes
	}
	return DefaultMaxJobResultBytes
}

// handleSubmitJob handles the SubmitJob reserved command.
func (h *PackageHandler) handleSubmitJob(ctx context.Context, req *CmdReq, encoder string) *CmdResp {
	if !h.EnableAsyncJobs {
		return NewCmdRespError("unknown_command", fmt.Sprintf("unknown command %s", req.Cmd))
	}

	h.initJobs()

	jobCmd := req.Headers["job-cmd"]
	if jobCmd == "" || jobCmd == SubmitJobName || jobCmd == GetJobStatusName {
		return NewCmdRespError("nested_job_forbidden", "job-cmd is missing or refers to a reserved command")
	}

	// Verify the inner command is registered.
	if _, ok := h.Package.InstanceOf(jobCmd); !ok {
		return NewCmdRespError("unknown_command", fmt.Sprintf("unknown command %s", jobCmd))
	}

	// Non-blocking acquire on the semaphore.
	select {
	case h.jobState.sem <- struct{}{}:
	default:
		return NewCmdRespError("job_queue_full", "job concurrency limit reached")
	}

	store := h.jobStore()
	jobID := GenerateJobID()

	// Copy submitter headers, stripping job-* keys.
	jobHeaders := make(map[string]string, len(req.Headers))
	for k, v := range req.Headers {
		if strings.HasPrefix(k, "job-") {
			continue
		}
		jobHeaders[k] = v
	}

	job := &Job{
		ID:      jobID,
		Cmd:     jobCmd,
		Encoder: encoder,
		Status:  JobPending,
		Headers: jobHeaders,
		Payload: req.Payload,
	}

	if err := store.Create(ctx, job); err != nil {
		// Release the acquired semaphore slot — runJob's defer hasn't started yet.
		<-h.jobState.sem
		return NewCmdRespError("job_store_error", fmt.Sprintf("failed to create job: %v", err))
	}

	// Hold jobMu across the closed-check and wg.Add so Shutdown's wg.Wait
	// cannot return before this job is counted. Releasing between them races
	// a returned Wait — which sync.WaitGroup explicitly forbids — and lets a
	// job run after shutdown claims to have drained.
	h.jobMu.Lock()
	if h.jobState.closed {
		h.jobMu.Unlock()
		// Job was created but server is shutting down; clean up.
		<-h.jobState.sem
		return NewCmdRespError("job_store_error", "server is shutting down")
	}
	h.jobState.wg.Add(1)
	h.jobMu.Unlock()

	go h.runJob(jobID, job)

	resp := NewCmdResp(nil)
	resp.Headers["job-id"] = jobID
	resp.Headers["job-cmd"] = jobCmd
	return resp
}

// runJob executes the wrapped command in a background goroutine.
func (h *PackageHandler) runJob(jobID string, job *Job) {
	defer h.jobState.wg.Done()
	defer func() { <-h.jobState.sem }()

	store := h.jobStore()
	now := time.Now()

	if err := store.MarkRunning(context.Background(), jobID, now); err != nil {
		_ = store.MarkFailed(context.Background(), jobID, &ErrorInfo{
			Code: "job_store_error", Message: fmt.Sprintf("MarkRunning failed: %v", err),
		}, time.Now())
		return
	}

	// Build the execution context from jobCtx, not the HTTP request context.
	jobCtx := h.jobState.ctx
	if h.JobTimeout > 0 {
		var jobCancel context.CancelFunc
		jobCtx, jobCancel = context.WithTimeout(jobCtx, h.JobTimeout)
		defer jobCancel()
	}
	// Attach a progress sink.
	jobCtx = context.WithValue(jobCtx, jobProgressSinkContextKey{}, &jobProgressSink{
		jobID: jobID,
		store: store,
	})

	// Build the inner CmdReq from the stored job data.
	innerHeaders := make(map[string]string, len(job.Headers))
	for k, v := range job.Headers {
		innerHeaders[k] = v
	}
	innerReq := &CmdReq{
		Cmd:     job.Cmd,
		Headers: innerHeaders,
		Payload: job.Payload,
	}

	resp := h.ServeCmdReq(jobCtx, innerReq, job.Encoder)

	if resp.HasError() {
		_ = store.MarkFailed(context.Background(), jobID, resp.Error, time.Now())
		return
	}

	// Check result size.
	if int64(len(resp.Payload)) > h.maxResultBytes() {
		_ = store.MarkFailed(context.Background(), jobID, &ErrorInfo{
			Code:    "job_result_too_large",
			Message: fmt.Sprintf("result %d bytes exceeds limit %d", len(resp.Payload), h.maxResultBytes()),
		}, time.Now())
		return
	}

	_ = store.MarkSucceeded(context.Background(), jobID, resp.Payload, time.Now())
}

// handleGetJobStatus handles the GetJobStatus reserved command.
func (h *PackageHandler) handleGetJobStatus(ctx context.Context, req *CmdReq, encoder string) *CmdResp {
	if !h.EnableAsyncJobs {
		return NewCmdRespError("unknown_command", fmt.Sprintf("unknown command %s", req.Cmd))
	}

	h.initJobs()

	jobID := req.Headers["job-id"]
	if jobID == "" {
		return NewCmdRespError("invalid_request", "missing job-id header")
	}

	store := h.jobStore()
	job, ok, err := store.Get(ctx, jobID)
	if err != nil {
		return NewCmdRespError("job_store_error", fmt.Sprintf("store error: %v", err))
	}
	if !ok {
		return NewCmdRespError("job_not_found", fmt.Sprintf("job %s not found", jobID))
	}

	resp := NewCmdResp(nil)
	resp.Headers["job-status"] = string(job.Status)

	if job.Progress != nil {
		resp.Headers["job-progress-percent"] = fmt.Sprintf("%d", job.Progress.Percent)
		resp.Headers["job-progress-message"] = sanitizeJobHeaderValue(job.Progress.Message, MaxJobMessageBytes)
	}

	if job.Status == JobFailed && job.Error != nil {
		resp.Headers["job-error-code"] = job.Error.Code
		resp.Headers["job-error-message"] = sanitizeJobHeaderValue(job.Error.Message, MaxJobMessageBytes)
	}

	if job.Status == JobSucceeded {
		// The result payload is encoded with job.Encoder. If the poller used a
		// different encoder, serving the bytes would be a Content-Type lie —
		// ServeHTTP labels the response with the poll request's media type.
		// Reject the mismatch rather than serving mislabeled bytes.
		if encoder != job.Encoder {
			return NewCmdRespError("unsupported_media_type",
				fmt.Sprintf("job result is %s-encoded; poll with that Content-Type", job.Encoder))
		}
		resp.Payload = job.ResultPayload
		// Echo the encoder so the convenience route can set Content-Type.
		resp.Headers["job-encoder"] = job.Encoder
	}

	return resp
}

// SetJobProgress reports progress for a running job. It is a no-op when called
// outside a job context (i.e. from a synchronous command).
func SetJobProgress(ctx context.Context, percent int, message string) {
	sink, ok := ctx.Value(jobProgressSinkContextKey{}).(*jobProgressSink)
	if !ok || sink == nil {
		return
	}
	_ = sink.store.SetProgress(ctx, sink.jobID, &JobProgress{
		Percent: percent,
		Message: message,
	})
}

// Shutdown cancels the job context and waits for in-flight jobs to finish,
// bounded by ctx. It is safe to call even if no jobs were ever submitted.
func (h *PackageHandler) Shutdown(ctx context.Context) error {
	h.initJobs()

	h.jobMu.Lock()
	h.jobState.closed = true
	h.jobMu.Unlock()

	h.jobState.cancel()

	// Wait for jobs to finish, respecting ctx's deadline.
	done := make(chan struct{})
	go func() {
		h.jobState.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
