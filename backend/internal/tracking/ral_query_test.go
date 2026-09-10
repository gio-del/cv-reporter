package tracking

import (
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

func intp(v int) *int { return &v }

func TestRALSortKey(t *testing.T) {
	tests := []struct {
		name    string
		ral     generation.RALRange
		wantKey float64
		wantOK  bool
	}{
		{
			name:    "stated range midpoint",
			ral:     generation.RALRange{Min: intp(40000), Max: intp(60000), Currency: "EUR", Source: generation.RALSourceStated},
			wantKey: 50000,
			wantOK:  true,
		},
		{
			name:    "stated single figure (min == max)",
			ral:     generation.RALRange{Min: intp(50000), Max: intp(50000), Currency: "EUR", Source: generation.RALSourceStated},
			wantKey: 50000,
			wantOK:  true,
		},
		{
			name:    "estimated range midpoint",
			ral:     generation.RALRange{Min: intp(45000), Max: intp(55000), Currency: "EUR", Source: generation.RALSourceEstimated},
			wantKey: 50000,
			wantOK:  true,
		},
		{
			name:    "odd-sum midpoint is fractional, not truncated",
			ral:     generation.RALRange{Min: intp(40000), Max: intp(40001), Currency: "EUR", Source: generation.RALSourceEstimated},
			wantKey: 40000.5,
			wantOK:  true,
		},
		{
			name:   "n/a has no sort key",
			ral:    generation.RALRange{Source: generation.RALSourceNA},
			wantOK: false,
		},
		{
			name:   "unresolved has no sort key",
			ral:    generation.RALRange{Source: generation.RALSourceUnresolved},
			wantOK: false,
		},
		{
			name: "conflict has no sort key even though sub-figures exist",
			ral: generation.RALRange{
				Source:            generation.RALSourceConflict,
				DescriptionStated: &generation.RALFigure{Min: 40000, Max: 40000, Currency: "EUR"},
				ListingStated:     &generation.RALFigure{Min: 90000, Max: 90000, Currency: "EUR"},
			},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, ok := RALSortKey(tt.ral)
			if ok != tt.wantOK {
				t.Fatalf("RALSortKey() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && key != tt.wantKey {
				t.Errorf("RALSortKey() key = %v, want %v", key, tt.wantKey)
			}
		})
	}
}

func TestRALRangeOverlaps(t *testing.T) {
	tests := []struct {
		name           string
		ral            generation.RALRange
		filterMin      int
		filterMax      int
		filterCurrency string
		want           bool
	}{
		{
			name:           "fully contained within filter window",
			ral:            generation.RALRange{Min: intp(50000), Max: intp(55000), Currency: "EUR", Source: generation.RALSourceStated},
			filterMin:      40000,
			filterMax:      60000,
			filterCurrency: "EUR",
			want:           true,
		},
		{
			name:           "partial overlap counts as a match (story 7)",
			ral:            generation.RALRange{Min: intp(30000), Max: intp(50000), Currency: "EUR", Source: generation.RALSourceStated},
			filterMin:      45000,
			filterMax:      70000,
			filterCurrency: "EUR",
			want:           true,
		},
		{
			name:           "exact boundary touch counts as overlap",
			ral:            generation.RALRange{Min: intp(50000), Max: intp(60000), Currency: "EUR", Source: generation.RALSourceStated},
			filterMin:      60000,
			filterMax:      70000,
			filterCurrency: "EUR",
			want:           true,
		},
		{
			name:           "non-overlapping ranges never match",
			ral:            generation.RALRange{Min: intp(30000), Max: intp(39999), Currency: "EUR", Source: generation.RALSourceStated},
			filterMin:      40000,
			filterMax:      50000,
			filterCurrency: "EUR",
			want:           false,
		},
		{
			name:           "estimated source is a comparable single figure too",
			ral:            generation.RALRange{Min: intp(45000), Max: intp(45000), Currency: "EUR", Source: generation.RALSourceEstimated},
			filterMin:      40000,
			filterMax:      50000,
			filterCurrency: "EUR",
			want:           true,
		},
		{
			name:           "n/a never matches regardless of window",
			ral:            generation.RALRange{Source: generation.RALSourceNA},
			filterMin:      0,
			filterMax:      1000000,
			filterCurrency: "EUR",
			want:           false,
		},
		{
			name:           "unresolved never matches regardless of window",
			ral:            generation.RALRange{Source: generation.RALSourceUnresolved},
			filterMin:      0,
			filterMax:      1000000,
			filterCurrency: "EUR",
			want:           false,
		},
		{
			name: "conflict never matches even if a sub-figure overlaps",
			ral: generation.RALRange{
				Source:            generation.RALSourceConflict,
				DescriptionStated: &generation.RALFigure{Min: 45000, Max: 45000, Currency: "EUR"},
				ListingStated:     &generation.RALFigure{Min: 90000, Max: 90000, Currency: "EUR"},
			},
			filterMin:      40000,
			filterMax:      50000,
			filterCurrency: "EUR",
			want:           false,
		},
		{
			name:           "mismatched currency excludes rather than cross-converts",
			ral:            generation.RALRange{Min: intp(45000), Max: intp(45000), Currency: "USD", Source: generation.RALSourceStated},
			filterMin:      40000,
			filterMax:      50000,
			filterCurrency: "EUR",
			want:           false,
		},
		{
			name:           "currency comparison is case-insensitive",
			ral:            generation.RALRange{Min: intp(45000), Max: intp(45000), Currency: "eur", Source: generation.RALSourceStated},
			filterMin:      40000,
			filterMax:      50000,
			filterCurrency: "EUR",
			want:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RALRangeOverlaps(tt.ral, tt.filterMin, tt.filterMax, tt.filterCurrency)
			if got != tt.want {
				t.Errorf("RALRangeOverlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func listingWithRAL(id, savedAt string, ral generation.RALRange) ListingWithApplication {
	return ListingWithApplication{JobListing: JobListing{ID: id, SavedAt: savedAt, RAL: ral}}
}

func idsOf(listings []ListingWithApplication) []string {
	ids := make([]string, len(listings))
	for i, l := range listings {
		ids[i] = l.JobListing.ID
	}
	return ids
}

func TestSortListingsByRAL(t *testing.T) {
	low := listingWithRAL("low", "2026-01-01T00:00:00Z", generation.RALRange{Min: intp(30000), Max: intp(30000), Currency: "EUR", Source: generation.RALSourceStated})
	mid := listingWithRAL("mid", "2026-01-02T00:00:00Z", generation.RALRange{Min: intp(50000), Max: intp(50000), Currency: "EUR", Source: generation.RALSourceEstimated})
	high := listingWithRAL("high", "2026-01-03T00:00:00Z", generation.RALRange{Min: intp(70000), Max: intp(70000), Currency: "EUR", Source: generation.RALSourceStated})
	na := listingWithRAL("na", "2026-01-04T00:00:00Z", generation.RALRange{Source: generation.RALSourceNA})
	unresolved := listingWithRAL("unresolved", "2026-01-06T00:00:00Z", generation.RALRange{Source: generation.RALSourceUnresolved})
	conflict := listingWithRAL("conflict", "2026-01-05T00:00:00Z", generation.RALRange{Source: generation.RALSourceConflict})
	usd := listingWithRAL("usd", "2026-01-01T00:00:00Z", generation.RALRange{Min: intp(60000), Max: intp(60000), Currency: "USD", Source: generation.RALSourceStated})

	t.Run("ascending numeric order, non-numeric trailing by most-recent-first", func(t *testing.T) {
		input := []ListingWithApplication{na, high, unresolved, low, conflict, mid}
		got := idsOf(SortListingsByRAL(input, SortOrderAsc))
		want := []string{"low", "mid", "high", "unresolved", "conflict", "na"}
		assertEqualStrings(t, got, want)
	})

	t.Run("descending numeric order still trails non-numeric last", func(t *testing.T) {
		input := []ListingWithApplication{na, high, unresolved, low, conflict, mid}
		got := idsOf(SortListingsByRAL(input, SortOrderDesc))
		want := []string{"high", "mid", "low", "unresolved", "conflict", "na"}
		assertEqualStrings(t, got, want)
	})

	t.Run("does not mutate the input slice", func(t *testing.T) {
		input := []ListingWithApplication{high, low}
		_ = SortListingsByRAL(input, SortOrderAsc)
		if input[0].JobListing.ID != "high" || input[1].JobListing.ID != "low" {
			t.Errorf("expected input slice untouched, got %v", idsOf(input))
		}
	})

	t.Run("groups by currency before sorting within each group", func(t *testing.T) {
		input := []ListingWithApplication{usd, high, low}
		got := idsOf(SortListingsByRAL(input, SortOrderAsc))
		// EUR ("high","low") sorts before USD ("usd") alphabetically by
		// currency code, ascending within each currency group.
		want := []string{"low", "high", "usd"}
		assertEqualStrings(t, got, want)
	})
}

func TestFilterListingsByRAL(t *testing.T) {
	inRange := listingWithRAL("in-range", "", generation.RALRange{Min: intp(45000), Max: intp(45000), Currency: "EUR", Source: generation.RALSourceStated})
	outOfRange := listingWithRAL("out-of-range", "", generation.RALRange{Min: intp(90000), Max: intp(90000), Currency: "EUR", Source: generation.RALSourceStated})
	wrongCurrency := listingWithRAL("wrong-currency", "", generation.RALRange{Min: intp(45000), Max: intp(45000), Currency: "USD", Source: generation.RALSourceStated})
	na := listingWithRAL("na", "", generation.RALRange{Source: generation.RALSourceNA})
	unresolved := listingWithRAL("unresolved", "", generation.RALRange{Source: generation.RALSourceUnresolved})
	conflict := listingWithRAL("conflict", "", generation.RALRange{Source: generation.RALSourceConflict})

	input := []ListingWithApplication{inRange, outOfRange, wrongCurrency, na, unresolved, conflict}
	got := idsOf(FilterListingsByRAL(input, 40000, 50000, "EUR"))
	want := []string{"in-range"}
	assertEqualStrings(t, got, want)
}

func assertEqualStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
