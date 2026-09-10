// Package freshness classifies whether a Job Listing's source URL is still
// live, per PRD "Job Description link-rot / staleness check". It is a
// small, independent module: the only thing it shares with
// tracking.downloadLogoBestEffort (ADR-0013's Company Logo download) or
// atsboard's ATS fetchers is the general-purpose injectable-HTTP-doer
// pattern — no code, so a bug in one can never affect the other (story
// 13).
package freshness

import (
	"context"
	"net/http"
	"strings"
)

// Status is a Job Listing source URL's classified liveness, as of the most
// recent check. It deliberately excludes a "not yet checked" value — that
// state belongs to the persisted Job Listing record (tracking.JobListing),
// not to a single check's result.
type Status string

const (
	// StatusLive means the check got a successful (2xx) response for what
	// looks like the original posting.
	StatusLive Status = "live"
	// StatusUnreachable means the posting is confidently gone: a 404, or a
	// redirect chain that landed on one (a generic "no longer accepting
	// applications" page these job boards commonly serve as a literal 404).
	StatusUnreachable Status = "unreachable"
	// StatusUnknown covers every ambiguous or erroring outcome that isn't
	// confidently "gone" — auth-walls (401/403 or a redirect to a login
	// page), timeouts, DNS failures, and anything else — so a flaky or
	// access-gated check never wrongly flags a live posting as dead
	// (stories 5, 6).
	StatusUnknown Status = "unknown"
)

// HTTPDoer is the minimal http.Client surface Check depends on, so callers
// (including tests) can inject a fake returning fixture responses instead
// of making live network calls — the same seam atsboard.HTTPDoer and
// tracking.HTTPDoer already establish for their own outbound calls.
// *http.Client satisfies this interface, so production code just passes
// http.DefaultClient (or an *http.Client with a bounded Timeout).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// authWallMarkers are URL path substrings commonly present on a login/SSO
// landing page that a redirect chain might land on. This is a shallow
// URL-shape check, not content parsing — Check stays a plain fetch (story
// 7), never re-scraping or diffing the response body.
var authWallMarkers = []string{"login", "signin", "sign-in", "sso", "authwall"}

// Check is the pure classification seam the PRD's Testing Decisions call
// for: given a doer and a URL, fetch it once and classify the result as
// Live, Unreachable, or Unknown. It never returns an error — any failure to
// even build or send the request is itself classified Unknown, since it's
// exactly the kind of ambiguous outcome that must not be reported as a
// confident "dead" (Unreachable) result.
func Check(ctx context.Context, doer HTTPDoer, targetURL string) Status {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return StatusUnknown
	}

	resp, err := doer.Do(req)
	if err != nil {
		// Covers timeouts, DNS failures, and any other transport-level
		// error (stories 6, 12) — never Unreachable, since none of these
		// confirm the posting itself is gone.
		return StatusUnknown
	}
	defer resp.Body.Close()

	return classify(targetURL, resp)
}

func classify(targetURL string, resp *http.Response) Status {
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return StatusUnreachable
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return StatusUnknown
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if landedOnAuthWall(targetURL, resp) {
			return StatusUnknown
		}
		return StatusLive
	default:
		// Any other status (5xx, unhandled redirect codes, etc.) is
		// ambiguous rather than a confirmed "gone" — Unknown (story 5's
		// "any other ambiguous/erroring outcome").
		return StatusUnknown
	}
}

// landedOnAuthWall reports whether resp's final URL (after following any
// redirect chain — *http.Client's default behavior, and what a fake test
// doer simulates via resp.Request) looks like a login/SSO page rather than
// the original posting, per story 5.
func landedOnAuthWall(targetURL string, resp *http.Response) bool {
	if resp.Request == nil || resp.Request.URL == nil {
		return false
	}
	finalURL := resp.Request.URL.String()
	if finalURL == targetURL {
		return false
	}
	path := strings.ToLower(resp.Request.URL.Path)
	for _, marker := range authWallMarkers {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}
