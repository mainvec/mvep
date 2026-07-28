package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// testTimeout bounds every lifecycle assertion so a regression fails the test
// instead of hanging the suite.
const testTimeout = 10 * time.Second

// newTestServer builds a server with a health endpoint on ephemeral loopback
// listeners and guarantees teardown.
func newTestServer(t *testing.T, listeners ...ListenerConfig) *Server {
	t.Helper()

	if len(listeners) == 0 {
		listeners = []ListenerConfig{{Address: "127.0.0.1:0"}}
	}

	svr, err := NewServer(&ServerConfig{
		Listeners:         listeners,
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })

	return svr
}

// mustGetHealth asserts the health endpoint on the given listener answers 200.
func mustGetHealth(t *testing.T, ln net.Listener) {
	t.Helper()

	resp, err := http.Get("http://" + ln.Addr().String() + "/health")
	if err != nil {
		t.Fatalf("GET /health on %s: %v", ln.Addr(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health on %s: expected 200, got %d", ln.Addr(), resp.StatusCode)
	}
}

// awaitDone fails the test if the server has not reached a terminal state.
func awaitDone(t *testing.T, svr *Server) {
	t.Helper()

	select {
	case <-svr.Done():
	case <-time.After(testTimeout):
		t.Fatal("server did not reach a terminal state before the deadline")
	}
}

// TestStartAsyncReadyWithoutPolling verifies StartAsync returns only once every
// listener is bound and immediately serving.
func TestStartAsyncReadyWithoutPolling(t *testing.T) {
	svr := newTestServer(t)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	ln := svr.GetListener()
	if ln == nil {
		t.Fatal("GetListener returned nil after StartAsync")
	}

	// No sleep or readiness poll: the listener must already answer.
	mustGetHealth(t, ln)

	if err := svr.Err(); err != nil {
		t.Fatalf("Err on a running server: expected nil, got %v", err)
	}

	select {
	case <-svr.Done():
		t.Fatal("Done closed while the server is still running")
	default:
	}
}

// TestBlockingStartReturnsAfterShutdown verifies programmatic shutdown releases
// a blocking Start.
func TestBlockingStartReturnsAfterShutdown(t *testing.T) {
	svr := newTestServer(t)

	startErr := make(chan error, 1)
	go func() { startErr <- svr.Start() }()

	waitForListener(t, svr)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := svr.ShutdownContext(ctx); err != nil {
		t.Fatalf("ShutdownContext: %v", err)
	}

	select {
	case err := <-startErr:
		if err != nil {
			t.Fatalf("Start returned an unexpected error: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Start did not return after ShutdownContext")
	}
}

// waitForListener blocks until a concurrently-started server is serving.
func waitForListener(t *testing.T, svr *Server) net.Listener {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		if ln := svr.GetListener(); ln != nil {
			return ln
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("server never bound a listener")
	return nil
}

// TestShutdownContextIsIdempotent verifies concurrent shutdown callers all
// observe the same completed lifecycle without racing or panicking.
func TestShutdownContextIsIdempotent(t *testing.T) {
	svr := newTestServer(t)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	const callers = 16

	var wg sync.WaitGroup
	errs := make([]error, callers)

	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			errs[i] = svr.ShutdownContext(ctx)
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("concurrent ShutdownContext callers did not all return")
	}

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent ShutdownContext caller %d: %v", i, err)
		}
	}

	awaitDone(t, svr)
}

// TestDoneClosesOnceAndWaitIsRepeatable verifies multiple waiters are released
// and Wait keeps returning the same terminal result.
func TestDoneClosesOnceAndWaitIsRepeatable(t *testing.T) {
	svr := newTestServer(t)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	const waiters = 8

	var wg sync.WaitGroup
	results := make([]error, waiters)

	for i := range waiters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = svr.Wait()
		}()
	}

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	released := make(chan struct{})
	go func() {
		wg.Wait()
		close(released)
	}()

	select {
	case <-released:
	case <-time.After(testTimeout):
		t.Fatal("not every Wait caller was released by shutdown")
	}

	for i, err := range results {
		if err != nil {
			t.Errorf("Wait caller %d: expected nil, got %v", i, err)
		}
	}

	// Wait remains callable and stable after the lifecycle ends.
	if err := svr.Wait(); err != nil {
		t.Errorf("Wait after shutdown: expected nil, got %v", err)
	}
	if err := svr.Err(); err != nil {
		t.Errorf("Err after shutdown: expected nil, got %v", err)
	}

	// Done is already closed and must not block or re-close.
	select {
	case <-svr.Done():
	default:
		t.Error("Done did not stay closed after shutdown")
	}
}

