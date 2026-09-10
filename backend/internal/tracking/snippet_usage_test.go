package tracking

import "testing"

func TestSnippetLastUsed_PicksMostRecentAcrossApplications(t *testing.T) {
	apps := []Application{
		{
			ID: "app-1",
			Generations: []GenerationRecord{
				{Slug: "a", CreatedAt: "2026-01-01T00:00:00Z", SourceSnippetIDs: []string{"opener-1"}},
				{Slug: "a", CreatedAt: "2026-03-01T00:00:00Z", SourceSnippetIDs: []string{"opener-1", "closer-2"}},
			},
		},
		{
			ID: "app-2",
			Generations: []GenerationRecord{
				{Slug: "b", CreatedAt: "2026-02-01T00:00:00Z", SourceSnippetIDs: []string{"opener-1"}},
			},
		},
	}

	got := SnippetLastUsed(apps)

	if got["opener-1"] != "2026-03-01T00:00:00Z" {
		t.Errorf("expected opener-1's latest use to be the app-1 regeneration (2026-03-01), got %q", got["opener-1"])
	}
	if got["closer-2"] != "2026-03-01T00:00:00Z" {
		t.Errorf("expected closer-2's only use, got %q", got["closer-2"])
	}
	if _, ok := got["never-used"]; ok {
		t.Errorf("expected no entry for a Snippet id that never appears, got %v", got["never-used"])
	}
}

func TestSnippetLastUsed_IgnoresRecordsWithNoSnippetIDs(t *testing.T) {
	apps := []Application{
		{
			ID: "app-1",
			Generations: []GenerationRecord{
				{Slug: "a", CreatedAt: "2026-01-01T00:00:00Z"},
			},
		},
	}

	got := SnippetLastUsed(apps)

	if len(got) != 0 {
		t.Errorf("expected no usage signal from a record with no SourceSnippetIDs, got %v", got)
	}
}

func TestSnippetLastUsed_EmptyApplications(t *testing.T) {
	got := SnippetLastUsed(nil)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}
