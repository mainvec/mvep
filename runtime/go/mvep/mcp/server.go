package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
	_ "github.com/mainvec/ugo/oencoding/json"
)

// DefaultProtocolVersion is the MCP protocol version advertised when the client
// does not request one or requests an unsupported version.
const DefaultProtocolVersion = "2025-03-26"

// Standard JSON-RPC 2.0 error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Server exposes a single MVEP package as an MCP tool server.
type Server struct {
	desc       *mvep.PackageDesc
	handler    *mvep.PackageHandler
	cfg        config
	tools      []Tool
	toolByName map[string]*mvep.CommandDesc
}

// NewServer creates an MCP server backed by desc (for tools/list schemas and
// argument validation) and handler (for tools/call execution through
// PackageHandler.ServeCmdReq).
//
// NewServer validates all command schemas and tool-name uniqueness upfront so
// configuration errors fail immediately rather than on the first client call.
func NewServer(desc *mvep.PackageDesc, handler *mvep.PackageHandler, opts ...Option) (*Server, error) {
	if desc == nil {
		return nil, errors.New("mcp: package descriptor is nil")
	}
	if handler == nil {
		return nil, errors.New("mcp: package handler is nil")
	}

	cfg := defaultConfig(desc)
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	entries, err := buildToolIndex(desc, opts...)
	if err != nil {
		return nil, err
	}

	tools := make([]Tool, len(entries))
	toolByName := make(map[string]*mvep.CommandDesc, len(entries))
	for i, e := range entries {
		tools[i] = e.tool
		toolByName[e.tool.Name] = e.cmd
	}

	return &Server{
		desc:       desc,
		handler:    handler,
		cfg:        cfg,
		tools:      tools,
		toolByName: toolByName,
	}, nil
}

// Tools returns a copy of the MCP tools exposed by this server in descriptor
// order.
func (s *Server) Tools() []Tool {
	out := make([]Tool, len(s.tools))
	copy(out, s.tools)
	return out
}

// Serve runs the JSON-RPC loop over transport until EOF, context cancellation,
// or an unrecoverable transport error. Clean EOF returns nil.
func (s *Server) Serve(ctx context.Context, transport Transport) error {
	if transport == nil {
		return errors.New("mcp: transport is nil")
	}
	for {
		raw, err := transport.Read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		resp, err := s.HandleMessage(ctx, raw)
		if err != nil {
			return err
		}
		if len(resp) == 0 {
			// Notification — JSON-RPC 2.0 requires no response.
			continue
		}
		if err := transport.Write(ctx, resp); err != nil {
			return err
		}
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type callToolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// HandleMessage processes a single JSON-RPC 2.0 request or notification and
// returns the serialized JSON-RPC response, or nil when no response should be
// emitted (notifications).
func (s *Server) HandleMessage(ctx context.Context, raw []byte) ([]byte, error) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &rpcError{Code: codeParseError, Message: fmt.Sprintf("parse error: %v", err)},
		})
	}

	isNotification := len(req.ID) == 0

	if req.JSONRPC != "2.0" || req.Method == "" {
		if isNotification {
			return nil, nil
		}
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: codeInvalidRequest, Message: "invalid JSON-RPC 2.0 request"},
		})
	}

	// JSON-RPC notifications (no id) must never produce a response.
	if isNotification {
		return nil, nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID, req.Params)
	case "ping":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		})
	case "tools/list":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": s.tools,
			},
		})
	case "tools/call":
		return s.handleToolsCall(ctx, req.ID, req.Params)
	default:
		if strings.HasPrefix(req.Method, "notifications/") {
			return nil, nil
		}
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: codeMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)},
		})
	}
}

func (s *Server) handleInitialize(id json.RawMessage, params json.RawMessage) ([]byte, error) {
	protoVersion := DefaultProtocolVersion
	if len(params) > 0 {
		var p initializeParams
		if err := json.Unmarshal(params, &p); err == nil && p.ProtocolVersion != "" {
			protoVersion = p.ProtocolVersion
		}
	}

	return marshalResponse(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"protocolVersion": protoVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    s.cfg.serverName,
				"version": s.cfg.serverVersion,
			},
		},
	})
}

func (s *Server) handleToolsCall(ctx context.Context, id json.RawMessage, params json.RawMessage) ([]byte, error) {
	var p callToolParams
	if len(params) == 0 {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: codeInvalidParams, Message: "missing params for tools/call"},
		})
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: codeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)},
		})
	}
	if p.Name == "" {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: codeInvalidParams, Message: "missing tool name"},
		})
	}

	cmdDesc, ok := s.toolByName[p.Name]
	if !ok {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: codeInvalidParams, Message: fmt.Sprintf("unknown tool: %s", p.Name)},
		})
	}

	args := p.Arguments
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage("{}")
	}

	if err := validateArgs(cmdDesc, s.desc, args); err != nil {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  toolErrorResult(err.Error()),
		})
	}

	// Dispatch through PackageHandler.ServeCmdReq so the full interceptor chain
	// (RecoveryInterceptor, AuthInterceptor, LoggingInterceptor, etc.) applies.
	cmdReq := mvep.NewCmdReq(cmdDesc.Name, args)
	cmdResp := s.handler.ServeCmdReq(ctx, cmdReq, "application/json")
	if cmdResp == nil {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  toolErrorResult("internal error: handler returned nil response"),
		})
	}
	if cmdResp.HasError() {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  toolErrorResult(fmt.Sprintf("[%s] %s", cmdResp.Error.Code, cmdResp.Error.Message)),
		})
	}

	text := string(cmdResp.Payload)
	if text == "" {
		text = "{}"
	}
	return marshalResponse(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: callToolResult{
			Content: []textContent{{Type: "text", Text: text}},
		},
	})
}

func toolErrorResult(msg string) callToolResult {
	return callToolResult{
		Content: []textContent{{Type: "text", Text: msg}},
		IsError: true,
	}
}

func marshalResponse(resp rpcResponse) ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshal response: %w", err)
	}
	return b, nil
}
