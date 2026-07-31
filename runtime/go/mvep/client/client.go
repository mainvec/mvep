package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
	oenc "github.com/mainvec/ugo/oencoding"
)

// DefaultEncoder is the default content type for encoding commands
const DefaultEncoder = "application/json"

// ClientConfig holds configuration for the MVP client
type ClientConfig struct {
	// BaseURL is the base URL of the MVP server (e.g., "http://127.0.0.1:8080")
	// For Unix sockets, use "unix:///path/to/socket"
	BaseURL string
	// BasePath is the base URL path for endpoints (e.g., "/api")
	BasePath string
	// Encoder is the content type for encoding commands (default: "application/json")
	Encoder string
	// Timeout is the HTTP client timeout (default: 30 seconds)
	Timeout time.Duration
	// HTTPClient allows using a custom HTTP client (optional)
	// If BaseURL is a Unix socket, this will be ignored
	HTTPClient *http.Client
	// Interceptor is the client interceptor chain for all outgoing requests
	Interceptor mvep.ClientInterceptor
}

// Client represents an MVP client for communicating with an MVP server
type Client struct {
	config      *ClientConfig
	httpClient  *http.Client
	httpBaseURL string // The actual HTTP base URL (for unix sockets this is "http://unixsocket")
	interceptor mvep.ClientInterceptor

	// mu guards packages and cmdIndex.
	mu       sync.RWMutex
	packages map[string]*PackageClient
	// cmdIndex maps a command name to its owning package for deterministic,
	// O(1) resolution in SendCmd. Built at registration time.
	cmdIndex map[string]*PackageClient
}

// PackageClient represents a client for a specific MVP package
type PackageClient struct {
	client  *Client
	pkg     mvep.Package
	handler *mvep.PackageHandler

	// encoderMu guards encoder.
	encoderMu sync.RWMutex
	encoder   string
}

// NewClient creates a new MVP client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	// if config == nil {
	// 	return nil, errors.New("config is required")
	// }
	if len(config.BaseURL) == 0 {
		return nil, errors.New("base URL is required")
	}
	if len(config.Encoder) == 0 {
		config.Encoder = DefaultEncoder
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	var httpClient *http.Client
	var httpBaseURL string

	// Check if this is a Unix socket connection
	if strings.HasPrefix(config.BaseURL, "unix://") {
		socketPath := strings.TrimPrefix(config.BaseURL, "unix://")
		dialer := &net.Dialer{}
		tr := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		}
		httpClient = &http.Client{
			Transport: tr,
			Timeout:   config.Timeout,
		}
		httpBaseURL = "http://unixsocket"
	} else {
		httpClient = config.HTTPClient
		if httpClient == nil {
			httpClient = &http.Client{
				Timeout: config.Timeout,
			}
		}
		httpBaseURL = config.BaseURL
	}

	return &Client{
		config:      &config,
		httpClient:  httpClient,
		packages:    make(map[string]*PackageClient),
		cmdIndex:    make(map[string]*PackageClient),
		httpBaseURL: httpBaseURL,
		interceptor: config.Interceptor,
	}, nil
}

// RegisterPackage registers an MVP package with the client
func (c *Client) RegisterPackage(pkg mvep.Package) (*PackageClient, error) {
	if pkg == nil {
		return nil, errors.New("package is required")
	}

	pkgName := pkg.GetName()

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.packages[pkgName]; exists {
		return nil, errors.New("package already registered: " + pkgName)
	}

	// Build the package path: basePath + "/" + packageName + "/cmd"
	pkgPath := c.config.BasePath + "/" + pkgName + "/cmd"

	transporter, err := mvep.NewHttpTransporterWithClient(c.httpBaseURL, pkgPath, c.httpClient)
	if err != nil {
		return nil, err
	}

	handler := &mvep.PackageHandler{
		Package:     pkg,
		Transporter: transporter,
	}

	pkgClient := &PackageClient{
		client:  c,
		pkg:     pkg,
		handler: handler,
		encoder: c.config.Encoder,
	}

	// Index the package's commands for deterministic resolution. Packages that
	// expose their command names via CommandLister get an O(1) index; a
	// duplicate command name across packages is an explicit error.
	if lister, ok := pkg.(mvep.CommandLister); ok {
		for _, cmdName := range lister.CommandNames() {
			if existing, dup := c.cmdIndex[cmdName]; dup {
				return nil, fmt.Errorf("command %q registered by both %q and %q", cmdName, existing.pkg.GetName(), pkgName)
			}
			c.cmdIndex[cmdName] = pkgClient
		}
	}

	c.packages[pkgName] = pkgClient

	return pkgClient, nil
}

// GetPackage returns a registered package client by name
func (c *Client) GetPackage(name string) (*PackageClient, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pkg, ok := c.packages[name]
	return pkg, ok
}

