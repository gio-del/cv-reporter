package api

import (
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
)

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
