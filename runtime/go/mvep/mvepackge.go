package mvep

import (
	"bytes"
	"context"
	"fmt"
	"io"

	oenc "github.com/mainvec/ugo/oencoding"
)

var _ Transporter = &HttpTransporter{}
var _ EnvelopeTransporter = &HttpTransporter{}

type Package interface {
	GetName() string
	InstanceOf(compName string) (any, bool)
	NameOf(comp any) string
	//RunCmd(ctx context.Context, cmd any) (any, error)
}

// CommandLister is an optional interface a Package can implement to expose its
// command names, enabling the client to build a deterministic command index.
type CommandLister interface {
	CommandNames() []string
}

// Transporter is the interface for transporting commands (legacy, without headers)
type Transporter interface {
	TransportCmd(ctx context.Context, cmdName string, contentType string, cmdData io.ReadCloser) (io.ReadCloser, error)
}

// EnvelopeTransporter is the interface for transporting commands with headers support
type EnvelopeTransporter interface {
	TransportCmdReq(ctx context.Context, req *CmdReq, contentType string) (*CmdResp, error)
}

type CommandRunner interface {
	RunCmd(ctx context.Context, cmd any) (any, error)
}

type PackageHandler struct {
	Package       Package
	Transporter   Transporter
	CommandRunner CommandRunner
	Interceptor   CmdInterceptor // Optional interceptor chain for command processing
	// MaxRequestBytes bounds the command request body read by ServeHTTP.
	// Zero uses DefaultMaxRequestBytes.
	MaxRequestBytes int64
	// MaxResponseBytes bounds the response body the client reads in SendCmdReq.
	// Zero uses DefaultMaxResponseBytes.
	MaxResponseBytes int64
	// VerboseErrors restores reflection of raw handler error text to callers.
	// Default false: the response carries a stable code and a generic message.
	VerboseErrors bool
}

// Default body bounds for the transport layer.
const (
	DefaultMaxRequestBytes  = 4 << 20 // 4 MiB
	DefaultMaxResponseBytes = 4 << 20 // 4 MiB
)

func (h *PackageHandler) ServeCmd(cmdName string, encoder string, cmdData io.ReadCloser) (io.ReadCloser, error) {
	if len(cmdName) == 0 {
		return nil, fmt.Errorf("missing command name")
	}
	if len(encoder) == 0 {
		return nil, fmt.Errorf("missing encoder")

	}
	if cmdData == nil {
		return nil, fmt.Errorf("missing command data")
	}
	cmd, ok := h.Package.InstanceOf(cmdName)
	if !ok {
		return nil, fmt.Errorf("unknown command %s", cmdName)
	}
	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return nil, fmt.Errorf("unknown encoder %s", encoder)
	}

	defer cmdData.Close()
	cmdBytes, err := io.ReadAll(cmdData)
	if err != nil {
		return nil, fmt.Errorf("failed to read command data: %w", err)
	}
	err = enc.Decode(cmdBytes, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to decode command data: %w", err)
	}
	cmdResult, err := h.CommandRunner.RunCmd(context.Background(), cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}
	cmdResultBytes, err := enc.Encode(cmdResult)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command result: %w", err)
	}

	return io.NopCloser(bytes.NewReader(cmdResultBytes)), nil

}

// ServeCmdReq handles a command request with headers and returns a response with headers
func (h *PackageHandler) ServeCmdReq(ctx context.Context, req *CmdReq, encoder string) *CmdResp {
	if req == nil {
		return NewCmdRespError("invalid_request", "missing request")
	}
	if len(req.Cmd) == 0 {
		return NewCmdRespError("invalid_request", "missing command name")
	}
	if len(encoder) == 0 {
		return NewCmdRespError("invalid_request", "missing encoder")
	}

	// Create the core handler that executes the command
	coreHandler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return h.executeCmd(ctx, req, encoder)
	}

	// If interceptor is set, wrap the core handler
	if h.Interceptor != nil {
		return h.Interceptor(ctx, req, coreHandler)
	}

	return coreHandler(ctx, req)
}