// TestGracefulShutdownDrainsActiveHandler verifies shutdown waits for in-flight
// requests instead of dropping them.
func TestGracefulShutdownDrainsActiveHandler(t *testing.T) {
	svr := newTestServer(t)

	handlerEntered := make(chan struct{})
	releaseHandler := make(chan struct{})

	svr.Handle("/slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerEntered)
		<-releaseHandler
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	respStatus := make(chan int, 1)
	reqErr := make(chan error, 1)

	go func() {
		resp, err := http.Get("http://" + svr.GetListener().Addr().String() + "/slow")
		if err != nil {
			reqErr <- err
			return
		}
		defer resp.Body.Close()
		respStatus <- resp.StatusCode
	}()

	select {
	case <-handlerEntered:
	case err := <-reqErr:
		t.Fatalf("request to /slow failed before the handler ran: %v", err)
	case <-time.After(testTimeout):
		t.Fatal("handler never started")
	}

	shutdownReturned := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		shutdownReturned <- svr.ShutdownContext(ctx)
	}()

	// Shutdown must still be draining while the handler is blocked.
	select {
	case err := <-shutdownReturned:
		t.Fatalf("ShutdownContext returned while a handler was active: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseHandler)

	select {
	case err := <-shutdownReturned:
		if err != nil {
			t.Fatalf("ShutdownContext after drain: %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("ShutdownContext did not return after the handler completed")
	}

	select {
	case code := <-respStatus:
		if code != http.StatusOK {
			t.Errorf("drained request: expected 200, got %d", code)
		}
	case err := <-reqErr:
		t.Errorf("drained request failed: %v", err)
	case <-time.After(testTimeout):
		t.Error("drained request never completed")
	}
}

// TestShutdownDeadlineForcesClose verifies an expired context force-closes the
// server and surfaces a context-bearing error.
func TestShutdownDeadlineForcesClose(t *testing.T) {
	svr := newTestServer(t)

	releaseHandler := make(chan struct{})
	t.Cleanup(func() { close(releaseHandler) })

	handlerEntered := make(chan struct{})

	svr.Handle("/stuck", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerEntered)
		<-releaseHandler
	}))

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	go func() {
		resp, err := http.Get("http://" + svr.GetListener().Addr().String() + "/stuck")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-handlerEntered:
	case <-time.After(testTimeout):
		t.Fatal("handler never started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	shutdownErr := svr.ShutdownContext(ctx)
	if shutdownErr == nil {
		t.Fatal("ShutdownContext: expected a deadline error, got nil")
	}
	if !errors.Is(shutdownErr, context.DeadlineExceeded) {
		t.Fatalf("ShutdownContext: expected context.DeadlineExceeded, got %v", shutdownErr)
	}

	// The lifecycle must still complete despite the unresponsive handler.
	awaitDone(t, svr)
}

// TestAsyncStopHandlerStopsServer verifies the documented response-first stop
// endpoint pattern does not deadlock.
func TestAsyncStopHandlerStopsServer(t *testing.T) {
	svr := newTestServer(t)

	svr.Handle("/stop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stopping\n"))

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			_ = svr.ShutdownContext(ctx)
		}()
	}))

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	resp, err := http.Get("http://" + svr.GetListener().Addr().String() + "/stop")
	if err != nil {
		t.Fatalf("GET /stop: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /stop: expected 200, got %d", resp.StatusCode)
	}

	awaitDone(t, svr)

	if err := svr.Wait(); err != nil {
		t.Errorf("Wait after async stop: expected nil, got %v", err)
	}
}

// TestAllListenersServeAndStop verifies multi-listener servers are fully
// reachable and fully stopped.
func TestAllListenersServeAndStop(t *testing.T) {
	preCreated, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-created listener: %v", err)
	}

	svr := newTestServer(t,
		ListenerConfig{Address: "127.0.0.1:0"},
		ListenerConfig{Listener: preCreated},
	)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	listeners := svr.GetListeners()
	if len(listeners) != 2 {
		t.Fatalf("GetListeners: expected 2 listeners, got %d", len(listeners))
	}

	addrs := make([]string, len(listeners))
	for i, ln := range listeners {
		mustGetHealth(t, ln)
		addrs[i] = ln.Addr().String()
	}

	if addrs[1] != preCreated.Addr().String() {
		t.Errorf("GetListeners order: expected %s at index 1, got %s",
			preCreated.Addr(), addrs[1])
	}

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Every listener must be closed, not just the primary one.
	for _, addr := range addrs {
		if _, err := http.Get("http://" + addr + "/health"); err == nil {
			t.Errorf("listener %s still accepts connections after shutdown", addr)
		}
	}
}

