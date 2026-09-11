package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"
)

// The process's lifecycle constants live together here, with what each one
// is traded against, so a future change to the pipeline's slowest path can
// revisit them without re-deriving the reasoning. They are starting values
// chosen against the work as it exists today, not measured limits.
const (
	// DefaultAddr is what NewServer listens on when RouterConfig.Addr is
	// empty. Binding 0.0.0.0 inside the container is not the LAN-reachable
	// opt-in: the published port is what decides reachability, and
	// docker-compose.yml maps it to 127.0.0.1 unless BIND_ADDR says
	// otherwise (issue #57, ADR-0004).
	DefaultAddr = "0.0.0.0:8080"

	// DefaultReadHeaderTimeout is hard, and is the one limit that answers
	// a connection which opens and then does nothing: no client on
	// localhost or a LAN needs longer than this to send request headers.
	DefaultReadHeaderTimeout = 10 * time.Second

	// DefaultReadTimeout covers headers plus body. The largest real body
	// is a pasted Job Description (kilobytes), so this is generous by
	// orders of magnitude and still bounded.
	DefaultReadTimeout = 60 * time.Second

	// DefaultIdleTimeout reaps keep-alive connections between requests, so
	// connection count tracks actual use.
	DefaultIdleTimeout = 120 * time.Second

	// DefaultDrainTimeout is how long Serve waits for in-flight requests
	// after a stop signal. Long enough for a Claude call that is nearly
	// done to land and for a typst compile (seconds, ADR-0012) to finish;
	// short enough that stopping the backend stays a quick action. A
	// Generation that has only just started does not survive a shutdown,
	// and should not — waiting minutes for it would make the window
	// useless as a bound.
	DefaultDrainTimeout = 15 * time.Second
)

// Run listens on srv.Addr and serves it until ctx is cancelled, then
// drains as Serve documents. It is the whole of the process's lifecycle,
// so main stays a thin wrapper over reading the environment and calling
// it. An error means the listener could not be opened or serving failed —
// a graceful stop returns nil.
func Run(ctx context.Context, srv *http.Server, drainTimeout time.Duration) error {
	addr := srv.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return Serve(ctx, srv, ln, drainTimeout)
}

// Serve serves ln with srv until ctx is cancelled — in the running process
// that is SIGINT or SIGTERM, via signal.NotifyContext — and then stops
// gracefully: it closes the listener so no new connection is accepted,
// gives requests already in flight up to drainTimeout to finish, and only
// then returns.
//
// Server.Shutdown waits for active handlers but does not cancel their
// request contexts, so Serve cancels the base context every request is
// derived from once the drain is over (cleanly or expired). Without that,
// a request mid-Claude-call would block the drain for its full
// multi-minute duration, the window would expire, and the process would
// exit killing it anyway — today's behaviour with extra steps. With it,
// the outbound call is cancelled and unwinds.
//
// A stop is a routine action, not a failure: Serve returns nil both when
// the drain completes and when it expires, logging which of the two
// happened. It returns an error only when serving itself failed.
//
// drainTimeout is a parameter rather than a constant so tests can inject a
// short window; callers in the running process pass DefaultDrainTimeout. A
// value <= 0 means DefaultDrainTimeout.
func Serve(ctx context.Context, srv *http.Server, ln net.Listener, drainTimeout time.Duration) error {
	if drainTimeout <= 0 {
		drainTimeout = DefaultDrainTimeout
	}

	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	srv.BaseContext = func(net.Listener) context.Context { return baseCtx }

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Printf("shutdown signal received: no longer accepting connections, draining in-flight requests (up to %s)", drainTimeout)
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer cancelDrain()
	err := srv.Shutdown(drainCtx)
	// Whatever the drain's outcome, cancel the contexts of anything still
	// running so it stops instead of being severed at the socket.
	cancelBase()

	switch {
	case err == nil:
		log.Print("drain complete: all in-flight requests finished, shutting down")
	case errors.Is(err, context.DeadlineExceeded):
		log.Printf("drain window (%s) expired with requests still in flight: cancelling them and shutting down", drainTimeout)
	default:
		return err
	}
	return nil
}

// NewServer builds the app's *http.Server from cfg: the handler NewRouter
// returns, wrapped in explicit connection limits rather than
// http.ListenAndServe's defaults, which impose none at all. See
// RouterConfig for what each field controls. Use Run to serve it, which
// owns the signal-and-drain loop.
func NewServer(cfg RouterConfig) *http.Server {
	addr := cfg.Addr
	if addr == "" {
		addr = DefaultAddr
	}

	return &http.Server{
		Addr:              addr,
		Handler:           NewRouter(cfg),
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		IdleTimeout:       DefaultIdleTimeout,

		// WriteTimeout is set to 0 explicitly, not left to the zero value
		// by omission, because the absence is the decision.
		//
		// Go runs WriteTimeout from the end of the header read to the end
		// of the response write, so it caps total handler duration. POST
		// /api/generations makes four Claude calls (Selection+Rewrite,
		// Cover Letter drafting, then RAL estimation's two-call
		// web-search pattern — ADR-0011) and POST /api/job-listings adds
		// RAL resolution plus Application Method inference at save time;
		// both legitimately run for minutes. Any value tight enough to be
		// a meaningful limit (30-60s) breaks them, and any value loose
		// enough to be safe (10+ minutes) protects nothing. Its failure
		// mode is also the wrong one here: it tears down the connection
		// but the handler keeps running, so the user sees a network error
		// while the server goes on spending Claude tokens.
		//
		// The Claude-calling routes get a request context deadline
		// instead (see withRequestDeadline), which actually stops the
		// outbound work rather than only closing the socket.
		WriteTimeout: 0,
	}
}
