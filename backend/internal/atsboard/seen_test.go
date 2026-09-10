package atsboard_test

import (
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

func TestMarkNewSinceLastCheck_FlagsUnseenURLs(t *testing.T) {
	listings := []atsboard.Listing{
		{Title: "Backend Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"},
		{Title: "Product Designer", URL: "https://boards.greenhouse.io/acme/jobs/2"},
	}
	seenURLs := []string{"https://boards.greenhouse.io/acme/jobs/1"}

	got := atsboard.MarkNewSinceLastCheck(listings, seenURLs)

	if len(got) != 2 {
		t.Fatalf("expected 2 listings, got %d", len(got))
	}
	if got[0].New {
		t.Errorf("expected listing 1 (in seenURLs) to be unmarked as new, got %+v", got[0])
	}
	if !got[1].New {
		t.Errorf("expected listing 2 (not in seenURLs) to be marked new, got %+v", got[1])
	}
	if got[1].Listing != listings[1] {
		t.Errorf("expected the underlying Listing to be preserved, got %+v", got[1].Listing)
	}
}

func TestMarkNewSinceLastCheck_NoSeenURLs_AllNew(t *testing.T) {
	listings := []atsboard.Listing{{Title: "Backend Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"}}

	got := atsboard.MarkNewSinceLastCheck(listings, nil)

	if len(got) != 1 || !got[0].New {
		t.Errorf("expected all listings marked new when nothing has been seen yet, got %+v", got)
	}
}

func TestSeenURLs_NoStateYet_ReturnsEmptyNotExisted(t *testing.T) {
	dataDir := t.TempDir()

	urls, existed, err := atsboard.SeenURLs(dataDir, "greenhouse:acme")
	if err != nil {
		t.Fatalf("SeenURLs: %v", err)
	}
	if existed {
		t.Errorf("expected existed=false for a board with no persisted seen-state, got true")
	}
	if len(urls) != 0 {
		t.Errorf("expected no seen URLs, got %v", urls)
	}
}

func TestRecordSeen_ThenSeenURLs_PersistsUnion(t *testing.T) {
	dataDir := t.TempDir()
	boardID := "greenhouse:acme"

	if err := atsboard.RecordSeen(dataDir, boardID, []string{"a", "b"}); err != nil {
		t.Fatalf("RecordSeen (1st): %v", err)
	}

	urls, existed, err := atsboard.SeenURLs(dataDir, boardID)
	if err != nil {
		t.Fatalf("SeenURLs: %v", err)
	}
	if !existed {
		t.Fatalf("expected existed=true after RecordSeen, got false")
	}
	if !sameSet(urls, []string{"a", "b"}) {
		t.Fatalf("expected seen URLs [a b], got %v", urls)
	}

	if err := atsboard.RecordSeen(dataDir, boardID, []string{"b", "c"}); err != nil {
		t.Fatalf("RecordSeen (2nd): %v", err)
	}

	urls, _, err = atsboard.SeenURLs(dataDir, boardID)
	if err != nil {
		t.Fatalf("SeenURLs: %v", err)
	}
	if !sameSet(urls, []string{"a", "b", "c"}) {
		t.Fatalf("expected the seen set to union rather than replace, got %v", urls)
	}
}

func TestRecordSeen_DistinctBoards_DoNotShareState(t *testing.T) {
	dataDir := t.TempDir()

	if err := atsboard.RecordSeen(dataDir, "greenhouse:acme", []string{"a"}); err != nil {
		t.Fatalf("RecordSeen: %v", err)
	}
	if err := atsboard.RecordSeen(dataDir, "lever:widgetco", []string{"z"}); err != nil {
		t.Fatalf("RecordSeen: %v", err)
	}

	urls, _, err := atsboard.SeenURLs(dataDir, "greenhouse:acme")
	if err != nil {
		t.Fatalf("SeenURLs: %v", err)
	}
	if !sameSet(urls, []string{"a"}) {
		t.Errorf("expected greenhouse:acme's seen state to be unaffected by lever:widgetco's, got %v", urls)
	}
}

func TestRemoveSeenState_ClearsBoard(t *testing.T) {
	dataDir := t.TempDir()
	boardID := "greenhouse:acme"

	if err := atsboard.RecordSeen(dataDir, boardID, []string{"a"}); err != nil {
		t.Fatalf("RecordSeen: %v", err)
	}
	if err := atsboard.RemoveSeenState(dataDir, boardID); err != nil {
		t.Fatalf("RemoveSeenState: %v", err)
	}

	_, existed, err := atsboard.SeenURLs(dataDir, boardID)
	if err != nil {
		t.Fatalf("SeenURLs: %v", err)
	}
	if existed {
		t.Errorf("expected the board's seen-state to be gone after RemoveSeenState, but existed=true")
	}
}

func TestRemoveSeenState_UnknownBoard_NoError(t *testing.T) {
	dataDir := t.TempDir()
	if err := atsboard.RemoveSeenState(dataDir, "greenhouse:never-tracked"); err != nil {
		t.Errorf("expected RemoveSeenState on an unknown board to be a no-op, got error: %v", err)
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		if !seen[v] {
			return false
		}
	}
	return true
}
