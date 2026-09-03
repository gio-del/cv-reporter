package atsboard

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const trackedBoardsFile = "tracked-boards.yaml"

// TrackedBoard is one company/board the user checks regularly (story 5),
// so they don't have to re-enter its provider and slug every time. Stored
// as a single small flat file (not a database), consistent with
// ADR-0008. Its ID is provider+slug, so re-adding the same board upserts
// rather than duplicating.
type TrackedBoard struct {
	ID       string   `json:"id" yaml:"id"`
	Provider Provider `json:"provider" yaml:"provider"`
	Slug     string   `json:"slug" yaml:"slug"`
	Label    string   `json:"label,omitempty" yaml:"label,omitempty"`
}

func trackedBoardID(provider Provider, slug string) string {
	return string(provider) + ":" + slug
}

// ListTrackedBoards reads dataDir/tracked-boards.yaml, returning an empty
// slice (not an error) when it doesn't exist yet.
func ListTrackedBoards(dataDir string) ([]TrackedBoard, error) {
	content, err := os.ReadFile(filepath.Join(dataDir, trackedBoardsFile))
	if os.IsNotExist(err) {
		return []TrackedBoard{}, nil
	}
	if err != nil {
		return nil, err
	}
	var boards []TrackedBoard
	if err := yaml.Unmarshal(content, &boards); err != nil {
		return nil, err
	}
	return boards, nil
}

// AddTrackedBoard adds provider/slug to the tracked list, or updates its
// label if that provider/slug is already tracked (upsert, keyed by ID).
func AddTrackedBoard(dataDir string, provider Provider, slug, label string) (TrackedBoard, error) {
	boards, err := ListTrackedBoards(dataDir)
	if err != nil {
		return TrackedBoard{}, err
	}

	board := TrackedBoard{ID: trackedBoardID(provider, slug), Provider: provider, Slug: slug, Label: label}

	replaced := false
	for i, b := range boards {
		if b.ID == board.ID {
			boards[i] = board
			replaced = true
			break
		}
	}
	if !replaced {
		boards = append(boards, board)
	}

	if err := writeTrackedBoards(dataDir, boards); err != nil {
		return TrackedBoard{}, err
	}
	return board, nil
}

// RemoveTrackedBoard removes the tracked board with the given id, if
// present.
func RemoveTrackedBoard(dataDir, id string) error {
	boards, err := ListTrackedBoards(dataDir)
	if err != nil {
		return err
	}

	kept := boards[:0]
	for _, b := range boards {
		if b.ID != id {
			kept = append(kept, b)
		}
	}
	return writeTrackedBoards(dataDir, kept)
}

func writeTrackedBoards(dataDir string, boards []TrackedBoard) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	content, err := yaml.Marshal(boards)
	if err != nil {
		return fmt.Errorf("marshaling tracked boards: %w", err)
	}
	return os.WriteFile(filepath.Join(dataDir, trackedBoardsFile), content, 0o644)
}
