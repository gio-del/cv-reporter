package tracking_test

import (
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func listing(id string, status tracking.Status, company, savedAt string) tracking.ListingWithApplication {
	return tracking.ListingWithApplication{
		JobListing:  tracking.JobListing{ID: id, Company: company, SavedAt: savedAt},
		Application: tracking.Application{ID: id, JobListingID: id, Status: status},
	}
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("invalid fixture date %q: %v", s, err)
	}
	return d
}

func TestFilterListings_NoFilters_ReturnsAllUnchanged(t *testing.T) {
	items := []tracking.ListingWithApplication{
		listing("1", tracking.StatusSaved, "Acme Corp", "2026-01-01T00:00:00Z"),
		listing("2", tracking.StatusInterviewing, "Beta Inc", "2026-02-01T00:00:00Z"),
	}

	got := tracking.FilterListings(items, tracking.FilterParams{})

	if len(got) != 2 {
		t.Fatalf("expected 2 items with no filters, got %d", len(got))
	}
}

func TestFilterListings_ByStatus_ReturnsOnlyMatching(t *testing.T) {
	items := []tracking.ListingWithApplication{
		listing("1", tracking.StatusSaved, "Acme Corp", "2026-01-01T00:00:00Z"),
		listing("2", tracking.StatusInterviewing, "Beta Inc", "2026-02-01T00:00:00Z"),
		listing("3", tracking.StatusInterviewing, "Gamma LLC", "2026-03-01T00:00:00Z"),
	}

	got := tracking.FilterListings(items, tracking.FilterParams{Status: tracking.StatusInterviewing})

	if len(got) != 2 {
		t.Fatalf("expected 2 interviewing items, got %d", len(got))
	}
	for _, item := range got {
		if item.Application.Status != tracking.StatusInterviewing {
			t.Errorf("expected only interviewing items, got %s for %s", item.Application.Status, item.JobListing.ID)
		}
	}
}

func TestFilterListings_ByCompanySubstring_CaseInsensitive(t *testing.T) {
	items := []tracking.ListingWithApplication{
		listing("1", tracking.StatusSaved, "Acme Corp", "2026-01-01T00:00:00Z"),
		listing("2", tracking.StatusSaved, "Beta Inc", "2026-02-01T00:00:00Z"),
	}

	got := tracking.FilterListings(items, tracking.FilterParams{Company: "acme"})

	if len(got) != 1 || got[0].JobListing.ID != "1" {
		t.Fatalf("expected only item 1 to match 'acme' case-insensitively, got %+v", got)
	}
}

func TestFilterListings_BySavedDateRange_InclusiveBoundaries(t *testing.T) {
	items := []tracking.ListingWithApplication{
		listing("1", tracking.StatusSaved, "Acme Corp", "2026-01-01T00:00:00Z"),
		listing("2", tracking.StatusSaved, "Beta Inc", "2026-01-15T00:00:00Z"),
		listing("3", tracking.StatusSaved, "Gamma LLC", "2026-02-01T00:00:00Z"),
	}

	from := mustParseDate(t, "2026-01-01")
	to := mustParseDate(t, "2026-01-15")
	got := tracking.FilterListings(items, tracking.FilterParams{SavedFrom: &from, SavedTo: &to})

	if len(got) != 2 {
		t.Fatalf("expected items 1 and 2 (inclusive boundaries), got %d: %+v", len(got), got)
	}
}

func TestFilterListings_CombinedFilters_AllMustMatch(t *testing.T) {
	items := []tracking.ListingWithApplication{
		listing("1", tracking.StatusInterviewing, "Acme Corp", "2026-01-01T00:00:00Z"),
		listing("2", tracking.StatusSaved, "Acme Corp", "2026-01-05T00:00:00Z"),
		listing("3", tracking.StatusInterviewing, "Beta Inc", "2026-01-05T00:00:00Z"),
	}

	from := mustParseDate(t, "2026-01-01")
	to := mustParseDate(t, "2026-01-31")
	got := tracking.FilterListings(items, tracking.FilterParams{
		Status:    tracking.StatusInterviewing,
		Company:   "acme",
		SavedFrom: &from,
		SavedTo:   &to,
	})

	if len(got) != 1 || got[0].JobListing.ID != "1" {
		t.Fatalf("expected only item 1 to satisfy all filters combined, got %+v", got)
	}
}
