package api

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// DefaultClaudeRouteTimeout is the upper bound on a single Claude-calling
// request. It is the counterpart to NewServer's deliberately absent
// WriteTimeout: ten minutes is far beyond any Generation observed so far
// (four Claude calls, including RAL estimation's two-call web-search
// pattern — ADR-0011), so it never cuts real work short, while still
// stopping a call that will never return from holding a connection and a
// goroutine for the life of the process. Like the drain window, it is a
// starting value chosen against the work as it exists today.
const DefaultClaudeRouteTimeout = 10 * time.Minute

// withRequestDeadline bounds a handler by giving its request context a
// deadline. This is the right mechanism for the Claude-calling routes
// because the generation path is already context-aware end to end —
// generation.Generate threads ctx into SelectAndRewrite, DraftCoverLetter,
// EstimateRAL and the Job Description URL fetch — so the deadline actually
// stops the outbound work instead of only closing the socket, the way
// http.Server.WriteTimeout would.
//
// It costs nothing for the abandoned-request case: a browser that closes
// the tab or reloads the page already cancels r.Context(), and these
// routes now unwind on that too.
func withRequestDeadline(d time.Duration, next http.HandlerFunc) http.HandlerFunc {
	if d <= 0 {
		d = DefaultClaudeRouteTimeout
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next(&deadlineAwareWriter{ResponseWriter: w, ctx: ctx}, r.WithContext(ctx))
	}
}

// deadlineAwareWriter reports a request that ran out of time as 504
// Gateway Timeout rather than as the 500 the handler writes when its
// Claude call comes back with a cancelled context. The handler still
// decides everything else about the response; only that one status is
// re-labelled, and only when the deadline is genuinely the reason.
type deadlineAwareWriter struct {
	http.ResponseWriter
	ctx context.Context
}

func (w *deadlineAwareWriter) WriteHeader(status int) {
	if status == http.StatusInternalServerError && errors.Is(w.ctx.Err(), context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}
	w.ResponseWriter.WriteHeader(status)
}
