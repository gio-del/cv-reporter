package generation

import "testing"

func TestCheckGroundedness_LightlyRewordedSentence_NoFlag(t *testing.T) {
	sources := []string{
		"Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
	}
	generated := "Migrated the legacy billing service onto a microservices architecture, which reduced incident response time by 60 percent."

	flags := checkGroundedness(sources, generated)
	if len(flags) != 0 {
		t.Errorf("expected no flags for a lightly reworded sentence, got %+v", flags)
	}
}

func TestCheckGroundedness_HeavyParaphraseSameFacts_NoFlag(t *testing.T) {
	sources := []string{
		"Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
	}
	generated := "Moved the aging billing service onto a microservices architecture, cutting incident response time 60 percent."

	flags := checkGroundedness(sources, generated)
	if len(flags) != 0 {
		t.Errorf("expected no flags for a heavy paraphrase that preserves the underlying facts, got %+v", flags)
	}
}

func TestCheckGroundedness_NumberChangedFromSource_FlaggedNumericMismatch(t *testing.T) {
	sources := []string{
		"Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
	}
	generated := "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 90 percent."

	flags := checkGroundedness(sources, generated)
	if len(flags) != 1 {
		t.Fatalf("expected exactly one flag for a fabricated number, got %+v", flags)
	}
	if flags[0].Reason != ReasonNumericMismatch {
		t.Errorf("expected reason %q, got %q", ReasonNumericMismatch, flags[0].Reason)
	}
}

func TestCheckGroundedness_NoPlausibleSourceMatch_FlaggedNoSourceMatch(t *testing.T) {
	sources := []string{
		"Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
	}
	generated := "Presented the quarterly roadmap to the executive leadership team in Q1."

	flags := checkGroundedness(sources, generated)
	if len(flags) != 1 {
		t.Fatalf("expected exactly one flag for an unrelated sentence, got %+v", flags)
	}
	if flags[0].Reason != ReasonNoSourceMatch {
		t.Errorf("expected reason %q, got %q", ReasonNoSourceMatch, flags[0].Reason)
	}
}

func TestCheckGroundedness_MultipleSentences_FlagsOnlyTheDriftingOne(t *testing.T) {
	sources := []string{
		"Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
	}
	generated := "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent. " +
		"Presented the quarterly roadmap to the executive leadership team in Q1."

	flags := checkGroundedness(sources, generated)
	if len(flags) != 1 {
		t.Fatalf("expected exactly one flagged sentence, got %+v", flags)
	}
	if flags[0].Sentence != "Presented the quarterly roadmap to the executive leadership team in Q1." {
		t.Errorf("expected the unrelated sentence to be the one flagged, got %q", flags[0].Sentence)
	}
}

func TestCheckGroundedness_EmptyGenerated_NoFlag(t *testing.T) {
	flags := checkGroundedness([]string{"Some source text."}, "")
	if len(flags) != 0 {
		t.Errorf("expected no flags for empty generated text, got %+v", flags)
	}
}

func TestCheckGroundedness_NoSources_NoFlag(t *testing.T) {
	// Degrades gracefully rather than flagging everything when it can't
	// reach a verdict (PRD story 8) — an empty source list means there's
	// nothing to compare against, not that the sentence is unsupported.
	flags := checkGroundedness(nil, "Migrated the legacy billing service to a microservices architecture.")
	if len(flags) != 0 {
		t.Errorf("expected no flags when there are no sources to compare against, got %+v", flags)
	}
}

func TestComputeGroundedness_FlagsOnlyDriftingBulletAndCoverLetterSentence(t *testing.T) {
	selection := SelectionResult{
		Entries: []SelectedEntry{
			{
				EntryID: "acme-eng",
				Bullets: []SelectedBullet{
					{
						SourceIndex: 0,
						Source:      "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 60 percent.",
						Rewritten:   "Migrated the legacy billing service to a microservices architecture, reducing incident response time by 90 percent.",
					},
					{
						SourceIndex: 1,
						Source:      "Led a cross-functional team of 5 engineers.",
						Rewritten:   "Led a cross-functional team of 5 engineers.",
					},
				},
			},
		},
	}
	coverLetter := CoverLetterResult{
		Body: "Presented the quarterly roadmap to the executive leadership team in Q1.",
	}

	result := computeGroundedness(selection, coverLetter, nil)

	if len(result.Bullets) != 1 {
		t.Fatalf("expected exactly one flagged bullet, got %+v", result.Bullets)
	}
	if result.Bullets[0].EntryID != "acme-eng" || result.Bullets[0].SourceIndex != 0 {
		t.Errorf("expected the flagged bullet to be acme-eng/0, got %+v", result.Bullets[0])
	}
	if len(result.CoverLetter) != 1 {
		t.Errorf("expected exactly one flagged cover letter sentence, got %+v", result.CoverLetter)
	}
}