// TestGetListenersReturnsDefensiveCopy verifies callers cannot corrupt server
// state through the returned slice.
func TestGetListenersReturnsDefensiveCopy(t *testing.T) {
	svr := newTestServer(t,
		ListenerConfig{Address: "127.0.0.1:0"},
		ListenerConfig{Address: "127.0.0.1:0"},
	)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	first := svr.GetListeners()
	if len(first) != 2 {
		t.Fatalf("GetListeners: expected 2 listeners, got %d", len(first))
	}

	original := first[0]
	first[0] = nil

	second := svr.GetListeners()
	if len(second) != 2 {
		t.Fatalf("GetListeners after mutation: expected 2 listeners, got %d", len(second))
	}
	if second[0] != original {
		t.Error("GetListeners exposed its backing array to callers")
	}
	if svr.GetListener() != original {
		t.Error("GetListener changed after the returned slice was mutated")
	}
}

// TestFatalServeErrorStopsAllListeners verifies an unexpected serve failure on
// one listener is reported and tears the whole server down.
func TestFatalServeErrorStopsAllListeners(t *testing.T) {
	failing := &failingListener{
		addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0},
		err:  errors.New("injected accept failure"),
	}

	svr := newTestServer(t,
		ListenerConfig{Address: "127.0.0.1:0"},
		ListenerConfig{Listener: failing},
	)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	healthy := svr.GetListeners()[0]
	healthyAddr := healthy.Addr().String()

	waitErr := make(chan error, 1)
	go func() { waitErr <- svr.Wait() }()

	select {
	case err := <-waitErr:
		if err == nil {
			t.Fatal("Wait: expected the injected serve failure, got nil")
		}
		if !errors.Is(err, failing.err) {
			t.Fatalf("Wait: expected the injected serve failure, got %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Wait was not released by the fatal serve error")
	}

	if err := svr.Err(); !errors.Is(err, failing.err) {
		t.Errorf("Err: expected the injected serve failure, got %v", err)
	}

	// The healthy listener must have been stopped too.
	if _, err := http.Get("http://" + healthyAddr + "/health"); err == nil {
		t.Error("healthy listener still accepts connections after a fatal serve error")
	}
}

// failingListener returns a permanent error from Accept to simulate a fatal
// serve failure on one listener of a multi-listener server.
type failingListener struct {
	addr net.Addr
	err  error
}

func (l *failingListener) Accept() (net.Conn, error) { return nil, l.err }

func (l *failingListener) Close() error { return nil }

func (l *failingListener) Addr() net.Addr { return l.addr }

// TestBindFailureRollsBackEarlierListeners verifies a partial startup does not
// leak listeners that were already bound.
func TestBindFailureRollsBackEarlierListeners(t *testing.T) {
	// Occupy a port so the second bind is guaranteed to fail.
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupied listener: %v", err)
	}
	defer occupied.Close()

	first, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("first listener: %v", err)
	}

	svr, err := NewServer(&ServerConfig{
		Listeners: []ListenerConfig{
			{Listener: first},
			{Address: occupied.Addr().String()},
		},
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })

	if err := svr.StartAsync(); err == nil {
		t.Fatal("StartAsync: expected a bind failure, got nil")
	}

	// The already-acquired listener must have been closed by the rollback.
	if _, err := first.Accept(); err == nil {
		t.Error("the first listener was left open after a rollback")
	}

	if len(svr.GetListeners()) != 0 {
		t.Errorf("GetListeners after rollback: expected 0, got %d", len(svr.GetListeners()))
	}
	if svr.GetListener() != nil {
		t.Error("GetListener returned a listener after a rollback")
	}

	awaitDone(t, svr)

	if svr.Err() == nil {
		t.Error("Err: expected the startup failure to be recorded")
	}
}

