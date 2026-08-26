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

// ApplicationMethodKind is how a Job Listing says to apply, per CONTEXT.md's
// Application Method entry.
type ApplicationMethodKind string

const (
	MethodPortal    ApplicationMethodKind = "portal"
	MethodEmail     ApplicationMethodKind = "email"
	MethodEasyApply ApplicationMethodKind = "easy_apply"
	MethodOther     ApplicationMethodKind = "other"
)

// ApplicationMethod is inferred by Claude from a Job Listing's Job
// Description at save time (story 5) and user-editable afterward (story
// 6). Value is the detected application URL or email address, where
// applicable (empty for Kind other, or when nothing was detected).
type ApplicationMethod struct {
	Kind  ApplicationMethodKind `json:"kind"`
	Value string                `json:"value,omitempty"`
}

// GenerationRecord is one Generation run recorded against an Application
// (story 11), after the FE has rendered it via POST
// /api/generations/render. Kept in a history rather than overwritten on
// regenerate (stories 12, 13) — the most recent is what the user would
// actually send.
type GenerationRecord struct {
	Slug            string `json:"slug"`
	CreatedAt       string `json:"createdAt"`
	CVPath          string `json:"cvPath"`
	CoverLetterPath string `json:"coverLetterPath,omitempty"`
}

// Application is the tracked record of one attempt to apply to a Job
// Listing — exactly one per Job Listing, created at Status Saved the
// moment it's saved (CONTEXT.md's Application entry, story 2). It shares
// its id with the Job Listing it belongs to, since the relationship is
// strictly 1:1 for this PRD.
type Application struct {
	ID           string             `json:"id"`
	JobListingID string             `json:"jobListingId"`
	Status       Status             `json:"status"`
	Method       ApplicationMethod  `json:"method"`
	Generations  []GenerationRecord `json:"generations,omitempty"`
}

// ListingWithApplication pairs a Job Listing with its 1:1 Application, the
// shape the pipeline list view needs (story 3): Status and RAL Range
// visible without opening each record (story 14).
type ListingWithApplication struct {
	JobListing  JobListing  `json:"jobListing"`
	Application Application `json:"application"`
}
