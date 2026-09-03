// Package atsboard fetches open roles from a company's public ATS job
// board API (Greenhouse, Lever, Ashby) and normalizes them into a common
// Listing shape, per PRD "ATS Job Board Aggregation" and ADR-0007's
// documented-public-API-only boundary.
package atsboard

import (
	"errors"
	"net/http"
)

// ErrBoardNotFound is returned when a provider's public job board API has
// no board matching the given slug (story 6) — distinct from a board that
// exists but currently has zero open roles.
var ErrBoardNotFound = errors.New("atsboard: job board not found")

// Listing is the common intermediate shape every provider's fetch function
// normalizes into, before it becomes a Job Listing (CONTEXT.md's Job
// Listing entry) via the existing Save path.
type Listing struct {
	Title       string `json:"title"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// HTTPDoer is the minimal http.Client surface fetch functions depend on, so
// callers (including tests) can inject a fake returning fixture responses
// instead of making live network calls (per the PRD's Testing Decisions).
// *http.Client satisfies this interface, so production code just passes
// http.DefaultClient.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