// TestShutdownBeforeStartPreventsStart verifies pre-start shutdown is safe,
// terminal, and releases waiters.
func TestShutdownBeforeStartPreventsStart(t *testing.T) {
	svr := newTestServer(t)

	waitErr := make(chan error, 1)
	go func() { waitErr <- svr.Wait() }()

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown before start: %v", err)
	}
	// Idempotent even before start.
	if err := svr.Shutdown(); err != nil {
		t.Fatalf("second Shutdown before start: %v", err)
	}

	select {
	case err := <-waitErr:
		if err != nil {
			t.Errorf("Wait after pre-start shutdown: expected nil, got %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatal("Wait was not released by a pre-start shutdown")
	}

	if err := svr.StartAsync(); !errors.Is(err, ErrServerStopped) {
		t.Errorf("StartAsync after shutdown: expected ErrServerStopped, got %v", err)
	}
	if err := svr.Start(); !errors.Is(err, ErrServerStopped) {
		t.Errorf("Start after shutdown: expected ErrServerStopped, got %v", err)
	}
	if svr.GetListener() != nil {
		t.Error("GetListener returned a listener for a server that never started")
	}
}

// TestRepeatedStartIsRejected verifies the server is single-use.
func TestRepeatedStartIsRejected(t *testing.T) {
	svr := newTestServer(t)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	if err := svr.StartAsync(); !errors.Is(err, ErrServerStarted) {
		t.Errorf("second StartAsync: expected ErrServerStarted, got %v", err)
	}
	if err := svr.Start(); !errors.Is(err, ErrServerStarted) {
		t.Errorf("Start on a running server: expected ErrServerStarted, got %v", err)
	}

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if err := svr.StartAsync(); !errors.Is(err, ErrServerStopped) {
		t.Errorf("StartAsync after shutdown: expected ErrServerStopped, got %v", err)
	}
}

// TestNewServerCopiesConfig verifies the server does not retain or mutate the
// caller's configuration.
func TestNewServerCopiesConfig(t *testing.T) {
	preCreated, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-created listener: %v", err)
	}

	config := &ServerConfig{
		Listeners:         []ListenerConfig{{Listener: preCreated}},
		EnableHealthCheck: true,
	}

	svr, err := NewServer(config)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = svr.Shutdown() })

	if config.HealthCheckPath != "" {
		t.Errorf("NewServer mutated the caller's config: HealthCheckPath = %q", config.HealthCheckPath)
	}

	// Mutating the caller's config after construction must not affect the server.
	config.BasePath = "/mutated"
	config.EnableCORS = true
	config.Listeners[0] = ListenerConfig{Address: "127.0.0.1:1"}

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	listeners := svr.GetListeners()
	if len(listeners) != 1 {
		t.Fatalf("GetListeners: expected 1 listener, got %d", len(listeners))
	}
	if listeners[0] != preCreated {
		t.Fatalf("server bound %s instead of the configured pre-created listener %s",
			listeners[0].Addr(), preCreated.Addr())
	}

	// The default health path resolved at construction time must still apply.
	mustGetHealth(t, preCreated)
}

// TestDefaultShutdownTimeoutIsBounded documents the compatibility wrapper's
// budget so consumers can reason about it.
func TestDefaultShutdownTimeoutIsBounded(t *testing.T) {
	if DefaultShutdownTimeout <= 0 {
		t.Fatalf("DefaultShutdownTimeout must be positive, got %v", DefaultShutdownTimeout)
	}
	if DefaultShutdownTimeout > time.Minute {
		t.Fatalf("DefaultShutdownTimeout should stay under common supervisor stop budgets, got %v",
			DefaultShutdownTimeout)
	}
}

// TestServerHasNoSignalHandling verifies a running server ignores the process's
// signal disposition: only explicit lifecycle calls stop it.
func TestServerHasNoSignalHandling(t *testing.T) {
	svr := newTestServer(t)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	// Nothing but an explicit shutdown may terminate the lifecycle.
	select {
	case <-svr.Done():
		t.Fatal("server terminated without an explicit shutdown")
	case <-time.After(200 * time.Millisecond):
	}

	mustGetHealth(t, svr.GetListener())

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	awaitDone(t, svr)
}

// TestConcurrentLifecycleAccess exercises the accessors against an active
// shutdown so the race detector can observe unsynchronized state.
func TestConcurrentLifecycleAccess(t *testing.T) {
	svr := newTestServer(t,
		ListenerConfig{Address: "127.0.0.1:0"},
		ListenerConfig{Address: "127.0.0.1:0"},
	)

	if err := svr.StartAsync(); err != nil {
		t.Fatalf("StartAsync: %v", err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = svr.GetListener()
					_ = svr.GetListeners()
					_ = svr.Err()
				}
			}
		}()
	}

	if err := svr.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	close(stop)
	wg.Wait()

	if err := svr.Wait(); err != nil {
		t.Errorf("Wait: expected nil, got %v", err)
	}
}
