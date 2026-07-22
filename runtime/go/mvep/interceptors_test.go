package mvep

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoggingInterceptor(t *testing.T) {
	interceptor := LoggingInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))
	handlerCalled := false

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if !handlerCalled {
		t.Error("Handler should have been called")
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
	if resp.HasError() {
		t.Error("Response should not have error")
	}
}

func TestLoggingInterceptor_WithError(t *testing.T) {
	interceptor := LoggingInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdRespError("test_error", "something went wrong")
	}

	resp := interceptor(context.Background(), req, handler)

	if !resp.HasError() {
		t.Error("Response should have error")
	}
}

// MockTokenValidator for testing
type mockTokenValidator struct {
	validTokens map[string]any
}

func (m *mockTokenValidator) Validate(ctx context.Context, token string) (any, error) {
	if user, ok := m.validTokens[token]; ok {
		return user, nil
	}
	return nil, errors.New("invalid token")
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validTokens: map[string]any{
			"valid-token": "user123",
		},
	}

	interceptor := AuthInterceptor(validator)
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithAuth("valid-token")

	var capturedCtx context.Context
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		capturedCtx = ctx
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if resp.HasError() {
		t.Errorf("Response should not have error, got: %s", resp.Error.Message)
	}

	// Check user was added to context
	user := UserFromContext(capturedCtx)
	if user != "user123" {
		t.Errorf("Expected user 'user123' in context, got %v", user)
	}
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	validator := &mockTokenValidator{}
	interceptor := AuthInterceptor(validator)
	req := NewCmdReq("TestCmd", []byte("{}"))
	// No auth header set

	handlerCalled := false
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if handlerCalled {
		t.Error("Handler should NOT have been called")
	}
	if !resp.HasError() {
		t.Error("Response should have error")
	}
	if resp.Error.Code != "unauthorized" {
		t.Errorf("Expected error code 'unauthorized', got '%s'", resp.Error.Code)
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validTokens: map[string]any{
			"valid-token": "user123",
		},
	}

	interceptor := AuthInterceptor(validator)
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithAuth("invalid-token")

	handlerCalled := false
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if handlerCalled {
		t.Error("Handler should NOT have been called")
	}
	if !resp.HasError() {
		t.Error("Response should have error")
	}
	if resp.Error.Code != "unauthorized" {
		t.Errorf("Expected error code 'unauthorized', got '%s'", resp.Error.Code)
	}
}

func TestAuthInterceptor_SkipCommands(t *testing.T) {
	validator := &mockTokenValidator{} // No valid tokens
	interceptor := SkipCommands(AuthInterceptor(validator), "PublicCmd")

	// Test public command (should skip auth)
	req := NewCmdReq("PublicCmd", []byte("{}"))
	handlerCalled := false
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if !handlerCalled {
		t.Error("Handler SHOULD have been called for skipped command")
	}
	if resp.HasError() {
		t.Error("Response should not have error for skipped command")
	}

	// Test protected command (should require auth)
	handlerCalled = false
	req = NewCmdReq("ProtectedCmd", []byte("{}"))
	resp = interceptor(context.Background(), req, handler)

	if handlerCalled {
		t.Error("Handler should NOT have been called for protected command without auth")
	}
	if !resp.HasError() {
		t.Error("Response should have error for protected command without auth")
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := RecoveryInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		panic("something went wrong!")
	}

	// Should not panic
	resp := interceptor(context.Background(), req, handler)

	if resp == nil {
		t.Fatal("Response should not be nil")
	}
	if !resp.HasError() {
		t.Error("Response should have error")
	}
	if resp.Error.Code != "internal_error" {
		t.Errorf("Expected error code 'internal_error', got '%s'", resp.Error.Code)
	}
}

func TestRecoveryInterceptor_NoPanic(t *testing.T) {
	interceptor := RecoveryInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if resp.HasError() {
		t.Error("Response should not have error when no panic")
	}
}