// Close closes all idle connections
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// Ping sends a health check request to the server
// It uses the configured BasePath + "/health" endpoint
func (c *Client) Ping(ctx context.Context) (string, error) {
	healthPath := c.config.BasePath + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.httpBaseURL+healthPath, nil)
	if err != nil {
		return "", err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// SendCmd sends a command to the appropriate package based on command type
// The command must be registered with one of the registered packages
func (c *Client) SendCmd(ctx context.Context, cmd any) (any, error) {
	if cmd == nil {
		return nil, errors.New("command is required")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Deterministic resolution: packages are checked in sorted-name order so a
	// command resolvable by more than one package always resolves the same way.
	names := make([]string, 0, len(c.packages))
	for name := range c.packages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pkgClient := c.packages[name]
		if cmdName := pkgClient.pkg.NameOf(cmd); cmdName != "" {
			return pkgClient.SendCmd(ctx, cmd)
		}
	}

	return nil, errors.New("no registered package found for command")
}

// SendCmd sends a command through this package client.
// Routes through the interceptor chain (auth, logging, etc.) when available.
func (p *PackageClient) SendCmd(ctx context.Context, cmd any) (any, error) {
	result, _, err := p.sendCmdReqInternal(ctx, cmd, nil, p.encoder)
	return result, err
}

// SendCmdWithEncoder sends a command with a specific encoder
func (p *PackageClient) SendCmdWithEncoder(ctx context.Context, cmd any, encoder string) (any, error) {
	return p.handler.SendCmd(ctx, cmd, encoder)
}

// SendRawCmd sends a raw command with bytes directly without encoding/decoding
// This is useful when you want to handle serialization yourself
func (p *PackageClient) SendRawCmd(ctx context.Context, cmdName string, cmdData []byte) ([]byte, error) {
	cmdDataBuf := bytes.NewBuffer(cmdData)
	respData, err := p.handler.Transporter.TransportCmd(ctx, cmdName, p.encoder, io.NopCloser(cmdDataBuf))
	if err != nil {
		return nil, err
	}
	defer respData.Close()
	respBytes, err := io.ReadAll(respData)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

// SetEncoder sets the default encoder for this package client
func (p *PackageClient) SetEncoder(encoder string) {
	p.encoderMu.Lock()
	defer p.encoderMu.Unlock()
	p.encoder = encoder
}

// GetEncoder returns the current encoder for this package client
func (p *PackageClient) GetEncoder() string {
	p.encoderMu.RLock()
	defer p.encoderMu.RUnlock()
	return p.encoder
}

// Package returns the underlying MVP package
func (p *PackageClient) Package() mvep.Package {
	return p.pkg
}

// SendCmdReq sends a command with headers and returns the result along with response headers
func (p *PackageClient) SendCmdReq(ctx context.Context, cmd any, headers map[string]string) (any, *mvep.CmdResp, error) {
	return p.sendCmdReqInternal(ctx, cmd, headers, p.encoder)
}

// SendCmdReqWithEncoder sends a command with headers using a specific encoder
func (p *PackageClient) SendCmdReqWithEncoder(ctx context.Context, cmd any, headers map[string]string, encoder string) (any, *mvep.CmdResp, error) {
	return p.sendCmdReqInternal(ctx, cmd, headers, encoder)
}

// sendCmdReqInternal handles the actual sending with optional interceptor
func (p *PackageClient) sendCmdReqInternal(ctx context.Context, cmd any, headers map[string]string, encoder string) (any, *mvep.CmdResp, error) {
	// If no interceptor, call handler directly
	if p.client.interceptor == nil {
		return p.handler.SendCmdReq(ctx, cmd, headers, encoder)
	}

	// Build a CmdReq with the command name and headers for the interceptor
	cmdName := p.pkg.NameOf(cmd)
	if cmdName == "" {
		return nil, nil, errors.New("invalid command")
	}

	req := mvep.NewCmdReq(cmdName, nil)
	if headers != nil {
		for k, v := range headers {
			req.Headers[k] = v
		}
	}

	// Variable to capture the result from the invoker
	var result any

	// Define the invoker that calls the actual handler
	invoker := func(ctx context.Context, req *mvep.CmdReq) (*mvep.CmdResp, error) {
		var resp *mvep.CmdResp
		var err error
		result, resp, err = p.handler.SendCmdReq(ctx, cmd, req.Headers, encoder)
		return resp, err
	}

	// Run through interceptor chain
	resp, err := p.client.interceptor(ctx, req, invoker)
	return result, resp, err
}

// =============================================================================
// Async job helpers (T8)
// =============================================================================

// SendEnvelope sends a raw CmdReq through the client's configured transport
// and interceptor chain, returning the raw CmdResp. It does not resolve
// command names or result types from the package registry, so it can carry
// reserved runtime commands like SubmitJob and GetJobStatus that are not
// registered package commands.
func (p *PackageClient) SendEnvelope(ctx context.Context, req *mvep.CmdReq) (*mvep.CmdResp, error) {
	if req == nil {
		return nil, errors.New("missing request")
	}

	encoder := p.GetEncoder()

	// If no interceptor, call transport directly.
	if p.client.interceptor == nil {
		return p.sendEnvelopeDirect(ctx, req, encoder)
	}

	// Run through the client interceptor chain.
	resp, err := p.client.interceptor(ctx, req, func(ctx context.Context, req *mvep.CmdReq) (*mvep.CmdResp, error) {
		return p.sendEnvelopeDirect(ctx, req, encoder)
	})
	return resp, err
}

// sendEnvelopeDirect calls the EnvelopeTransporter directly.
func (p *PackageClient) sendEnvelopeDirect(ctx context.Context, req *mvep.CmdReq, encoder string) (*mvep.CmdResp, error) {
	envTransporter, ok := p.handler.Transporter.(mvep.EnvelopeTransporter)
	if !ok {
		return nil, errors.New("transporter does not support envelope transport")
	}

	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return nil, fmt.Errorf("encoding not found, %s", encoder)
	}

	return envTransporter.TransportCmdReq(ctx, req, enc.MimeType())
}

// SubmitJob encodes cmd with the client's encoder, sends it as a SubmitJob
// request, and returns the job ID from the response header.
func (p *PackageClient) SubmitJob(ctx context.Context, cmd any, headers map[string]string) (string, error) {
	if cmd == nil {
		return "", errors.New("missing command")
	}

	encoder := p.GetEncoder()
	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return "", fmt.Errorf("encoding not found, %s", encoder)
	}

	cmdName := p.pkg.NameOf(cmd)
	if cmdName == "" {
		return "", errors.New("invalid command")
	}

	cmdBytes, err := enc.Encode(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to encode command: %w", err)
	}

	req := mvep.NewCmdReq(mvep.SubmitJobName, cmdBytes)
	if headers != nil {
		for k, v := range headers {
			req.Headers[k] = v
		}
	}
	req.Headers["job-cmd"] = cmdName

	resp, err := p.SendEnvelope(ctx, req)
	if err != nil {
		return "", fmt.Errorf("SubmitJob failed: %w", err)
	}
	if resp.HasError() {
		return "", fmt.Errorf("SubmitJob error: [%s] %s", resp.Error.Code, resp.Error.Message)
	}

	jobID := resp.Headers["job-id"]
	if jobID == "" {
		return "", errors.New("server did not return a job-id")
	}
	return jobID, nil
}

