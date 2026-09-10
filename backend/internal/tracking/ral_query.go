package tracking

import (
	"sort"
	"strings"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// SortOrder is the direction a RAL Range sort runs in — see
// SortListingsByRAL.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// RALSortKey derives the numeric sort key for a RALRange, per PRD "Sort/
// filter Job Listings by RAL Range"'s Implementation Decisions: the
// midpoint of (min, max) for stated/estimated sources (or min itself when
// min == max, which the midpoint formula already yields). n/a, unresolved,
// and conflict carry no single comparable figure — even conflict's own
// DescriptionStated/ListingStated sub-figures are deliberately not used
// here, since picking either would silently prefer one disagreeing source
// over the other (ADR-0014) — so ok is false for all three.
func RALSortKey(ral generation.RALRange) (float64, bool) {
	if ral.Source != generation.RALSourceStated && ral.Source != generation.RALSourceEstimated {
		return 0, false
	}
	if ral.Min == nil || ral.Max == nil {
		return 0, false
	}
	return float64(*ral.Min+*ral.Max) / 2, true
}

// RALRangeOverlaps reports whether ral's own [min, max] figure overlaps
// the filter window [filterMin, filterMax] — an exact boundary touch
// counts as overlap (story 7) — and ral's currency matches filterCurrency
// exactly (case-insensitively). Listings without a comparable single
// figure (n/a, unresolved, conflict) never match a filter (story 6), and a
// currency mismatch excludes rather than cross-converts (Implementation
// Decisions' currency-handling call, mirroring ADR-0014's "never pick
// silently" stance).
func RALRangeOverlaps(ral generation.RALRange, filterMin, filterMax int, filterCurrency string) bool {
	if _, ok := RALSortKey(ral); !ok {
		return false
	}
	if !strings.EqualFold(ral.Currency, filterCurrency) {
		return false
	}
	return *ral.Min <= filterMax && *ral.Max >= filterMin
}

// SortListingsByRAL returns a new slice of listings ordered by RAL Range
// (does not mutate listings). Listings with a comparable numeric figure
// (stated/estimated) are grouped by currency first — never silently
// cross-converting between currencies, per the Implementation Decisions'
// currency-handling call — currency groups ordered alphabetically by code,
// each group internally sorted by RALSortKey in the requested direction.
// Listings with no comparable figure (n/a, unresolved, conflict) always
// trail after every numeric currency group regardless of order, themselves
// ordered by SavedAt descending (most recent first) for stable re-sorts
// (stories 3-4).
func SortListingsByRAL(listings []ListingWithApplication, order SortOrder) []ListingWithApplication {
	byCurrency := map[string][]ListingWithApplication{}
	var currencies []string
	var trailing []ListingWithApplication

	for _, l := range listings {
		if _, ok := RALSortKey(l.JobListing.RAL); !ok {
			trailing = append(trailing, l)
			continue
		}
		currency := l.JobListing.RAL.Currency
		if _, exists := byCurrency[currency]; !exists {
			currencies = append(currencies, currency)
		}
		byCurrency[currency] = append(byCurrency[currency], l)
	}
	sort.Strings(currencies)

	result := make([]ListingWithApplication, 0, len(listings))
	for _, currency := range currencies {
		group := byCurrency[currency]
		sort.SliceStable(group, func(i, j int) bool {
			ki, _ := RALSortKey(group[i].JobListing.RAL)
			kj, _ := RALSortKey(group[j].JobListing.RAL)
			if order == SortOrderDesc {
				return ki > kj
			}
			return ki < kj
		})
		result = append(result, group...)
	}

	sort.SliceStable(trailing, func(i, j int) bool {
		return trailing[i].JobListing.SavedAt > trailing[j].JobListing.SavedAt
	})
	result = append(result, trailing...)

	return result
}

// FilterListingsByRAL returns only the listings whose RAL Range overlaps
// [min, max] in currency (stories 5-7); listings without a comparable
// figure, or priced in a different currency, are excluded (story 6) rather
// than defaulted to zero or silently included.
func FilterListingsByRAL(listings []ListingWithApplication, min, max int, currency string) []ListingWithApplication {
	var result []ListingWithApplication
	for _, l := range listings {
		if RALRangeOverlaps(l.JobListing.RAL, min, max, currency) {
			result = append(result, l)
		}
	}
	return result
}
