package atsboard_test

import (
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

func TestTrackedBoards_AddThenList_ReturnsIt(t *testing.T) {
	dataDir := t.TempDir()

	added, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderGreenhouse, "acme", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added.Provider != atsboard.ProviderGreenhouse || added.Slug != "acme" || added.Label != "Acme Corp" {
		t.Errorf("unexpected added board: %+v", added)
	}
	if added.ID == "" {
		t.Error("expected a non-empty id")
	}

	boards, err := atsboard.ListTrackedBoards(dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(boards) != 1 {
		t.Fatalf("expected 1 tracked board, got %d", len(boards))
	}
	if boards[0] != added {
		t.Errorf("expected listed board %+v to match added %+v", boards[0], added)
	}
}

func TestTrackedBoards_ListOnEmptyDataDir_ReturnsEmptyNotError(t *testing.T) {
	dataDir := t.TempDir()

	boards, err := atsboard.ListTrackedBoards(dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(boards) != 0 {
		t.Errorf("expected no tracked boards, got %d", len(boards))
	}
}

func TestTrackedBoards_AddSameProviderAndSlugTwice_Upserts(t *testing.T) {
	dataDir := t.TempDir()

	first, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderGreenhouse, "acme", "Acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderGreenhouse, "acme", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("expected re-adding the same provider+slug to keep the same id, got %q then %q", first.ID, second.ID)
	}

	boards, err := atsboard.ListTrackedBoards(dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(boards) != 1 {
		t.Fatalf("expected upsert to keep a single entry, got %d", len(boards))
	}
	if boards[0].Label != "Acme Corp" {
		t.Errorf("expected upsert to update the label, got %q", boards[0].Label)
	}
}

func TestTrackedBoards_Remove_RemovesIt(t *testing.T) {
	dataDir := t.TempDir()

	added, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderLever, "acme", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := atsboard.RemoveTrackedBoard(dataDir, added.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boards, err := atsboard.ListTrackedBoards(dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(boards) != 0 {
		t.Errorf("expected no tracked boards after removal, got %d", len(boards))
	}
}
