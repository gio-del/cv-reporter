package atsboard_test

import (
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

func TestMarkAlreadySaved_MatchesByURL(t *testing.T) {
	listings := []atsboard.Listing{
		{Title: "Backend Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"},
		{Title: "Product Designer", URL: "https://boards.greenhouse.io/acme/jobs/2"},
	}
	existingURLs := []string{"https://boards.greenhouse.io/acme/jobs/2"}

	got := atsboard.MarkAlreadySaved(listings, existingURLs)

	if len(got) != 2 {
		t.Fatalf("expected 2 listings, got %d", len(got))
	}
	if got[0].AlreadySaved {
		t.Errorf("expected listing 1 (not in existingURLs) to be unmarked, got %+v", got[0])
	}
	if !got[1].AlreadySaved {
		t.Errorf("expected listing 2 (in existingURLs) to be marked already saved, got %+v", got[1])
	}
	if got[1].Listing != listings[1] {
		t.Errorf("expected the underlying Listing to be preserved, got %+v", got[1].Listing)
	}
}

func TestMarkAlreadySaved_NoExistingURLs_NoneMarked(t *testing.T) {
	listings := []atsboard.Listing{{Title: "Backend Engineer", URL: "https://boards.greenhouse.io/acme/jobs/1"}}

	got := atsboard.MarkAlreadySaved(listings, nil)

	if len(got) != 1 || got[0].AlreadySaved {
		t.Errorf("expected no listings marked already saved, got %+v", got)
	}
}
