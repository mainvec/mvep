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
	SubmitJobName     = "SubmitJob"
	GetJobStatusName  = "GetJobStatus"
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
	StartedAt    time.Time
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
	s.jobs[job.ID] = job
	return nil
}

func (s *InMemoryJobStore) Get(_ context.Context, jobID string) (*Job, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	return job, ok, nil
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