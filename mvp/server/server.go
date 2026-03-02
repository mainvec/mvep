package server

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mainvec/mvp/mvpgo/mvp"
)

// ServerConfig holds configuration for the MVP server
type ServerConfig struct {
	// ListenAddress is the address to listen on (e.g., "127.0.0.1:8080", "unix:///tmp/socket")
	ListenAddress string
	// BasePath is the base URL path for endpoints (e.g., "/api")
	BasePath string
	// EnableHealthCheck adds a /health endpoint if true
	EnableHealthCheck bool
	// HealthCheckPath is the path for the health check endpoint (default: "/health")
	HealthCheckPath string
	// EnableCORS adds CORS headers to all responses if true
	EnableCORS bool
	// OnShutdown is called when the server receives a shutdown signal
	OnShutdown func()
	// Interceptor is the global interceptor chain applied to all commands
	Interceptor mvp.CmdInterceptor
	// Listener allows passing a pre-created net.Listener (e.g. a TLS listener).
	// When set, Start() uses this listener instead of creating one from ListenAddress.
	Listener net.Listener
}

// Server represents an MVP package server
type Server struct {
	config   *ServerConfig
	listener net.Listener
	mux      *http.ServeMux
	packages []*PackageRegistration
}

// PackageRegistration represents a registered package with its command runner
type PackageRegistration struct {
	Package       mvp.Package
	CommandRunner mvp.CommandRunner
}

// NewServer creates a new MVP server with the given configuration
func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if len(config.ListenAddress) == 0 {
		config.ListenAddress = "127.0.0.1:8080"
	}
	if config.EnableHealthCheck && len(config.HealthCheckPath) == 0 {
		config.HealthCheckPath = "/health"
	}

	return &Server{
		config:   config,
		mux:      http.NewServeMux(),
		packages: make([]*PackageRegistration, 0),
	}, nil
}

// RegisterPackage registers an MVP package with its command runner
func (s *Server) RegisterPackage(pkg mvp.Package, runner mvp.CommandRunner) error {
	if pkg == nil {
		return errors.New("package is required")
	}
	if runner == nil {
		return errors.New("command runner is required")
	}

	registration := &PackageRegistration{
		Package:       pkg,
		CommandRunner: runner,
	}
	s.packages = append(s.packages, registration)

	// Create the package handler
	pkgHandler := &mvp.PackageHandler{
		Package:       pkg,
		CommandRunner: runner,
		Interceptor:   s.config.Interceptor,
	}

	// Register the command endpoint
	cmdPath := s.config.BasePath + "/" + pkg.GetName() + "/cmd"
	if s.config.EnableCORS {
		s.mux.Handle(cmdPath, corsHandler(pkgHandler))
	} else {
		s.mux.Handle(cmdPath, pkgHandler)
	}

	slog.Info("Registered package", "name", pkg.GetName(), "path", cmdPath)

	return nil
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// Start starts the server
func (s *Server) Start() error {
	var ln net.Listener

	if s.config.Listener != nil {
		// Use pre-created listener (e.g. TLS)
		ln = s.config.Listener
	} else {
		options, err := parseListenAddr(s.config.ListenAddress)
		if err != nil {
			return err
		}

		var listenErr error
		ln, listenErr = net.Listen(options.Network(), options.String())
		if listenErr != nil {
			return listenErr
		}
	}

	s.listener = ln

	// Setup health check if enabled
	if s.config.EnableHealthCheck {
		healthPath := s.config.BasePath + s.config.HealthCheckPath
		if s.config.EnableCORS {
			s.mux.HandleFunc(healthPath, corsFunc(func(w http.ResponseWriter, r *http.Request) {
				slog.Debug("Health check request")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
		} else {
			s.mux.HandleFunc(healthPath, func(w http.ResponseWriter, r *http.Request) {
				slog.Debug("Health check request")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
		}
	}

	slog.Info("MVP Server started", "addr", ln.Addr(), "packages", len(s.packages))

	// Start HTTP server in goroutine
	go func() {
		http.Serve(ln, s.mux)
	}()

	// Wait for shutdown signal
	quitChan := make(chan bool)
	go waitForShutdown(quitChan, s.config.OnShutdown)
	<-quitChan

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// GetListener returns the current listener (useful for getting the actual address)
func (s *Server) GetListener() net.Listener {
	return s.listener
}

// parseListenAddr parses the listen address and returns a net.Addr.
// Supported schemes: unix://, tcp://, http://, https://, or bare host:port.
func parseListenAddr(addr string) (net.Addr, error) {
	if strings.HasPrefix(addr, "unix://") {
		socketFile := strings.TrimPrefix(addr, "unix://")
		address, err := net.ResolveUnixAddr("unix", socketFile)
		if err != nil {
			return nil, err
		}
		return address, nil
	}
	// Strip known scheme prefixes to get bare host:port
	hostPort := addr
	switch {
	case strings.HasPrefix(hostPort, "https://"):
		hostPort = strings.TrimPrefix(hostPort, "https://")
	case strings.HasPrefix(hostPort, "http://"):
		hostPort = strings.TrimPrefix(hostPort, "http://")
	case strings.HasPrefix(hostPort, "tcp://"):
		hostPort = strings.TrimPrefix(hostPort, "tcp://")
	}
	address, err := net.ResolveTCPAddr("tcp", hostPort)
	if err != nil {
		return nil, err
	}
	return address, nil
}

// corsFunc wraps a handler function with CORS headers
func corsFunc(nextFunc func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		nextFunc(w, r)
	}
}

// corsHandler wraps an http.Handler with CORS headers
func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// waitForShutdown waits for OS signals and triggers shutdown
func waitForShutdown(quitSignal chan bool, onShutdown func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	slog.Info("Shutting down...")
	if onShutdown != nil {
		onShutdown()
	}
	quitSignal <- true
}
