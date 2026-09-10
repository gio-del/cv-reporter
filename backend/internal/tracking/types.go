// Package tracking implements Job Listing and Application storage and
// lifecycle — the persisted, trackable records CONTEXT.md's Job Listing and
// Application entries describe, stored as flat files alongside Master Data
// (ADR-0008).
package tracking

import (
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// SourceManual is the only Source this PRD can produce — ATS feeds and the
// browser extension (CONTEXT.md's other two Job Listing sources) are out of
// scope here (see ADR-0007).
const SourceManual = "manual"

// JobListing is a persisted, tracked record of a role the user is
// considering, per CONTEXT.md's Job Listing entry.
type JobListing struct {
	ID             string              `json:"id"`
	Title          string              `json:"title,omitempty"`
	Company        string              `json:"company"`
	URL            string              `json:"url,omitempty"`
	Source         string              `json:"source"`
	SavedAt        string              `json:"savedAt"`
	JobDescription string              `json:"jobDescription"`
	RAL            generation.RALRange `json:"ral"`
	// Logo is the filename (relative to the jobs dir) of the Company Logo
	// downloaded server-side at save time (ADR-0013), empty when none was
	// captured or the download failed.
	Logo string `json:"logo,omitempty"`
	// FreshnessStatus is the URL's most recent on-demand liveness check
	// result (issue #59), defaulting to FreshnessNotYetChecked until a
	// check is run via CheckFreshness. Kept independent of Logo/ADR-0013 —
	// a broken freshness check can never affect the stored logo file.
	FreshnessStatus FreshnessStatus `json:"freshnessStatus"`
	// FreshnessCheckedAt is when FreshnessStatus was last updated (UTC,
	// RFC3339Nano), empty until the first completed check.
	FreshnessCheckedAt string `json:"freshnessCheckedAt,omitempty"`
}

// FreshnessStatus is a Job Listing source URL's most recent on-demand
// liveness check result (issue #59, "Job Description link-rot / staleness
// check"). Unlike freshness.Status (the pure classification result), this
// type also carries FreshnessNotYetChecked — a persisted-record concept
// with no equivalent in a single check's outcome.
type FreshnessStatus string

const (
	// FreshnessNotYetChecked is the default for every Job Listing until a
	// freshness check is run (story 9) — distinct from FreshnessUnknown so
	// "never checked" is never confused with "checked and inconclusive".
	FreshnessNotYetChecked FreshnessStatus = "not-yet-checked"
	FreshnessLive          FreshnessStatus = "live"
	FreshnessUnreachable   FreshnessStatus = "unreachable"
	FreshnessUnknown       FreshnessStatus = "unknown"
)

// Status is where an Application stands, per CONTEXT.md's Status entry.
type Status string

const (
	StatusSaved        Status = "saved"
	StatusTailoring    Status = "tailoring"
	StatusSent         Status = "sent"
	StatusInterviewing Status = "interviewing"
	StatusRejected     Status = "rejected"
	StatusOffer        Status = "offer"
	// StatusWithdrawn records the user ending the process on their own
	// initiative — distinct from StatusRejected, which means the employer
	// ended it. Terminal-but-reopenable back to StatusInterviewing, mirroring
	// StatusRejected's Reopen precedent exactly (see transition.go).
	StatusWithdrawn Status = "withdrawn"
)

// ApplicationMethodKind is how a Job Listing says to apply, per CONTEXT.md's
// Application Method entry.
type ApplicationMethodKind string

const (
	MethodPortal    ApplicationMethodKind = "portal"
	MethodEmail     ApplicationMethodKind = "email"
	MethodEasyApply ApplicationMethodKind = "easy_apply"
	MethodOther     ApplicationMethodKind = "other"
	// MethodUnresolved is a system-set sentinel meaning Application Method
	// inference couldn't even be attempted (client.InferApplicationMethod
	// returned an error) — not a real method the user can pick by hand (see
	// knownMethodKinds in method.go, which intentionally excludes it).
	MethodUnresolved ApplicationMethodKind = "unresolved"
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
	Slug            string                         `json:"slug"`
	CreatedAt       string                         `json:"createdAt"`
	CVPath          string                         `json:"cvPath"`
	CoverLetterPath string                         `json:"coverLetterPath,omitempty"`
	Groundedness    *generation.GroundednessResult `json:"groundedness,omitempty" yaml:"groundedness,omitempty"`

	// SourceSnippetIDs are the Cover Letter Snippet ids this Generation's
	// Cover Letter drew from, as returned by POST /api/generations at
	// generation time. Empty/absent means either no Snippet was used (fresh
	// prose) or this record predates the field — the two are indistinguishable,
	// and both must be treated as "no usage signal from this record" rather
	// than "never used" (issue #48).
	SourceSnippetIDs []string `json:"sourceSnippetIds,omitempty"`

	// Usage is the Claude API usage/cost the Generate call that produced this
	// Generation caused, as returned in GenerateResult.Usage — passed through
	// verbatim by the FE at record time (issue #39: cost/usage visibility
	// persisted per-Generation, not just shown transiently at Generate time).
	// Zero-value (omitted) for a Default Mode Generation, or one recorded
	// before this field existed.
	Usage generation.GenerationUsage `json:"usage,omitempty"`

	// Language is the final, normalized target language the CV/Cover
	// Letter were written in (generation.GenerateResult.Language) — kept
	// on the record so the Application's Generation history shows it
	// without reopening the PDF (issue #41's PRD, story 9).
	Language string `json:"language,omitempty"`

	// EntryIDs are the Master Data Entry ids Selection chose for this
	// Generation (issue #52), populated at record time from the Selection
	// result the caller already has (RecordGeneration adds no new
	// Selection logic of its own). Nil for GenerationRecords persisted
	// before this field existed — the durable source of truth a later
	// staleness check compares against, since the assembled per-Generation
	// JSON under output/ is gitignored and not guaranteed to still exist
	// (ADR-0008).
	EntryIDs []string `json:"entryIds,omitempty"`
	// StaleEntries names (by employer/client + role) which of EntryIDs have
	// been modified since CreatedAt, per masterdata.EntryLastModified
	// (issue #52). Computed read-time by the API layer for the Application
	// detail response — never persisted (yaml:"-"), since it's a
	// point-in-time mechanical comparison, not durable state.
	StaleEntries []string `json:"staleEntries,omitempty" yaml:"-"`
}

// Contact is the recruiter/hiring-manager name and email for an
// email-method Application, per CONTEXT.md's Contact entry — entered
// manually or Claude-suggested via web search, but never persisted without
// explicit user confirmation (story 7): SuggestContact never writes to
// disk, only UpdateApplicationContact does.
type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// StatusChange is one entry in an Application's StatusHistory: the Status
// it moved to, and when. Appended on every successful Transition
// (including Reopen), so aggregate views like the Application funnel/stats
// page (issue #36) have a real data trail to compute time-in-stage from,
// instead of only ever seeing the current Status.
type StatusChange struct {
	Status    Status    `json:"status"`
	ChangedAt time.Time `json:"changedAt"`
}

// Application is the tracked record of one attempt to apply to a Job
// Listing — exactly one per Job Listing, created at Status Saved the
// moment it's saved (CONTEXT.md's Application entry, story 2). It shares
// its id with the Job Listing it belongs to, since the relationship is
// strictly 1:1 for this PRD.
type Application struct {
	ID           string `json:"id"`
	JobListingID string `json:"jobListingId"`
	Status       Status `json:"status"`
	// StatusUpdatedAt (RFC3339Nano) is when Status last changed — set at
	// Application creation and on every successful Transition, including
	// Reopen (story 1-3, 8). Empty for records written before this field
	// existed; IsStale treats that as not stale rather than erroring.
	StatusUpdatedAt string            `json:"statusUpdatedAt,omitempty"`
	Method          ApplicationMethod `json:"method"`
	Contact         *Contact          `json:"contact,omitempty"`
	// IsStale is computed at read time (never persisted) from Status,
	// StatusUpdatedAt and DefaultStaleThreshold (story 4, 10-11) — see
	// IsStale in staleness.go.
	IsStale     bool               `json:"isStale"`
	Generations []GenerationRecord `json:"generations,omitempty"`
	// StatusHistory is append-only: one entry per Status the Application has
	// ever moved to (including its initial Saved entry at creation), oldest
	// first. Applications saved before this field existed simply have an
	// empty slice — there is nothing to backfill it from.
	StatusHistory []StatusChange `json:"statusHistory,omitempty"`
}

// ListingWithApplication pairs a Job Listing with its 1:1 Application, the
// shape the pipeline list view needs (story 3): Status and RAL Range
// visible without opening each record (story 14).
type ListingWithApplication struct {
	JobListing  JobListing  `json:"jobListing"`
	Application Application `json:"application"`
}
