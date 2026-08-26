package generation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

// ErrInvalidSelection marks a Client response that Generate rejects because
// it can't be trusted: it names an Entry/bullet absent from Master Data, or
// misreports a bullet's source text. Either would let fabricated content
// slip past Text Review, which the no-invented-facts constraint (see
// CONTEXT.md's Rewrite entry, ADR-0002) does not allow.
var ErrInvalidSelection = errors.New("client returned a selection not traceable to Master Data")

// Generate runs Selection+Rewrite for one Generation: pasted/fetched Job
// Description text drives Tailoring via client; no Job Description at all
// runs Default Mode, which never calls client (see CONTEXT.md's Default
// Mode entry).
func Generate(ctx context.Context, dataDir string, client Client, req GenerateRequest) (GenerateResult, error) {
	entries, err := masterdata.ListEntries(dataDir)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("loading master data: %w", err)
	}

	jobDescription, err := resolveJobDescription(ctx, req)
	if err != nil {
		return GenerateResult{}, err
	}

	if jobDescription == "" {
		return GenerateResult{Mode: ModeDefault, Selection: defaultModeSelection(entries)}, nil
	}

	selection, err := client.SelectAndRewrite(ctx, SelectionRequest{
		JobDescription: jobDescription,
		Candidates:     toCandidates(entries),
	})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("selecting and rewriting: %w", err)
	}

	if err := validateSelection(selection, entries); err != nil {
		return GenerateResult{}, err
	}

	return GenerateResult{Mode: ModeTailored, JobDescription: jobDescription, Selection: selection}, nil
}

// resolveJobDescription returns the Job Description text to Tailor against:
// req.JobDescription verbatim if set, the text extracted from
// req.JobDescriptionURL if that's set instead, or "" (Default Mode) if
// neither is.
func resolveJobDescription(ctx context.Context, req GenerateRequest) (string, error) {
	if req.JobDescription != "" {
		return req.JobDescription, nil
	}
	if req.JobDescriptionURL == "" {
		return "", nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.JobDescriptionURL, nil)
	if err != nil {
		return "", fmt.Errorf("building request for job description URL: %w", err)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("fetching job description URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching job description URL: got status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading job description URL body: %w", err)
	}
	return htmlToText(string(body)), nil
}

var (
	scriptOrStyleRe = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	tagRe           = regexp.MustCompile(`(?s)<[^>]+>`)
	whitespaceRe    = regexp.MustCompile(`[ \t]+`)
	blankLinesRe    = regexp.MustCompile(`\n{3,}`)
)

// htmlToText strips an HTML document down to its visible text. Good enough
// for extracting a Job Description from a job-posting page, not a general
// HTML renderer.
func htmlToText(html string) string {
	text := scriptOrStyleRe.ReplaceAllString(html, "\n")
	text = tagRe.ReplaceAllString(text, "\n")
	text = whitespaceRe.ReplaceAllString(text, " ")
	text = blankLinesRe.ReplaceAllString(text, "\n\n")
	lines := strings.Split(text, "\n")
	var kept []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			kept = append(kept, t)
		}
	}
	return strings.Join(kept, "\n")
}

func toCandidates(entries []masterdata.Entry) []CandidateEntry {
	candidates := make([]CandidateEntry, len(entries))
	for i, e := range entries {
		candidates[i] = CandidateEntry{
			ID:       e.ID,
			Type:     e.Type,
			Employer: e.Employer,
			Client:   e.Client,
			Role:     e.Role,
			Name:     e.Name,
			Start:    e.Start,
			Flagship: e.Flagship,
			Tags:     e.Tags,
			Bullets:  e.Bullets,
		}
	}
	return candidates
}

// defaultModeSelection includes every Entry, unmodified (Rewrite is
// skipped, per CONTEXT.md's Default Mode entry), Flagship Entries first
// then by most recent Start date, so the FE and Render see the most
// representative work first without the user needing to reorder it.
func defaultModeSelection(entries []masterdata.Entry) SelectionResult {
	sorted := make([]masterdata.Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Flagship != sorted[j].Flagship {
			return sorted[i].Flagship
		}
		return sorted[i].Start > sorted[j].Start
	})

	result := SelectionResult{Entries: make([]SelectedEntry, len(sorted))}
	for i, e := range sorted {
		bullets := make([]SelectedBullet, len(e.Bullets))
		for j, b := range e.Bullets {
			bullets[j] = SelectedBullet{SourceIndex: j, Source: b, Rewritten: b}
		}
		result.Entries[i] = SelectedEntry{EntryID: e.ID, Reason: "Default Mode: included in full.", Bullets: bullets}
	}
	return result
}

// validateSelection rejects a Client response that isn't traceable to
// Master Data: an unknown EntryID, an out-of-range SourceIndex, or a
// reported Source that doesn't match what's actually on disk for that
// bullet. Rewritten text is exempt — Rewrite is allowed to reword it.
func validateSelection(selection SelectionResult, entries []masterdata.Entry) error {
	byID := make(map[string]masterdata.Entry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}

	for _, se := range selection.Entries {
		entry, ok := byID[se.EntryID]
		if !ok {
			return fmt.Errorf("%w: unknown entry id %q", ErrInvalidSelection, se.EntryID)
		}
		for _, b := range se.Bullets {
			if b.SourceIndex < 0 || b.SourceIndex >= len(entry.Bullets) {
				return fmt.Errorf("%w: entry %q has no bullet at index %d", ErrInvalidSelection, se.EntryID, b.SourceIndex)
			}
			if entry.Bullets[b.SourceIndex] != b.Source {
				return fmt.Errorf("%w: entry %q bullet %d source text does not match master data", ErrInvalidSelection, se.EntryID, b.SourceIndex)
			}
		}
	}
	return nil
}
