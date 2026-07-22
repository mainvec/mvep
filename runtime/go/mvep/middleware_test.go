package mvep

import (
	"context"
	"testing"
)

func TestChain_Empty(t *testing.T) {
	chain := Chain()
	if chain != nil {
		t.Error("Chain() with no interceptors should return nil")
	}
}

func TestChain_Single(t *testing.T) {
	called := false
	interceptor := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		called = true
		return next(ctx, req)
	}

	chain := Chain(interceptor)
	req := NewCmdReq("TestCmd", []byte("{}"))
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	resp := chain(context.Background(), req, handler)

	if !called {
		t.Error("Single interceptor should have been called")
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestChain_Multiple_Order(t *testing.T) {
	var order []int

	interceptor1 := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		order = append(order, 1)
		resp := next(ctx, req)
		order = append(order, -1)
		return resp
	}

	interceptor2 := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		order = append(order, 2)
		resp := next(ctx, req)
		order = append(order, -2)
		return resp
	}

	interceptor3 := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		order = append(order, 3)
		resp := next(ctx, req)
		order = append(order, -3)
		return resp
	}

	chain := Chain(interceptor1, interceptor2, interceptor3)
	req := NewCmdReq("TestCmd", []byte("{}"))
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		order = append(order, 0)
		return NewCmdResp([]byte("result"))
	}

	chain(context.Background(), req, handler)

	expected := []int{1, 2, 3, 0, -3, -2, -1}
	if len(order) != len(expected) {
		t.Errorf("Expected %d calls, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("Expected order %v, got %v", expected, order)
			break
		}
	}
}

func TestChain_ShortCircuit(t *testing.T) {
	interceptor1Called := false
	interceptor2Called := false
	handlerCalled := false

	interceptor1 := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		interceptor1Called = true
		return NewCmdRespError("blocked", "request blocked")
	}

	interceptor2 := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		interceptor2Called = true
		return next(ctx, req)
	}

	chain := Chain(interceptor1, interceptor2)
	req := NewCmdReq("TestCmd", []byte("{}"))
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("result"))
	}

	resp := chain(context.Background(), req, handler)

	if !interceptor1Called {
		t.Error("Interceptor1 should have been called")
	}
	if interceptor2Called {
		t.Error("Interceptor2 should NOT have been called (short-circuited)")
	}
	if handlerCalled {
		t.Error("Handler should NOT have been called (short-circuited)")
	}
	if !resp.HasError() {
		t.Error("Response should have error")
	}
	if resp.Error.Code != "blocked" {
		t.Errorf("Expected error code 'blocked', got '%s'", resp.Error.Code)
	}
}

func TestSkipCommands(t *testing.T) {
	interceptorCalled := false
	interceptor := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		interceptorCalled = true
		return next(ctx, req)
	}

	wrapped := SkipCommands(interceptor, "SkipMe", "SkipMeToo")
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	req := NewCmdReq("SkipMe", []byte("{}"))
	wrapped(context.Background(), req, handler)
	if interceptorCalled {
		t.Error("Interceptor should NOT be called for skipped command")
	}

	interceptorCalled = false
	req = NewCmdReq("ProcessMe", []byte("{}"))
	wrapped(context.Background(), req, handler)
	if !interceptorCalled {
		t.Error("Interceptor SHOULD be called for non-skipped command")
	}
}

func TestSkipCommands_NilInterceptor(t *testing.T) {
	result := SkipCommands(nil, "SomeCmd")
	if result != nil {
		t.Error("SkipCommands with nil interceptor should return nil")
	}
}

func TestOnlyCommands(t *testing.T) {
	interceptorCalled := false
	interceptor := func(ctx context.Context, req *CmdReq, next CmdHandler) *CmdResp {
		interceptorCalled = true
		return next(ctx, req)
	}

	wrapped := OnlyCommands(interceptor, "OnlyMe", "OnlyMeToo")
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	req := NewCmdReq("OnlyMe", []byte("{}"))
	wrapped(context.Background(), req, handler)
	if !interceptorCalled {
		t.Error("Interceptor SHOULD be called for included command")
	}

	interceptorCalled = false
	req = NewCmdReq("NotIncluded", []byte("{}"))
	wrapped(context.Background(), req, handler)
	if interceptorCalled {
		t.Error("Interceptor should NOT be called for excluded command")
	}
}

