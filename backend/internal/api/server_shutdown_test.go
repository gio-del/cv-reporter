package api_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// The tests in this file assert externally observable lifecycle behaviour
// — a request succeeding, a connection being refused, a context being
// cancelled — and never reach into how the drain is implemented. Timings
// are expressed as "finishes inside the window" / "is cancelled after the
// window" with a deliberately short injected drain, so the suite stays
// fast and doesn't depend on wall-clock luck.

// listenLocal binds a listener on port 0, so tests never collide on a
// fixed port.
func listenLocal(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// TestServe_InFlightRequestCompletesAcrossShutdown is story 1's core
// promise: a request already running when the stop signal arrives gets to
// finish and send its normal response — the Job Listing save that is one
// Claude call from completing is not thrown away.
func TestServe_InFlightRequestCompletesAcrossShutdown(t *testing.T) {
	started := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("finished"))
	})}
	ln := listenLocal(t)
	url := "http://" + ln.Addr().String() + "/api/healthz"

	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- api.Serve(ctx, srv, ln, 5*time.Second) }()

	type result struct {
		status int
		err    error
	}
	responses := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			responses <- result{err: err}
			return
		}
		defer resp.Body.Close()
		responses <- result{status: resp.StatusCode}
	}()

	<-started
	stop() // the SIGTERM equivalent, with the request mid-flight

	got := <-responses
	if got.err != nil {
		t.Fatalf("in-flight request should have completed across shutdown, got error: %v", got.err)
	}
	if got.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", got.status)
	}
	if err := <-served; err != nil {
		t.Fatalf("a graceful stop must not be reported as a failure, got: %v", err)
	}
}

// TestServe_RefusesNewConnectionsOnceShutdownStarted proves the drain
// isn't silently still accepting work: once the stop signal has been
// handled, the listener is closed.
func TestServe_RefusesNewConnectionsOnceShutdownStarted(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})}
	ln := listenLocal(t)
	addr := ln.Addr().String()

	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- api.Serve(ctx, srv, ln, 5*time.Second) }()

	go func() {
		resp, err := http.Get("http://" + addr + "/api/healthz")
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	stop()

	// The listener closes as part of Shutdown, which runs once the stop
	// signal is observed; poll briefly rather than assuming an ordering.
	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			lastErr = err
			break
		}
		conn.Close()
		time.Sleep(10 * time.Millisecond)
	}
	close(release)
	<-served

	if lastErr == nil {
		t.Fatal("expected new connections to be refused once shutdown started, but one was accepted")
	}
}

// TestServe_HandlerOutlivingDrainWindowSeesCancelledContext is story 5:
// when the drain window expires with a request still running, that
// request's context is cancelled so its outbound Claude call stops instead
// of burning tokens for a response nobody will receive.
func TestServe_HandlerOutlivingDrainWindowSeesCancelledContext(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan error, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		cancelled <- r.Context().Err()
	})}
	ln := listenLocal(t)
	addr := ln.Addr().String()

	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	// A 50ms drain window: the handler below outlives it by design.
	go func() { served <- api.Serve(ctx, srv, ln, 50*time.Millisecond) }()

	go func() {
		resp, err := http.Get("http://" + addr + "/api/healthz")
		if err == nil {
			resp.Body.Close()
		}
	}()

	<-started
	stop()

	select {
	case err := <-cancelled:
		if err == nil {
			t.Fatal("expected the request context to report why it was cancelled")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request context was never cancelled after the drain window expired")
	}

	if err := <-served; err != nil {
		t.Fatalf("an expired drain is a bounded stop, not a crash, got: %v", err)
	}
}

// TestServe_NoTraffic_StopReturnsNil covers the ordinary case: stopping an
// idle backend is a routine action that reports success.
func TestServe_NoTraffic_StopReturnsNil(t *testing.T) {
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})}
	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- api.Serve(ctx, srv, listenLocal(t), time.Second) }()

	stop()
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("expected a clean stop to return nil, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the stop signal")
	}
}

// TestRun_UnavailableAddress_ReturnsError keeps a genuine startup failure
// distinguishable from a graceful stop: only the former is an error.
func TestRun_UnavailableAddress_ReturnsError(t *testing.T) {
	taken := listenLocal(t)
	defer taken.Close()

	srv := &http.Server{Addr: taken.Addr().String(), Handler: http.NewServeMux()}
	if err := api.Run(context.Background(), srv, time.Second); err == nil {
		t.Fatal("expected Run to report that the address is already in use")
	}
}
