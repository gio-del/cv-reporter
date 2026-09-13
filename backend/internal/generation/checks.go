package generation

import (
	"fmt"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

// This file holds the exported facades over the Generation pipeline's
// automated quality checks (groundedness, ATS-parsability, page count,
// language), one per check point. They exist so a second caller — the
// cvcheck CLI under backend/cmd/cvcheck, which the tailor-cv skill shells
// out to (issue #104, ADR-0028) — runs the exact same check code the web
// app's Generate/Render path runs, without exporting the scoring internals
// (checkGroundedness, computeGroundedness, checkPDFParsability,
// cvExpectedFields, countPDFPages) themselves.

// CheckSelectionGroundedness is the Text Review check point's facade: it
// verifies selection is traceable to the Master Data under dataDir (the
// same rule Generate enforces via validateSelection — each bullet's Source
// must be the verbatim Master Data bullet at SourceIndex, otherwise the
// overlap score would be measured against invented "source" text), then
// runs the groundedness check over every rewritten bullet against its own
// source bullet.
//
// Bullets only: a skill-side Selection+Rewrite result has no Cover Letter,
// so GroundednessResult.CoverLetter is always empty here. An empty result
// means the check ran and flagged nothing.
func CheckSelectionGroundedness(dataDir string, selection SelectionResult) (GroundednessResult, error) {
	if info, err := os.Stat(dataDir); err != nil {
		return GroundednessResult{}, fmt.Errorf("reading master data directory: %w", err)
	} else if !info.IsDir() {
		return GroundednessResult{}, fmt.Errorf("master data directory %s is not a directory", dataDir)
	}

	entries, err := masterdata.ListEntries(dataDir)
	if err != nil {
		return GroundednessResult{}, fmt.Errorf("loading master data: %w", err)
	}
	if err := validateSelection(selection, entries); err != nil {
		return GroundednessResult{}, err
	}
	return computeGroundedness(selection, CoverLetterResult{}, nil), nil
}
