package tracking

import "time"

// DefaultStaleThreshold is how long an Application may sit in Sent or
// Interviewing without a Status change before IsStale flags it (story 5).
// Not user-configurable in this PRD (see Out of Scope).
const DefaultStaleThreshold = 14 * 24 * time.Hour

// IsStale reports whether an Application currently in status has gone
// longer than threshold since statusUpdatedAt without a Status change
// (story 4). Only StatusSent and StatusInterviewing are ever stale (story
// 7) — Saved/Tailoring haven't reached a followed-up stage yet, and
// Rejected/Offer are terminal. An empty or unparseable statusUpdatedAt is
// treated as not stale rather than erroring (story 12): that is the
// documented legacy-record rule — an Application saved before the field
// existed and already past Saved has no recoverable date, and migration
// deliberately leaves it empty rather than fabricating a clock (issue #100).
func IsStale(status Status, statusUpdatedAt string, now time.Time, threshold time.Duration) bool {
	if status != StatusSent && status != StatusInterviewing {
		return false
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, statusUpdatedAt)
	if err != nil {
		return false
	}
	return now.Sub(updatedAt) > threshold
}
