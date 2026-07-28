package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// DefaultShutdownTimeout bounds Shutdown and every shutdown the server
// initiates itself (for example after a fatal Serve error), where no caller
// context is available. It is deliberately shorter than common supervisor stop
// budgets such as systemd's 90s TimeoutStopSec.
const DefaultShutdownTimeout = 30 * time.Second

// defaultReadHeaderTimeout bounds how long a client may take to send request
// headers, mitigating Slowloris-style connection exhaustion.
const defaultReadHeaderTimeout = 10 * time.Second

var (
	// ErrServerStarted is returned when Start or StartAsync is called on a
	// server that is already starting or running. A Server is single-use.
	ErrServerStarted = errors.New("mvep/server: server already started")

	// ErrServerStopped is returned when Start or StartAsync is called on a
	// server that has already been shut down. A Server is single-use.
	ErrServerStopped = errors.New("mvep/server: server already stopped")
)

// ListenerConfig describes a listener the server should serve on.
type ListenerConfig struct {
	// Address to listen on (e.g., "127.0.0.1:8080", "unix:///tmp/socket").
	// Ignored when Listener is set.
	Address string
	// Listener allows passing a pre-created net.Listener (e.g. a TLS listener).
	// When set, Address is ignored. The server takes ownership of the listener
	// once Start or StartAsync begins and closes it during shutdown or during
	// startup rollback.
	Listener net.Listener
	// Middleware wraps the server mux for this listener (e.g. LocalTrustMiddleware).
	Middleware func(http.Handler) http.Handler
}

// ServerConfig holds configuration for the MVEP server.
//
// NewServer copies the configuration. Mutating the struct passed to NewServer
// afterwards has no effect on the server, and NewServer does not modify it.
type ServerConfig struct {
	// Listeners defines the set of listeners the server will serve on.
	// Each can specify Address (to auto-create) or Listener (pre-created),
	// with optional per-listener Middleware.
	Listeners []ListenerConfig
	// Deprecated: ListenAddress is kept for backward compatibility.
	// Use Listeners instead. If Listeners is empty and ListenAddress is
	// set, it is auto-converted to a single Listeners entry with a warning.
	ListenAddress string
	// BasePath is the base URL path for endpoints (e.g., "/api")
	BasePath string
	// EnableHealthCheck adds a /health endpoint if true
	EnableHealthCheck bool
	// HealthCheckPath is the path for the health check endpoint (default: "/health")
	HealthCheckPath string
	// EnableCORS adds CORS headers to all responses if true
	EnableCORS bool
	// Interceptor is the global interceptor chain applied to all commands
	Interceptor mvep.CmdInterceptor
}

// serverState tracks the single-use lifecycle of a Server.
type serverState uint8

const (
	stateNew serverState = iota
	stateStarting
	stateRunning
	stateStopping
	stateStopped
)

// Server represents an MVEP package server.
//
// The server owns the HTTP surface only. It installs no process signal
// handlers: the owning application decides which signals matter and calls
// ShutdownContext as one explicit step of its own shutdown sequence.
//
// A Server is single-use. Once it has stopped it cannot be restarted;
// construct a new Server instead.
type Server struct {
	config   ServerConfig
	mux      *http.ServeMux
	packages []*PackageRegistration

	// lifecycleMu guards the state machine and the listener/httpServer slices.
	// It is never held while draining connections or waiting on goroutines.
	lifecycleMu sync.Mutex
	state       serverState
	listeners   []net.Listener
	httpServers []*http.Server

	serveWG  sync.WaitGroup
	done     chan struct{}
	doneOnce sync.Once

	errMu sync.Mutex
	err   error
}

// PackageRegistration represents a registered package with its command runner
type PackageRegistration struct {
	Package       mvep.Package
	CommandRunner mvep.CommandRunner
}

// NewServer creates a new MVEP server with the given configuration.
//
// The configuration is copied, so the caller may reuse or modify the passed
// struct afterwards without affecting the server.
func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	cfg := *config
	cfg.Listeners = slices.Clone(config.Listeners)
	if cfg.EnableHealthCheck && len(cfg.HealthCheckPath) == 0 {
		cfg.HealthCheckPath = "/health"
	}

	return &Server{
		config:   cfg,
		mux:      http.NewServeMux(),
		packages: make([]*PackageRegistration, 0),
		done:     make(chan struct{}),
	}, nil
}