func TestRequestIDInterceptor(t *testing.T) {
	idCounter := 0
	generator := func() string {
		idCounter++
		return "req-123"
	}

	interceptor := RequestIDInterceptor(generator)
	req := NewCmdReq("TestCmd", []byte("{}"))

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	// Check request ID was added to request
	if req.Headers["request-id"] != "req-123" {
		t.Errorf("Expected request-id 'req-123' in request headers, got '%s'", req.Headers["request-id"])
	}

	// Check request ID was echoed in response
	if resp.GetHeader("request-id") != "req-123" {
		t.Errorf("Expected request-id 'req-123' in response headers, got '%s'", resp.GetHeader("request-id"))
	}
}

func TestRequestIDInterceptor_ExistingID(t *testing.T) {
	generatorCalled := false
	generator := func() string {
		generatorCalled = true
		return "new-id"
	}

	interceptor := RequestIDInterceptor(generator)
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithHeader("request-id", "existing-id")

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	if generatorCalled {
		t.Error("Generator should NOT have been called when ID exists")
	}
	if req.Headers["request-id"] != "existing-id" {
		t.Error("Existing request ID should be preserved")
	}
	if resp.GetHeader("request-id") != "existing-id" {
		t.Error("Existing request ID should be echoed in response")
	}
}

func TestRequestIDInterceptor_NilGenerator(t *testing.T) {
	interceptor := RequestIDInterceptor(nil)
	req := NewCmdReq("TestCmd", []byte("{}"))

	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		return NewCmdResp([]byte("result"))
	}

	resp := interceptor(context.Background(), req, handler)

	// Should not panic and should work without generating ID
	if resp.HasError() {
		t.Error("Should not have error")
	}
}

func TestUserFromContext_NotPresent(t *testing.T) {
	ctx := context.Background()
	user := UserFromContext(ctx)
	if user != nil {
		t.Error("UserFromContext should return nil when not present")
	}
}

// =============================================================================
// Client Interceptor Tests
// =============================================================================

func TestClientLoggingInterceptor(t *testing.T) {
	interceptor := ClientLoggingInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))
	invokerCalled := false

	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		invokerCalled = true
		return NewCmdResp([]byte("result")), nil
	}

	resp, err := interceptor(context.Background(), req, invoker)

	if !invokerCalled {
		t.Error("Invoker should have been called")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestClientLoggingInterceptor_WithError(t *testing.T) {
	interceptor := ClientLoggingInterceptor()
	req := NewCmdReq("TestCmd", []byte("{}"))

	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		return nil, errors.New("transport error")
	}

	_, err := interceptor(context.Background(), req, invoker)

	if err == nil {
		t.Error("Error should be propagated")
	}
}

func TestAuthHeaderInterceptor(t *testing.T) {
	tokenProvider := func(ctx context.Context) (string, error) {
		return "Bearer test-token", nil
	}

	interceptor := AuthHeaderInterceptor(tokenProvider)
	req := NewCmdReq("TestCmd", []byte("{}"))

	var capturedReq *CmdReq
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		capturedReq = req
		return NewCmdResp([]byte("result")), nil
	}

	_, err := interceptor(context.Background(), req, invoker)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if capturedReq.Headers["auth"] != "Bearer test-token" {
		t.Errorf("Expected auth header 'Bearer test-token', got '%s'", capturedReq.Headers["auth"])
	}
}

func TestAuthHeaderInterceptor_ProviderError(t *testing.T) {
	tokenProvider := func(ctx context.Context) (string, error) {
		return "", errors.New("token refresh failed")
	}

	interceptor := AuthHeaderInterceptor(tokenProvider)
	req := NewCmdReq("TestCmd", []byte("{}"))

	invokerCalled := false
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		invokerCalled = true
		return NewCmdResp([]byte("result")), nil
	}

	_, err := interceptor(context.Background(), req, invoker)

	if err == nil {
		t.Error("Error should be returned when token provider fails")
	}
	if invokerCalled {
		t.Error("Invoker should NOT be called when token provider fails")
	}
}

