package tracking

import (
	"strings"
	"time"
)

// FilterParams narrows a List result down for the Job Listings list view
// (issue #45). Every field is optional and independent — the zero value
// matches everything, so FilterListings(items, FilterParams{}) is a no-op.
type FilterParams struct {
	Status    Status
	Company   string
	SavedFrom *time.Time
	SavedTo   *time.Time
}

// FilterListings narrows items down to those matching every non-zero field
// of params. It's a pure, in-memory filter over an already-loaded List
// result — no new storage query mechanism, per ADR-0008's flat-file model.
func FilterListings(items []ListingWithApplication, params FilterParams) []ListingWithApplication {
	var result []ListingWithApplication
	for _, item := range items {
		if params.Status != "" && item.Application.Status != params.Status {
			continue
		}
		if params.Company != "" && !strings.Contains(
			strings.ToLower(item.JobListing.Company),
			strings.ToLower(params.Company),
		) {
			continue
		}
		if params.SavedFrom != nil || params.SavedTo != nil {
			savedAt, err := time.Parse(time.RFC3339Nano, item.JobListing.SavedAt)
			if err != nil {
				continue
			}
			if params.SavedFrom != nil && savedAt.Before(*params.SavedFrom) {
				continue
			}
			if params.SavedTo != nil && savedAt.After(endOfDay(*params.SavedTo)) {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

// endOfDay makes a date-only SavedTo bound inclusive of the whole day.
func endOfDay(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 999999999, d.Location())
}
