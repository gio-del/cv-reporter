package tracking

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/freshness"
)

// CheckFreshness runs an on-demand liveness check (issue #59, "Job
// Description link-rot / staleness check") against the Job Listing
// identified by id's own source URL, persists the classified result, and
// returns the updated Job Listing. It's on-demand only — triggered by an
// explicit user action via the API handler, never a background poller —
// per the PRD's Testing Decisions.
func CheckFreshness(ctx context.Context, dataDir string, doer HTTPDoer, id string) (JobListing, error) {
	listing, err := getJobListing(dataDir, id)
	if err != nil {
		return JobListing{}, err
	}

	// A Job Listing with no recorded source URL has nothing to check —
	// leave FreshnessStatus/FreshnessCheckedAt untouched rather than firing
	// a request at an empty URL or corrupting a prior result (story 12).
	if strings.TrimSpace(listing.URL) == "" {
		return listing, nil
	}

	status := freshness.Check(ctx, doer, listing.URL)
	listing.FreshnessStatus = toFreshnessStatus(status)
	listing.FreshnessCheckedAt = time.Now().UTC().Format(time.RFC3339Nano)

	if err := os.WriteFile(filepath.Join(dataDir, jobsDir, id+".md"), renderJobListing(listing), 0o644); err != nil {
		return JobListing{}, err
	}
	return listing, nil
}

// toFreshnessStatus maps freshness.Check's pure classification result onto
// the persisted FreshnessStatus enum, which additionally carries
// FreshnessNotYetChecked (a record-level concept with no equivalent in a
// single check's outcome, so it never appears here).
func toFreshnessStatus(s freshness.Status) FreshnessStatus {
	switch s {
	case freshness.StatusLive:
		return FreshnessLive
	case freshness.StatusUnreachable:
		return FreshnessUnreachable
	default:
		return FreshnessUnknown
	}
}