func TestStaticAuthHeaderInterceptor(t *testing.T) {
	interceptor := StaticAuthHeaderInterceptor("Bearer static-token")
	req := NewCmdReq("TestCmd", []byte("{}"))

	var capturedReq *CmdReq
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		capturedReq = req
		return NewCmdResp([]byte("result")), nil
	}

	interceptor(context.Background(), req, invoker)

	if capturedReq.Headers["auth"] != "Bearer static-token" {
		t.Errorf("Expected auth header 'Bearer static-token', got '%s'", capturedReq.Headers["auth"])
	}
}

func TestHeaderInterceptor(t *testing.T) {
	headers := map[string]string{
		"custom-header": "custom-value",
		"another":       "value",
	}

	interceptor := HeaderInterceptor(headers)
	req := NewCmdReq("TestCmd", []byte("{}"))

	var capturedReq *CmdReq
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		capturedReq = req
		return NewCmdResp([]byte("result")), nil
	}

	interceptor(context.Background(), req, invoker)

	if capturedReq.Headers["custom-header"] != "custom-value" {
		t.Error("Custom header should be set")
	}
	if capturedReq.Headers["another"] != "value" {
		t.Error("Another header should be set")
	}
}

func TestRetryInterceptor_Success(t *testing.T) {
	interceptor := RetryInterceptor(3, time.Millisecond)
	req := NewCmdReq("TestCmd", []byte("{}"))

	callCount := 0
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		callCount++
		return NewCmdResp([]byte("result")), nil
	}

	resp, err := interceptor(context.Background(), req, invoker)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
	if callCount != 1 {
		t.Errorf("Should only call once on success, called %d times", callCount)
	}
}

func TestRetryInterceptor_RetryOnError(t *testing.T) {
	interceptor := RetryInterceptor(2, time.Millisecond)
	req := NewCmdReq("TestCmd", []byte("{}"))

	callCount := 0
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		callCount++
		if callCount < 3 {
			return nil, errors.New("transient error")
		}
		return NewCmdResp([]byte("result")), nil
	}

	resp, err := interceptor(context.Background(), req, invoker)

	if err != nil {
		t.Errorf("Should succeed after retries: %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
	if callCount != 3 {
		t.Errorf("Should retry and succeed on 3rd attempt, called %d times", callCount)
	}
}

func TestRetryInterceptor_MaxRetriesExceeded(t *testing.T) {
	interceptor := RetryInterceptor(2, time.Millisecond)
	req := NewCmdReq("TestCmd", []byte("{}"))

	callCount := 0
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		callCount++
		return nil, errors.New("persistent error")
	}

	_, err := interceptor(context.Background(), req, invoker)

	if err == nil {
		t.Error("Should return error after max retries")
	}
	if callCount != 3 {
		t.Errorf("Should call 1 + 2 retries = 3 times, called %d times", callCount)
	}
}

func TestClientRequestIDInterceptor(t *testing.T) {
	generator := func() string {
		return "generated-id-123"
	}

	interceptor := ClientRequestIDInterceptor(generator)
	req := NewCmdReq("TestCmd", []byte("{}"))

	var capturedReq *CmdReq
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		capturedReq = req
		return NewCmdResp([]byte("result")), nil
	}

	interceptor(context.Background(), req, invoker)

	if capturedReq.Headers["request-id"] != "generated-id-123" {
		t.Errorf("Expected request-id 'generated-id-123', got '%s'", capturedReq.Headers["request-id"])
	}
}

func TestClientRequestIDInterceptor_ExistingID(t *testing.T) {
	generatorCalled := false
	generator := func() string {
		generatorCalled = true
		return "new-id"
	}

	interceptor := ClientRequestIDInterceptor(generator)
	req := NewCmdReq("TestCmd", []byte("{}"))
	req.WithHeader("request-id", "existing-id")

	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		return NewCmdResp([]byte("result")), nil
	}

	interceptor(context.Background(), req, invoker)

	if generatorCalled {
		t.Error("Generator should NOT be called when ID exists")
	}
	if req.Headers["request-id"] != "existing-id" {
		t.Error("Existing ID should be preserved")
	}
}