func TestOnlyCommands_NilInterceptor(t *testing.T) {
	result := OnlyCommands(nil, "SomeCmd")
	if result != nil {
		t.Error("OnlyCommands with nil interceptor should return nil")
	}
}

func TestCmdReq_WithHeader(t *testing.T) {
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithHeader("key1", "value1").WithHeader("key2", "value2")

	if req.Headers["key1"] != "value1" {
		t.Errorf("Expected header key1=value1, got %s", req.Headers["key1"])
	}
	if req.Headers["key2"] != "value2" {
		t.Errorf("Expected header key2=value2, got %s", req.Headers["key2"])
	}
}

func TestCmdReq_WithAuth(t *testing.T) {
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithAuth("Bearer token123")

	if req.Headers["auth"] != "Bearer token123" {
		t.Errorf("Expected auth header, got %s", req.Headers["auth"])
	}
}

func TestCmdResp_WithHeader(t *testing.T) {
	resp := NewCmdResp([]byte("result"))
	resp.WithHeader("key1", "value1")

	if resp.Headers["key1"] != "value1" {
		t.Errorf("Expected header key1=value1, got %s", resp.Headers["key1"])
	}
}

func TestCmdResp_GetHeader(t *testing.T) {
	resp := NewCmdResp([]byte("result"))
	resp.Headers["existing"] = "value"

	if resp.GetHeader("existing") != "value" {
		t.Error("GetHeader should return existing header value")
	}
	if resp.GetHeader("nonexistent") != "" {
		t.Error("GetHeader should return empty string for nonexistent header")
	}
}

func TestCmdResp_HasError(t *testing.T) {
	resp := NewCmdResp([]byte("result"))
	if resp.HasError() {
		t.Error("Response without error should return false")
	}

	errResp := NewCmdRespError("err_code", "err message")
	if !errResp.HasError() {
		t.Error("Response with error should return true")
	}
}

func TestContextWithCmdReq(t *testing.T) {
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.Headers["test"] = "value"

	ctx := ContextWithCmdReq(context.Background(), req)
	retrieved := CmdReqFromContext(ctx)

	if retrieved == nil {
		t.Fatal("CmdReqFromContext should return the request")
	}
	if retrieved.Cmd != "TestCmd" {
		t.Errorf("Expected Cmd 'TestCmd', got '%s'", retrieved.Cmd)
	}
	if retrieved.Headers["test"] != "value" {
		t.Error("Headers should be preserved")
	}
}

func TestCmdReqFromContext_NotPresent(t *testing.T) {
	ctx := context.Background()
	req := CmdReqFromContext(ctx)
	if req != nil {
		t.Error("CmdReqFromContext should return nil when not present")
	}
}

func TestGetRequestHeader(t *testing.T) {
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.Headers["auth"] = "Bearer token"

	ctx := ContextWithCmdReq(context.Background(), req)

	if GetRequestHeader(ctx, "auth") != "Bearer token" {
		t.Error("GetRequestHeader should return header value")
	}
	if GetRequestHeader(ctx, "nonexistent") != "" {
		t.Error("GetRequestHeader should return empty string for nonexistent header")
	}
	if GetRequestHeader(context.Background(), "auth") != "" {
		t.Error("GetRequestHeader should return empty string when no CmdReq in context")
	}
}

func TestSetResponseHeader(t *testing.T) {
	ctx := context.Background()

	ctx = SetResponseHeader(ctx, "key1", "value1")
	resp := CmdRespFromContext(ctx)
	if resp == nil {
		t.Fatal("SetResponseHeader should create CmdResp in context")
	}
	if resp.Headers["key1"] != "value1" {
		t.Error("Header should be set")
	}

	ctx = SetResponseHeader(ctx, "key2", "value2")
	resp = CmdRespFromContext(ctx)
	if resp.Headers["key1"] != "value1" {
		t.Error("Previous header should still exist")
	}
	if resp.Headers["key2"] != "value2" {
		t.Error("New header should be set")
	}
}