// RegisterPackage registers an MVP package with its command runner
func (s *Server) RegisterPackage(pkg mvep.Package, runner mvep.CommandRunner) error {
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
	pkgHandler := &mvep.PackageHandler{
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

// Start binds every configured listener, begins serving, and blocks until the
// server has completely stopped. It is equivalent to StartAsync followed by
// Wait.
//
// Start does not observe process signals. It returns once an owner calls
// Shutdown or ShutdownContext, or once a fatal Serve error tears the server
// down.
func (s *Server) Start() error {
	if err := s.StartAsync(); err != nil {
		return err
	}
	return s.Wait()
}

// StartAsync binds every configured listener synchronously and returns once
// all of them are serving. Listener binding is transactional: if any listener
// fails to resolve or bind, every listener already acquired during the attempt
// is closed and the server transitions straight to its stopped state.
//
// A Server is single-use: StartAsync returns ErrServerStarted if the server is
// already starting or running, and ErrServerStopped once it has stopped.
func (s *Server) StartAsync() error {
	// The lifecycle lock is held for the whole binding phase so a concurrent
	// shutdown observes either a committed startup or a completed rollback.
	s.lifecycleMu.Lock()

	switch s.state {
	case stateStarting, stateRunning:
		s.lifecycleMu.Unlock()
		return ErrServerStarted
	case stateStopping, stateStopped:
		s.lifecycleMu.Unlock()
		return ErrServerStopped
	}

	s.state = stateStarting
	s.registerHealthCheck()

	listeners, httpServers, err := s.bindListeners()
	if err != nil {
		s.state = stateStopped
		s.lifecycleMu.Unlock()

		s.recordError(err)
		s.closeDone()
		return err
	}

	s.listeners = listeners
	s.httpServers = httpServers

	for i := range listeners {
		s.serveWG.Add(1)
		go s.serve(httpServers[i], listeners[i])
		slog.Info("Listener started", "addr", listeners[i].Addr())
	}

	s.state = stateRunning
	s.lifecycleMu.Unlock()

	slog.Info("MVEP Server started", "listeners", len(listeners), "packages", len(s.packages))

	return nil
}

// Wait blocks until the server reaches a terminal state and then returns the
// final lifecycle error. It may be called by any number of goroutines and
// remains callable after the server has stopped.
//
// Wait is released only by a terminal transition: a startup failure, a fatal
// Serve error, or a shutdown. A server that is constructed and never started
// has no terminal transition, so Wait blocks until Shutdown is called.
func (s *Server) Wait() error {
	<-s.done
	return s.Err()
}

// Done returns a channel that is closed exactly once, after the final
// lifecycle error has been recorded.
//
// Like Wait, Done is only closed by a terminal transition. It never closes for
// a server that is neither started nor shut down.
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// Err returns the final lifecycle error. It returns nil while the server is
// starting or running, and after a clean shutdown.
func (s *Server) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

// ShutdownContext stops accepting new connections, drains active requests until
// ctx expires, force-closes anything still open once it does, and waits for the
// server to stop.
//
// Shutdown work runs exactly once. The context supplied by the first caller is
// the graceful shutdown budget; later callers join the same shutdown and may
// stop waiting when their own context expires without changing that budget.
//
// Calling ShutdownContext synchronously from a handler served by this server
// waits for that handler and therefore blocks until ctx expires. Stop endpoints
// must write their response and then trigger shutdown from a new goroutine.
//
// Handlers that ignore their request context cannot be terminated by Go. Such a
// handler keeps running after ShutdownContext returns.
func (s *Server) ShutdownContext(ctx context.Context) error {
	s.lifecycleMu.Lock()

	switch s.state {
	case stateNew:
		// Shutdown before start is terminal and prevents a later start.
		s.state = stateStopped
		s.lifecycleMu.Unlock()
		s.closeDone()
		return nil

	case stateStopping, stateStopped:
		s.lifecycleMu.Unlock()
		return s.joinShutdown(ctx)
	}

	s.state = stateStopping
	httpServers := slices.Clone(s.httpServers)
	s.lifecycleMu.Unlock()

	slog.Info("MVEP Server shutting down", "listeners", len(httpServers))

	err := s.drain(ctx, httpServers)
	s.recordError(err)

	s.lifecycleMu.Lock()
	s.state = stateStopped
	s.lifecycleMu.Unlock()

	s.closeDone()

	return err
}

// Shutdown gracefully shuts down the server using DefaultShutdownTimeout as its
// drain budget. ShutdownContext is preferred where the caller owns the budget.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()
	return s.ShutdownContext(ctx)
}

// GetListener returns the primary listener (useful for getting the actual
// address). It returns nil before the server has bound its listeners.
func (s *Server) GetListener() net.Listener {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if len(s.listeners) == 0 {
		return nil
	}
	return s.listeners[0]
}

// GetListeners returns a copy of every bound listener, in configuration order.
// Mutating the returned slice does not affect the server.
func (s *Server) GetListeners() []net.Listener {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return slices.Clone(s.listeners)
}

// registerHealthCheck installs the health endpoint exactly once, before any
// listener starts serving. It must be called with lifecycleMu held.
func (s *Server) registerHealthCheck() {
	if !s.config.EnableHealthCheck {
		return
	}

	healthPath := s.config.BasePath + s.config.HealthCheckPath
	handler := func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("Health check request")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	if s.config.EnableCORS {
		s.mux.HandleFunc(healthPath, corsFunc(handler))
	} else {
		s.mux.HandleFunc(healthPath, handler)
	}
}

// bindListeners acquires every configured listener and builds its http.Server.
// On failure it closes everything acquired during this attempt so a partial
// startup never leaks listeners. It must be called with lifecycleMu held.
func (s *Server) bindListeners() ([]net.Listener, []*http.Server, error) {
	configs := s.config.Listeners
	if len(configs) == 0 {
		addr := s.config.ListenAddress
		if addr != "" {
			slog.Warn("ServerConfig.ListenAddress is deprecated, use Listeners instead")
		} else {
			addr = "127.0.0.1:8080"
		}
		configs = []ListenerConfig{{Address: addr}}
	}

	listeners := make([]net.Listener, 0, len(configs))
	httpServers := make([]*http.Server, 0, len(configs))

	for _, lc := range configs {
		ln := lc.Listener
		if ln == nil {
			options, err := parseListenAddr(lc.Address)
			if err != nil {
				closeListeners(listeners)
				return nil, nil, err
			}
			ln, err = net.Listen(options.Network(), options.String())
			if err != nil {
				closeListeners(listeners)
				return nil, nil, err
			}
		}

		handler := http.Handler(s.mux)
		if lc.Middleware != nil {
			handler = lc.Middleware(s.mux)
		}

		listeners = append(listeners, ln)
		httpServers = append(httpServers, &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		})
	}

	return listeners, httpServers, nil
}