func TestChainClient(t *testing.T) {
	var order []int

	interceptor1 := func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		order = append(order, 1)
		resp, err := invoker(ctx, req)
		order = append(order, -1)
		return resp, err
	}

	interceptor2 := func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		order = append(order, 2)
		resp, err := invoker(ctx, req)
		order = append(order, -2)
		return resp, err
	}

	chain := ChainClient(interceptor1, interceptor2)
	req := NewCmdReq("TestCmd", []byte("{}"))

	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		order = append(order, 0)
		return NewCmdResp([]byte("result")), nil
	}

	chain(context.Background(), req, invoker)

	expected := []int{1, 2, 0, -2, -1}
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

func TestChainClient_Empty(t *testing.T) {
	chain := ChainClient()
	if chain != nil {
		t.Error("ChainClient() with no interceptors should return nil")
	}
}

func TestChainClient_Single(t *testing.T) {
	called := false
	interceptor := func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		called = true
		return invoker(ctx, req)
	}

	chain := ChainClient(interceptor)
	req := NewCmdReq("TestCmd", []byte("{}"))

	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		return NewCmdResp([]byte("result")), nil
	}

	chain(context.Background(), req, invoker)

	if !called {
		t.Error("Single interceptor should have been called")
	}
}

func TestSkipCommandsClient(t *testing.T) {
	interceptorCalled := false
	interceptor := func(ctx context.Context, req *CmdReq, invoker ClientInvoker) (*CmdResp, error) {
		interceptorCalled = true
		return invoker(ctx, req)
	}

	wrapped := SkipCommandsClient(interceptor, "SkipMe")
	invoker := func(ctx context.Context, req *CmdReq) (*CmdResp, error) {
		return NewCmdResp([]byte("result")), nil
	}

	// Test skipped command
	req := NewCmdReq("SkipMe", []byte("{}"))
	wrapped(context.Background(), req, invoker)
	if interceptorCalled {
		t.Error("Interceptor should NOT be called for skipped command")
	}

	// Test non-skipped command
	interceptorCalled = false
	req = NewCmdReq("ProcessMe", []byte("{}"))
	wrapped(context.Background(), req, invoker)
	if !interceptorCalled {
		t.Error("Interceptor SHOULD be called for non-skipped command")
	}
}

func TestAuthInterceptor_LocalTrustBypass(t *testing.T) {
	validator := &mockTokenValidator{} // No valid tokens
	interceptor := AuthInterceptor(validator)
	req := NewCmdReq("ProtectedCmd", []byte("{}"))
	// No auth token set — would normally fail.

	handlerCalled := false
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("ok"))
	}

	// With local trust, auth should be skipped.
	ctx := ContextWithLocalTrust(context.Background())
	resp := interceptor(ctx, req, handler)

	if !handlerCalled {
		t.Error("Handler SHOULD have been called for locally trusted request")
	}
	if resp.HasError() {
		t.Errorf("Response should not have error for locally trusted request, got: %s", resp.Error.Message)
	}
}

func TestAuthInterceptor_LocalTrustDoesNotBypassWithoutMarker(t *testing.T) {
	validator := &mockTokenValidator{} // No valid tokens
	interceptor := AuthInterceptor(validator)
	req := NewCmdReq("ProtectedCmd", []byte("{}"))

	handlerCalled := false
	handler := func(ctx context.Context, req *CmdReq) *CmdResp {
		handlerCalled = true
		return NewCmdResp([]byte("ok"))
	}

	// Without local trust marker, should fail.
	resp := interceptor(context.Background(), req, handler)

	if handlerCalled {
		t.Error("Handler should NOT have been called without auth or local trust")
	}
	if !resp.HasError() {
		t.Error("Response should have error without auth or local trust")
	}
}