// GetJobStatus polls the status of a job. A failed job returns a populated
// JobStatusResult with a nil Go error — the query succeeded. A nil result
// plus a non-nil error means the query itself failed (job_not_found, etc.).
func (p *PackageClient) GetJobStatus(ctx context.Context, jobID string) (*mvep.JobStatusResult, error) {
	if jobID == "" {
		return nil, errors.New("missing job ID")
	}

	req := mvep.NewCmdReq(mvep.GetJobStatusName, nil)
	req.Headers["job-id"] = jobID

	resp, err := p.SendEnvelope(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("GetJobStatus failed: %w", err)
	}
	if resp.HasError() {
		return nil, fmt.Errorf("GetJobStatus error: [%s] %s", resp.Error.Code, resp.Error.Message)
	}

	result := &mvep.JobStatusResult{
		Status:  resp.Headers["job-status"],
		Cmd:     resp.Headers["job-cmd"],
		Payload: resp.Payload,
	}

	// Progress headers are optional.
	if pct := resp.Headers["job-progress-percent"]; pct != "" {
		var percent int
		if _, err := fmt.Sscanf(pct, "%d", &percent); err == nil {
			result.Progress = &mvep.JobProgress{
				Percent: percent,
				Message: resp.Headers["job-progress-message"],
			}
		}
	}

	// Job failure headers are set only when status == "failed".
	if resp.Headers["job-error-code"] != "" {
		result.Error = &mvep.ErrorInfo{
			Code:    resp.Headers["job-error-code"],
			Message: resp.Headers["job-error-message"],
		}
	}

	return result, nil
}

// WaitForJob polls GetJobStatus at the given interval until the job reaches a
// terminal state (succeeded or failed) or ctx is cancelled. On success it
// decodes the result payload into the package's <CmdName>Result type.
func (p *PackageClient) WaitForJob(ctx context.Context, jobID string, pollInterval time.Duration) (any, error) {
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}

	for {
		status, err := p.GetJobStatus(ctx, jobID)
		if err != nil {
			return nil, err
		}

		if status.Status == string(mvep.JobSucceeded) {
			// Decode the payload into the result type. The inner command name
			// was echoed back as the job-cmd response header.
			innerCmd := status.Cmd
			if innerCmd == "" {
				return nil, errors.New("job succeeded but no job-cmd header in response")
			}
			resultType, ok := p.pkg.InstanceOf(innerCmd + "Result")
			if !ok {
				return nil, fmt.Errorf("unknown command result %s", innerCmd+"Result")
			}
			encoder := p.GetEncoder()
			enc, ok := oenc.LookupEncoding(encoder)
			if !ok {
				return nil, fmt.Errorf("encoding not found, %s", encoder)
			}
			if err := enc.Decode(status.Payload, resultType); err != nil {
				return nil, fmt.Errorf("failed to decode result: %w", err)
			}
			return resultType, nil
		}

		if status.Status == string(mvep.JobFailed) {
			if status.Error != nil {
				return nil, fmt.Errorf("job failed: [%s] %s", status.Error.Code, status.Error.Message)
			}
			return nil, errors.New("job failed")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