// serve runs one listener and reports any unexpected failure as fatal.
func (s *Server) serve(httpServer *http.Server, ln net.Listener) {
	defer s.serveWG.Done()

	err := httpServer.Serve(ln)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		// Expected during a normal shutdown.
		return
	}

	slog.Error("Listener stopped unexpectedly", "addr", ln.Addr(), "error", err)
	s.recordError(err)

	// A multi-listener server is one service lifecycle, so a single failed
	// listener stops the rest. This runs in its own goroutine because shutdown
	// waits for this serve goroutine to finish.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
		defer cancel()
		_ = s.ShutdownContext(ctx)
	}()
}

// drain shuts every http.Server down concurrently, force-closes any that do not
// finish within ctx, and waits for all serve goroutines to return.
func (s *Server) drain(ctx context.Context, httpServers []*http.Server) error {
	var (
		mu       sync.Mutex
		drainErr error
		wg       sync.WaitGroup
	)

	for _, httpServer := range httpServers {
		wg.Add(1)
		go func(httpServer *http.Server) {
			defer wg.Done()

			err := httpServer.Shutdown(ctx)
			if err == nil {
				return
			}

			// The graceful budget expired: bound the lifecycle by force-closing
			// whatever is still open.
			if closeErr := httpServer.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				err = errors.Join(err, closeErr)
			}

			mu.Lock()
			drainErr = errors.Join(drainErr, err)
			mu.Unlock()
		}(httpServer)
	}

	wg.Wait()
	s.serveWG.Wait()

	return drainErr
}

// joinShutdown waits for an already-running shutdown to finish, or for the
// caller's own context to expire.
func (s *Server) joinShutdown(ctx context.Context) error {
	select {
	case <-s.done:
		return s.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// recordError accumulates lifecycle errors without losing earlier ones.
func (s *Server) recordError(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	s.err = errors.Join(s.err, err)
}

// closeDone publishes the terminal state exactly once. The final error is
// always recorded before this runs.
func (s *Server) closeDone() {
	s.doneOnce.Do(func() { close(s.done) })
}

// closeListeners closes every listener, used for startup rollback.
func closeListeners(listeners []net.Listener) {
	for _, ln := range listeners {
		if err := ln.Close(); err != nil {
			slog.Warn("Failed to close listener during startup rollback",
				"addr", ln.Addr(), "error", err)
		}
	}
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
