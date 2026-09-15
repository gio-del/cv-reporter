package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gio-del/sumisura/backend/internal/masterdata"
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

// RenderedCVCheck is the Visual Review check point's result for one
// rendered Tailored CV: the mechanical signals Render attaches in the app
// (page count, ATS-parsability), plus the target language the assembled
// data asked the template to render in.
type RenderedCVCheck struct {
	PageCount   int               `json:"pageCount"`
	Parsability ParsabilityResult `json:"parsability"`

	// Language is the assembled data's lang after NormalizeLanguage — what
	// the app's Render would have stamped into the document.
	Language string `json:"language"`
	// LanguageWarning is set when the assembled data's lang is missing or
	// isn't already a supported, normalized code, i.e. when the rendered
	// document's language may not be the one intended.
	LanguageWarning string `json:"languageWarning,omitempty"`
}

// CheckRenderedCV is the Visual Review check point's facade: given the
// rendered CV PDF at pdfPath and the assembled data (the data.json it was
// compiled from, in the shape template/cv.typ reads), it counts pages and
// runs the ATS-parsability check with the same expected-field list Render
// uses. A missing pdftotext degrades to ParsabilityUnavailable rather than
// an error, exactly as in Render; an unreadable PDF or malformed assembled
// data is an error, since no verdict can be reached at all.
func CheckRenderedCV(pdfPath string, assembledData []byte) (RenderedCVCheck, error) {
	var cv cvData
	if err := json.Unmarshal(assembledData, &cv); err != nil {
		return RenderedCVCheck{}, fmt.Errorf("parsing assembled data: %w", err)
	}

	pageCount, err := countPDFPages(pdfPath)
	if err != nil {
		return RenderedCVCheck{}, fmt.Errorf("counting rendered pages: %w", err)
	}

	result := RenderedCVCheck{
		PageCount:   pageCount,
		Parsability: checkPDFParsability(pdfPath, cvExpectedFields(cv)),
		Language:    NormalizeLanguage(cv.Lang),
	}
	switch {
	case strings.TrimSpace(cv.Lang) == "":
		result.LanguageWarning = fmt.Sprintf("assembled data has no lang; the CV was rendered in the default language (%s)", result.Language)
	case cv.Lang != result.Language:
		result.LanguageWarning = fmt.Sprintf("assembled data's lang %q is not a supported language code; the app would render this CV as %q", cv.Lang, result.Language)
	}
	return result, nil
}
