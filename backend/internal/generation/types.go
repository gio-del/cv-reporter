// Package generation implements the Generation pipeline (Selection,
// Rewrite, Cover Letter drafting, RAL Range lookup, Render) described in
// CONTEXT.md and ADR-0005: the app calls the Claude API directly rather
// than delegating to the tailor-cv skill.
package generation

// CandidateEntry is the subset of a Master Data Entry that Selection
// reasons over: identity plus its bullets. Keeping this separate from
// masterdata.Entry stops the full Entry shape (including fields Selection
// never needs) from leaking into prompts and Client responses.
type CandidateEntry struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Employer string   `json:"employer,omitempty"`
	Client   string   `json:"client,omitempty"`
	Role     string   `json:"role,omitempty"`
	Name     string   `json:"name,omitempty"`
	Start    string   `json:"start"`
	Flagship bool     `json:"flagship,omitempty"`
	Tags     []string `json:"tags"`
	Bullets  []string `json:"bullets"`
}

// SelectionRequest is what the generation service asks a Client to choose
// and rewrite Entries from.
type SelectionRequest struct {
	JobDescription string
	Candidates     []CandidateEntry
}

// SelectedBullet is one bullet Selection chose for a Generation. Source is
// kept verbatim from Master Data (so Text Review can diff Rewritten against
// it, per story 4) alongside SourceIndex, the bullet's position in the
// source Entry's Bullets slice.
type SelectedBullet struct {
	SourceIndex int    `json:"sourceIndex"`
	Source      string `json:"source"`
	Rewritten   string `json:"rewritten"`
}

// SelectedEntry is one Entry Selection chose, with the bullets it kept (in
// the order they should render) and why it was chosen.
type SelectedEntry struct {
	EntryID string           `json:"entryId"`
	Reason  string           `json:"reason"`
	Bullets []SelectedBullet `json:"bullets"`
}

// SelectionResult is a Client's Selection+Rewrite output for one Generation.
type SelectionResult struct {
	Entries []SelectedEntry `json:"entries"`
}

// CandidateSnippet is a Cover Letter Snippet made available to the Client
// for drafting a Cover Letter, per CONTEXT.md's Cover Letter Snippet entry.
type CandidateSnippet struct {
	ID   string   `json:"id"`
	Kind string   `json:"kind"`
	Tags []string `json:"tags"`
	Body string   `json:"body"`
}

// CoverLetterRequest is what the generation service asks a Client to draft
// a Cover Letter from: the Job Description plus grounding material (Master
// Data Entries and any Cover Letter Snippets).
type CoverLetterRequest struct {
	JobDescription string
	Candidates     []CandidateEntry
	Snippets       []CandidateSnippet
}

// CoverLetterResult is a Client's Cover Letter draft. SourceSnippetIDs
// names which Cover Letter Snippets (if any) it selected/adapted from — nil
// or empty means it was freshly generated prose, per CONTEXT.md's Cover
// Letter entry.
type CoverLetterResult struct {
	Body             string   `json:"body"`
	SourceSnippetIDs []string `json:"sourceSnippetIds,omitempty"`
}

// GenerateRequest is the input to Generate: a pasted Job Description, a URL
// to fetch one from, or neither (Default Mode).
type GenerateRequest struct {
	JobDescription    string `json:"jobDescription"`
	JobDescriptionURL string `json:"jobDescriptionUrl"`
}

// GenerateMode distinguishes a Tailoring run from Default Mode (see
// CONTEXT.md).
type GenerateMode string

const (
	ModeDefault  GenerateMode = "default"
	ModeTailored GenerateMode = "tailored"
)

// GenerateResult is the Text-Review-ready output of a Generation's
// Selection+Rewrite (and, in Tailored Mode, Cover Letter drafting) step.
// CoverLetter is nil in Default Mode: there's no Job Description to ground
// fresh prose in, mirroring Rewrite being skipped (see CONTEXT.md's Default
// Mode entry).
type GenerateResult struct {
	Mode           GenerateMode       `json:"mode"`
	JobDescription string             `json:"jobDescription,omitempty"`
	Selection      SelectionResult    `json:"selection"`
	CoverLetter    *CoverLetterResult `json:"coverLetter,omitempty"`
}