// executeCmd is the core command execution logic
func (h *PackageHandler) executeCmd(ctx context.Context, req *CmdReq, encoder string) *CmdResp {
	cmd, ok := h.Package.InstanceOf(req.Cmd)
	if !ok {
		return NewCmdRespError("unknown_command", fmt.Sprintf("unknown command %s", req.Cmd))
	}
	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return NewCmdRespError("unsupported_media_type", fmt.Sprintf("unknown encoder %s", encoder))
	}

	err := enc.Decode(req.Payload, cmd)
	if err != nil {
		return NewCmdRespError("decode_error", fmt.Sprintf("failed to decode command data: %v", err))
	}

	// Pass request headers via context and seed a CmdResp so handlers can set
	// response headers via SetResponseHeader.
	ctx = ContextWithCmdReq(ctx, req)
	ctx = ContextWithCmdResp(ctx, NewCmdResp(nil))

	cmdResult, err := h.CommandRunner.RunCmd(ctx, cmd)
	if err != nil {
		return NewCmdRespError("command_error", fmt.Sprintf("failed to run command: %v", err))
	}

	cmdResultBytes, err := enc.Encode(cmdResult)
	if err != nil {
		return NewCmdRespError("encode_error", fmt.Sprintf("failed to encode command result: %v", err))
	}

	// Get response headers seeded into the context above.
	resp := NewCmdResp(cmdResultBytes)
	if cmdResp := CmdRespFromContext(ctx); cmdResp != nil {
		resp.Headers = cmdResp.Headers
	}
	return resp
}

func (h *PackageHandler) SendCmd(ctx context.Context, cmd any, encoder string) (any, error) {
	if cmd == nil {
		return nil, fmt.Errorf("missing command")
	}
	cmdName := h.Package.NameOf(cmd)

	if len(cmdName) == 0 {
		return nil, fmt.Errorf("invalid command")
	}
	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return nil, fmt.Errorf("encoding not found, %s", encoder)
	}
	cmdBytes, err := enc.Encode(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command: %w", err)
	}
	respData, err := h.Transporter.TransportCmd(ctx, cmdName, enc.MimeType(), io.NopCloser(bytes.NewReader(cmdBytes)))
	if err != nil {
		return nil, fmt.Errorf("command failed. %w", err)
	}
	defer respData.Close()
	respBytes, err := io.ReadAll(respData)
	if err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}
	cmdResult, ok := h.Package.InstanceOf(cmdName + "Result")
	if !ok {
		return nil, fmt.Errorf("unknown command result %s", cmdName+"Result")
	}
	err = enc.Decode(respBytes, cmdResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response data: %w", err)
	}
	return cmdResult, nil

}

// SendCmdReq sends a command with headers and returns a response with headers
func (h *PackageHandler) SendCmdReq(ctx context.Context, cmd any, headers map[string]string, encoder string) (any, *CmdResp, error) {
	if cmd == nil {
		return nil, nil, fmt.Errorf("missing command")
	}
	cmdName := h.Package.NameOf(cmd)
	if len(cmdName) == 0 {
		return nil, nil, fmt.Errorf("invalid command")
	}

	enc, ok := oenc.LookupEncoding(encoder)
	if !ok {
		return nil, nil, fmt.Errorf("encoding not found, %s", encoder)
	}

	cmdBytes, err := enc.Encode(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode command: %w", err)
	}

	req := NewCmdReq(cmdName, cmdBytes)
	if headers != nil {
		// Copy so later mutation of the request cannot alias the caller's map.
		for k, v := range headers {
			req.Headers[k] = v
		}
	}

	// Check if transporter supports envelope transport
	envTransporter, ok := h.Transporter.(EnvelopeTransporter)
	if !ok {
		return nil, nil, fmt.Errorf("transporter does not support envelope transport")
	}

	resp, err := envTransporter.TransportCmdReq(ctx, req, enc.MimeType())
	if err != nil {
		return nil, nil, fmt.Errorf("command failed: %w", err)
	}

	if resp.HasError() {
		return nil, resp, fmt.Errorf("command error: [%s] %s", resp.Error.Code, resp.Error.Message)
	}

	cmdResult, ok := h.Package.InstanceOf(cmdName + "Result")
	if !ok {
		return nil, resp, fmt.Errorf("unknown command result %s", cmdName+"Result")
	}

	err = enc.Decode(resp.Payload, cmdResult)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to decode response data: %w", err)
	}

	return cmdResult, resp, nil
}
