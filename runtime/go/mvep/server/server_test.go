package server

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestExampleServer demonstrates server creation with the Listeners API.
func TestExampleServer(t *testing.T) {
	config := &ServerConfig{
		Listeners:         []ListenerConfig{{Address: "127.0.0.1:0"}},
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		EnableCORS:        true,
		OnShutdown: func() {
			slog.Info("Custom shutdown handler called")
		},
	}

	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		err := server.Start()
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for the server to start
	for server.GetListener() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	// Check health endpoint
	url := "http://" + server.GetListener().Addr().String() + "/health"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "OK" {
		t.Errorf("Expected health check response 'OK', got '%s'", string(body))
	}

	err = server.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestServerWithInterceptor demonstrates server configuration with middleware
func TestServerWithInterceptor(t *testing.T) {
	interceptorCalled := false
	testInterceptor := func(ctx context.Context, req *mvep.CmdReq, next mvep.CmdHandler) *mvep.CmdResp {
		interceptorCalled = true
		return next(ctx, req)
	}

	config := &ServerConfig{
		Listeners:         []ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		Interceptor:       testInterceptor,
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Verify interceptor is set in config
	if server.config.Interceptor == nil {
		t.Error("Interceptor should be set in server config")
	}

	// The interceptorCalled variable would be set to true when a command is processed
	// For this test, we just verify the setup is correct
	_ = interceptorCalled

	// Shutdown
	server.Shutdown()
}

// TestServerWithChainedInterceptors demonstrates multiple interceptors
func TestServerWithChainedInterceptors(t *testing.T) {
	var callOrder []string

	interceptor1 := func(ctx context.Context, req *mvep.CmdReq, next mvep.CmdHandler) *mvep.CmdResp {
		callOrder = append(callOrder, "interceptor1-before")
		resp := next(ctx, req)
		callOrder = append(callOrder, "interceptor1-after")
		return resp
	}

	interceptor2 := func(ctx context.Context, req *mvep.CmdReq, next mvep.CmdHandler) *mvep.CmdResp {
		callOrder = append(callOrder, "interceptor2-before")
		resp := next(ctx, req)
		callOrder = append(callOrder, "interceptor2-after")
		return resp
	}

	config := &ServerConfig{
		Listeners:         []ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		Interceptor: mvep.Chain(
			interceptor1,
			interceptor2,
		),
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Verify chain is set
	if server.config.Interceptor == nil {
		t.Error("Chained interceptor should be set in server config")
	}

	server.Shutdown()
}

// TestServerWithSkipCommands demonstrates skip pattern
func TestServerWithSkipCommands(t *testing.T) {
	authCalled := false
	authInterceptor := func(ctx context.Context, req *mvep.CmdReq, next mvep.CmdHandler) *mvep.CmdResp {
		authCalled = true
		// Simulate auth check
		if req.Headers["auth"] == "" {
			return mvep.NewCmdRespError("unauthorized", "missing auth")
		}
		return next(ctx, req)
	}

	config := &ServerConfig{
		Listeners:         []ListenerConfig{{Address: "127.0.0.1:0"}},
		BasePath:          "/api",
		EnableHealthCheck: true,
		Interceptor: mvep.Chain(
			mvep.LoggingInterceptor(),
			mvep.SkipCommands(authInterceptor, "HealthCheck", "PublicQuery"),
		),
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.config.Interceptor == nil {
		t.Error("Interceptor with skip should be set")
	}

	// The authCalled variable would be set when processing commands
	_ = authCalled

	server.Shutdown()
}

// TestMultipleListeners verifies that the server serves on multiple listeners.
func TestMultipleListeners(t *testing.T) {
	// Create a pre-created listener for the second entry.
	extraLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create extra listener: %v", err)
	}

	config := &ServerConfig{
		Listeners: []ListenerConfig{
			{Address: "127.0.0.1:0"},
			{Listener: extraLn},
		},
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	}

	svr, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go svr.Start()

	// Wait for the first listener to be ready.
	for svr.GetListener() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	// Verify first listener serves health check.
	primaryURL := "http://" + svr.GetListener().Addr().String() + "/health"
	resp, err := http.Get(primaryURL)
	if err != nil {
		t.Fatalf("Failed to reach first listener: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("First listener health check: expected 200, got %d", resp.StatusCode)
	}

	// Verify second listener serves the same health check.
	extraURL := "http://" + extraLn.Addr().String() + "/health"
	resp2, err := http.Get(extraURL)
	if err != nil {
		t.Fatalf("Failed to reach second listener: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Second listener health check: expected 200, got %d", resp2.StatusCode)
	}

	svr.Shutdown()
}

// TestListenerWithMiddleware verifies per-listener middleware is applied.
func TestListenerWithMiddleware(t *testing.T) {
	middlewareCalled := false
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	extraLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create extra listener: %v", err)
	}

	config := &ServerConfig{
		Listeners: []ListenerConfig{
			{Address: "127.0.0.1:0"},
			{Listener: extraLn, Middleware: testMiddleware},
		},
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	}

	svr, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go svr.Start()
	for svr.GetListener() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	// Hit the listener with middleware — it should fire.
	resp, err := http.Get("http://" + extraLn.Addr().String() + "/health")
	if err != nil {
		t.Fatalf("Failed to reach listener with middleware: %v", err)
	}
	resp.Body.Close()
	if !middlewareCalled {
		t.Error("Middleware should have been called on listener")
	}

	svr.Shutdown()
}

// TestDeprecatedListenAddress verifies backward compat with ListenAddress.
func TestDeprecatedListenAddress(t *testing.T) {
	config := &ServerConfig{
		ListenAddress:     "127.0.0.1:0",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	}

	svr, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go svr.Start()
	for svr.GetListener() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	url := "http://" + svr.GetListener().Addr().String() + "/health"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to reach server via deprecated ListenAddress: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	svr.Shutdown()
}
