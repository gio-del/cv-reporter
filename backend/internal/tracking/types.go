// Package tracking implements Job Listing and Application storage and
// lifecycle — the persisted, trackable records CONTEXT.md's Job Listing and
// Application entries describe, stored as flat files alongside Master Data
// (ADR-0008).
package tracking

import "github.com/gio-del/cv-reporter/backend/internal/generation"

// SourceManual is the only Source this PRD can produce — ATS feeds and the
// browser extension (CONTEXT.md's other two Job Listing sources) are out of
// scope here (see ADR-0007).
const SourceManual = "manual"

// JobListing is a persisted, tracked record of a role the user is
// considering, per CONTEXT.md's Job Listing entry.
type JobListing struct {
	ID             string              `json:"id"`
	Company        string              `json:"company"`
	URL            string              `json:"url,omitempty"`
	Source         string              `json:"source"`
	SavedAt        string              `json:"savedAt"`
	JobDescription string              `json:"jobDescription"`
	RAL            generation.RALRange `json:"ral"`
}

// Status is where an Application stands, per CONTEXT.md's Status entry.
type Status string

const (
	StatusSaved        Status = "saved"
	StatusTailoring    Status = "tailoring"
	StatusSent         Status = "sent"
	StatusInterviewing Status = "interviewing"
	StatusRejected     Status = "rejected"
	StatusOffer        Status = "offer"
)

// Application is the tracked record of one attempt to apply to a Job
// Listing — exactly one per Job Listing, created at Status Saved the
// moment it's saved (CONTEXT.md's Application entry, story 2). It shares
// its id with the Job Listing it belongs to, since the relationship is
// strictly 1:1 for this PRD.
type Application struct {
	ID           string `json:"id"`
	JobListingID string `json:"jobListingId"`
	Status       Status `json:"status"`
}
