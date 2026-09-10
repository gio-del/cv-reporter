package generation

import (
	"fmt"
	"strings"
)

// ParsabilityStatus is the outcome of an ATS-parsability check (PRD "PDF
// ATS-parsability check") against a rendered PDF's extracted text layer.
type ParsabilityStatus string

const (
	// ParsabilityOK means every expected field was found in the extracted
	// text, in the expected relative order.
	ParsabilityOK ParsabilityStatus = "ok"
	// ParsabilityWarning means at least one expected field was missing or
	// out of order. Never blocks Render or Visual Review (PRD story 7) —
	// it's a signal surfaced alongside Visual Review, not a gate.
	ParsabilityWarning ParsabilityStatus = "warning"
	// ParsabilityUnavailable means the check itself couldn't run (e.g.
	// pdftotext missing or erroring) — distinct from ParsabilityWarning,
	// since a tooling failure says nothing about the PDF's actual
	// parsability (PRD story 10).
	ParsabilityUnavailable ParsabilityStatus = "unavailable"
)

// ParsabilityResult is the ATS-parsability check's structured outcome
// (PRD story 6): which expected fields never showed up in the PDF's
// extracted text layer, and which showed up out of the order the template
// renders them in.
type ParsabilityResult struct {
	Status             ParsabilityStatus `json:"status"`
	MissingFields      []string          `json:"missingFields,omitempty"`
	OrderingViolations []string          `json:"orderingViolations,omitempty"`
	// Reason explains an Unavailable Status (PRD story 10) — empty
	// otherwise.
	Reason string `json:"reason,omitempty"`
}

// expectedField is one piece of source data (cvData or coverLetterData)
// the rendered PDF's text layer is expected to contain, in the order the
// template should render it in.
type expectedField struct {
	// label identifies the field in MissingFields/OrderingViolations,
	// e.g. "Experience employer \"Acme Corp\"".
	label string
	// text is the literal substring looked up in the extracted text.
	text string
}

// checkParsability is the PRD's pure comparison function: given a PDF's
// extracted text (from pdftotext) and the ordered list of fields the
// template should have rendered from the source data, it reports which
// fields are missing and which appear out of their expected relative
// order. It does no I/O and no PDF parsing of its own — extractPDFText
// handles that — which is what makes it directly unit-testable against
// fixture strings (PRD's Testing Decisions).
//
// Ordering is checked by tracking the furthest-right index seen so far
// among present fields: a field whose index falls before that point has
// been read out of order relative to something that should have preceded
// it. A field skipped as missing doesn't participate in ordering at all —
// there's no index to compare.
func checkParsability(extractedText string, fields []expectedField) ParsabilityResult {
	var missing []string
	var violations []string

	runningMax := -1
	lastInOrderLabel := ""
	for _, f := range fields {
		idx := strings.Index(extractedText, f.text)
		if idx == -1 {
			missing = append(missing, f.label)
			continue
		}
		if idx < runningMax {
			violations = append(violations, fmt.Sprintf("%s appears before %s in the extracted text", f.label, lastInOrderLabel))
			continue
		}
		runningMax = idx
		lastInOrderLabel = f.label
	}

	status := ParsabilityOK
	if len(missing) > 0 || len(violations) > 0 {
		status = ParsabilityWarning
	}
	return ParsabilityResult{Status: status, MissingFields: missing, OrderingViolations: violations}
}

// cvExpectedFields builds the ordered list of fields cv.typ is expected to
// render from cv, per the PRD's Implementation Decisions: the name, each
// section's expected header text (only for sections cv.typ actually
// renders — Education and Experience unconditionally, the rest only when
// non-empty), and each Experience employer / Project name.
func cvExpectedFields(cv cvData) []expectedField {
	fields := []expectedField{
		{label: "Name", text: cv.Name},
		{label: "Education section header", text: "Education"},
		{label: "Experience section header", text: "Experience"},
	}

	seenEmployer := map[string]bool{}
	for _, exp := range cv.Experience {
		if exp.Employer == "" || seenEmployer[exp.Employer] {
			continue
		}
		seenEmployer[exp.Employer] = true
		fields = append(fields, expectedField{
			label: fmt.Sprintf("Experience employer %q", exp.Employer),
			text:  exp.Employer,
		})
	}

	if len(cv.Projects) > 0 {
		fields = append(fields, expectedField{label: "Projects section header", text: "Projects"})
		for _, p := range cv.Projects {
			fields = append(fields, expectedField{label: fmt.Sprintf("Project %q", p.Name), text: p.Name})
		}
	}

	if len(cv.TechStack) > 0 {
		fields = append(fields, expectedField{label: "Tech Stack section header", text: "Tech Stack"})
	}
	if len(cv.Publications) > 0 {
		fields = append(fields, expectedField{label: "Publications section header", text: "Publications"})
	}
	if len(cv.Awards) > 0 {
		fields = append(fields, expectedField{label: "Awards section header", text: "Awards"})
	}
	if len(cv.Activities) > 0 {
		fields = append(fields, expectedField{label: "Activities section header", text: "Activities"})
	}
	if len(cv.Languages) > 0 {
		fields = append(fields, expectedField{label: "Languages section header", text: "Languages"})
	}

	return fields
}

// coverLetterExpectedFields builds the ordered list of fields
// cover-letter.typ is expected to render from cl: the name (in the
// contact-line header) followed by the body's opening line (cover-letter.typ
// renders the full body verbatim, but a whole-body substring match would be
// fragile against pdftotext's line-wrapping, so the first non-empty line is
// used as a representative anchor).
func coverLetterExpectedFields(cl coverLetterData) []expectedField {
	fields := []expectedField{
		{label: "Name", text: cl.Name},
	}
	if firstLine := firstNonEmptyLine(cl.Body); firstLine != "" {
		fields = append(fields, expectedField{label: "Body opening", text: firstLine})
	}
	return fields
}

// firstNonEmptyLine returns s's first line with non-whitespace content,
// trimmed, or "" if s has none.
func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
