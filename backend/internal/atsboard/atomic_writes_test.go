package atsboard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

// makeReadOnly makes dir unwritable, which is what a write that cannot
// complete looks like from the caller's side: the previous state file must
// survive it.
func makeReadOnly(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

// Story 14: ATS tracking state survives an interrupted write.
func TestAddTrackedBoard_FailedWriteLeavesPreviousListIntact(t *testing.T) {
	dataDir := t.TempDir()
	if _, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderGreenhouse, "acme", "Acme Corp"); err != nil {
		t.Fatalf("unexpected error seeding the tracked list: %v", err)
	}
	path := filepath.Join(dataDir, "tracked-boards.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, dataDir)

	if _, err := atsboard.AddTrackedBoard(dataDir, atsboard.ProviderLever, "globex", "Globex"); err == nil {
		t.Fatal("expected an error when the write cannot complete, got nil")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading tracked boards after failed write: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("tracked boards were damaged by a failed write:\nwant %q\ngot  %q", string(before), string(after))
	}

	boards, err := atsboard.ListTrackedBoards(dataDir)
	if err != nil {
		t.Fatalf("tracked boards are unreadable after a failed write: %v", err)
	}
	if len(boards) != 1 || boards[0].Slug != "acme" {
		t.Errorf("expected the previously tracked board preserved, got %+v", boards)
	}
}

func TestRecordSeen_FailedWriteLeavesPreviousSeenStateIntact(t *testing.T) {
	dataDir := t.TempDir()
	if err := atsboard.RecordSeen(dataDir, "greenhouse:acme", []string{"https://example.com/1"}); err != nil {
		t.Fatalf("unexpected error seeding seen-state: %v", err)
	}
	path := filepath.Join(dataDir, "tracked-boards-seen.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, dataDir)

	if err := atsboard.RecordSeen(dataDir, "greenhouse:acme", []string{"https://example.com/2"}); err == nil {
		t.Fatal("expected an error when the write cannot complete, got nil")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading seen-state after failed write: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("seen-state was damaged by a failed write:\nwant %q\ngot  %q", string(before), string(after))
	}

	urls, existed, err := atsboard.SeenURLs(dataDir, "greenhouse:acme")
	if err != nil {
		t.Fatalf("seen-state is unreadable after a failed write: %v", err)
	}
	if !existed || len(urls) != 1 || urls[0] != "https://example.com/1" {
		t.Errorf("expected the previous seen-state preserved, got %v (existed=%v)", urls, existed)
	}
}
