package mvp

import "context"

// HeaderPrefix is the prefix for custom MVP headers in HTTP transport
const HeaderPrefix = "x-mvp-"

// Context keys for passing CmdReq/CmdResp through context
type cmdReqContextKey struct{}
type cmdRespContextKey struct{}

// CmdReq represents a command request with headers
type CmdReq struct {
	// Cmd is the command name
	Cmd string
	// Headers contains request metadata (auth tokens, trace IDs, etc.)
	Headers map[string]string
	// Payload is the encoded command bytes
	Payload []byte
}

// CmdResp represents a command response with headers
type CmdResp struct {
	// Headers contains response metadata (new tokens, rate limits, etc.)
	Headers map[string]string
	// Payload is the encoded result bytes
	Payload []byte
	// Error contains error information if the command failed
	Error *ErrorInfo
}

// ErrorInfo provides structured error information
type ErrorInfo struct {
	Code    string
	Message string
}

// NewCmdReq creates a new CmdReq with initialized headers map
func NewCmdReq(cmd string, payload []byte) *CmdReq {
	return &CmdReq{
		Cmd:     cmd,
		Headers: make(map[string]string),
		Payload: payload,
	}
}

// WithHeader adds a header to the request and returns the request for chaining
func (r *CmdReq) WithHeader(key, value string) *CmdReq {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// WithAuth adds an authorization header
func (r *CmdReq) WithAuth(token string) *CmdReq {
	return r.WithHeader("auth", token)
}

// NewCmdResp creates a new CmdResp with initialized headers map
func NewCmdResp(payload []byte) *CmdResp {
	return &CmdResp{
		Headers: make(map[string]string),
		Payload: payload,
	}
}

// NewCmdRespError creates a new CmdResp with an error
func NewCmdRespError(code, message string) *CmdResp {
	return &CmdResp{
		Headers: make(map[string]string),
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// WithHeader adds a header to the response and returns the response for chaining
func (r *CmdResp) WithHeader(key, value string) *CmdResp {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// GetHeader returns a header value from the response
func (r *CmdResp) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers[key]
}

// HasError returns true if the response contains an error
func (r *CmdResp) HasError() bool {
	return r.Error != nil
}

// ContextWithCmdReq embeds CmdReq into a context.Context
func ContextWithCmdReq(ctx context.Context, req *CmdReq) context.Context {
	return context.WithValue(ctx, cmdReqContextKey{}, req)
}

// CmdReqFromContext extracts CmdReq from context.Context
// Returns nil if not present
func CmdReqFromContext(ctx context.Context) *CmdReq {
	if req, ok := ctx.Value(cmdReqContextKey{}).(*CmdReq); ok {
		return req
	}
	return nil
}

// ContextWithCmdResp embeds CmdResp into a context.Context
func ContextWithCmdResp(ctx context.Context, resp *CmdResp) context.Context {
	return context.WithValue(ctx, cmdRespContextKey{}, resp)
}

// CmdRespFromContext extracts CmdResp from context.Context
// Returns nil if not present
func CmdRespFromContext(ctx context.Context) *CmdResp {
	if resp, ok := ctx.Value(cmdRespContextKey{}).(*CmdResp); ok {
		return resp
	}
	return nil
}

// GetRequestHeader is a convenience function to get a request header from context
func GetRequestHeader(ctx context.Context, key string) string {
	if req := CmdReqFromContext(ctx); req != nil {
		return req.Headers[key]
	}
	return ""
}

// SetResponseHeader is a convenience function to set a response header in context
// It creates a CmdResp in context if not present
func SetResponseHeader(ctx context.Context, key, value string) context.Context {
	resp := CmdRespFromContext(ctx)
	if resp == nil {
		resp = &CmdResp{Headers: make(map[string]string)}
		ctx = ContextWithCmdResp(ctx, resp)
	}
	resp.Headers[key] = value
	return ctx
}
