package generation

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const checksFixtureEntry = `---
employer: Acme Corp
role: Senior Engineer
start: "2022"
end: null
tags:
  - Go
---

- Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.
- Led a cross-functional team of 5 engineers.
`

// writeChecksDataDir writes a minimal Master Data tree (one experience
// Entry, id "experience/acme") under a temp dir and returns its path.
func writeChecksDataDir(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "experience"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "experience", "acme.md"), []byte(checksFixtureEntry), 0o644); err != nil {
		t.Fatal(err)
	}
	return dataDir
}

func TestCheckSelectionGroundedness_FabricatedNumber_FlagsOnlyThatBullet(t *testing.T) {
	dataDir := writeChecksDataDir(t)
	selection := SelectionResult{
		Entries: []SelectedEntry{{
			EntryID: "experience/acme",
			Bullets: []SelectedBullet{
				{
					SourceIndex: 0,
					Source:      "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
					Rewritten:   "Moved the aging billing service onto microservices, cutting incident response time by 90 percent.",
				},
				{
					SourceIndex: 1,
					Source:      "Led a cross-functional team of 5 engineers.",
					Rewritten:   "Led a cross-functional team of five engineers across 5 disciplines.",
				},
			},
		}},
	}

	result, err := CheckSelectionGroundedness(dataDir, selection)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Bullets) != 1 {
		t.Fatalf("expected exactly one flagged bullet, got %+v", result.Bullets)
	}
	got := result.Bullets[0]
	if got.EntryID != "experience/acme" || got.SourceIndex != 0 {
		t.Errorf("expected experience/acme bullet 0 flagged, got %+v", got)
	}
	if len(got.Flags) != 1 || got.Flags[0].Reason != ReasonNumericMismatch {
		t.Errorf("expected one numeric-mismatch flag, got %+v", got.Flags)
	}
	if result.CoverLetter != nil {
		t.Errorf("expected no cover letter flags from a bullets-only check, got %+v", result.CoverLetter)
	}
}

func TestCheckSelectionGroundedness_HeavyParaphrase_NoFlags(t *testing.T) {
	dataDir := writeChecksDataDir(t)
	selection := SelectionResult{
		Entries: []SelectedEntry{{
			EntryID: "experience/acme",
			Bullets: []SelectedBullet{{
				SourceIndex: 0,
				Source:      "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
				Rewritten:   "Moved the aging billing service onto a microservices architecture, cutting incident response time 60 percent.",
			}},
		}},
	}

	result, err := CheckSelectionGroundedness(dataDir, selection)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Bullets) != 0 || len(result.CoverLetter) != 0 {
		t.Errorf("expected an empty result for a grounded paraphrase, got %+v", result)
	}
}

func TestCheckSelectionGroundedness_SourceNotVerbatimFromMasterData_ReturnsInvalidSelection(t *testing.T) {
	// A source bullet that was itself edited would make the overlap check
	// meaningless (the rewrite would be scored against invented "source"
	// text), so the facade applies the same traceability rule Generate does.
	dataDir := writeChecksDataDir(t)
	selection := SelectionResult{
		Entries: []SelectedEntry{{
			EntryID: "experience/acme",
			Bullets: []SelectedBullet{{
				SourceIndex: 1,
				Source:      "Led a cross-functional team of 12 engineers.",
				Rewritten:   "Led a cross-functional team of 12 engineers.",
			}},
		}},
	}

	_, err := CheckSelectionGroundedness(dataDir, selection)
	if !errors.Is(err, ErrInvalidSelection) {
		t.Fatalf("expected ErrInvalidSelection, got %v", err)
	}
}

func TestCheckSelectionGroundedness_MissingDataDir_ReturnsError(t *testing.T) {
	_, err := CheckSelectionGroundedness(filepath.Join(t.TempDir(), "nope"), SelectionResult{})
	if err == nil {
		t.Fatal("expected an error for a missing Master Data directory")
	}
}
