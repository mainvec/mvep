package server

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/mainvec/mvp/mvpgo/mvp"
)

// Example demonstrates how to create a server using the MVP server
func TestExampleServer(t *testing.T) {
	// Create server configuration
	config := &ServerConfig{
		ListenAddress:     "127.0.0.1:8080",
		BasePath:          "",
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		EnableCORS:        true,
		OnShutdown: func() {
			slog.Info("Custom shutdown handler called")
		},
	}

	// Create the server
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create your package and command runner
	// (This example assumes you have a generated package)
	// myPackage := api.NewPackage()
	// myCommandRunner := implementation.GetCommandRunner()

	// Register your package(s)
	// err = server.RegisterPackage(myPackage, myCommandRunner)
	// if err != nil {
	//     log.Fatalf("Failed to register package: %v", err)
	// }

	// Start the server in a goroutine
	go func() {
		slog.Info("Starting server...")
		err := server.Start()
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Check health endpoint
	resp, err := http.Get("http://127.0.0.1:8080/health")
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

	// Shutdown the server
	err = server.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	// Wait a bit for shutdown to complete
	time.Sleep(100 * time.Millisecond)
}

// TestServerWithInterceptor demonstrates server configuration with middleware
func TestServerWithInterceptor(t *testing.T) {
	interceptorCalled := false
	testInterceptor := func(ctx context.Context, req *mvp.CmdReq, next mvp.CmdHandler) *mvp.CmdResp {
		interceptorCalled = true
		return next(ctx, req)
	}

	config := &ServerConfig{
		ListenAddress:     "127.0.0.1:8082",
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

	interceptor1 := func(ctx context.Context, req *mvp.CmdReq, next mvp.CmdHandler) *mvp.CmdResp {
		callOrder = append(callOrder, "interceptor1-before")
		resp := next(ctx, req)
		callOrder = append(callOrder, "interceptor1-after")
		return resp
	}

	interceptor2 := func(ctx context.Context, req *mvp.CmdReq, next mvp.CmdHandler) *mvp.CmdResp {
		callOrder = append(callOrder, "interceptor2-before")
		resp := next(ctx, req)
		callOrder = append(callOrder, "interceptor2-after")
		return resp
	}

	config := &ServerConfig{
		ListenAddress:     "127.0.0.1:8083",
		BasePath:          "/api",
		EnableHealthCheck: true,
		Interceptor: mvp.Chain(
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
	authInterceptor := func(ctx context.Context, req *mvp.CmdReq, next mvp.CmdHandler) *mvp.CmdResp {
		authCalled = true
		// Simulate auth check
		if req.Headers["auth"] == "" {
			return mvp.NewCmdRespError("unauthorized", "missing auth")
		}
		return next(ctx, req)
	}

	config := &ServerConfig{
		ListenAddress:     "127.0.0.1:8084",
		BasePath:          "/api",
		EnableHealthCheck: true,
		Interceptor: mvp.Chain(
			mvp.LoggingInterceptor(),
			mvp.SkipCommands(authInterceptor, "HealthCheck", "PublicQuery"),
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
