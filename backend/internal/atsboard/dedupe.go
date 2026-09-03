package atsboard

// ListingWithSaved pairs a fetched Listing with whether it's already been
// saved as a Job Listing, so the browse view can mark it rather than let
// the user accidentally create a duplicate (story 7).
type ListingWithSaved struct {
	Listing
	AlreadySaved bool `json:"alreadySaved"`
}

// MarkAlreadySaved matches listings against existingURLs (the URLs of
// already-saved Job Listings) by URL — the one field both a fetched
// Listing and a saved Job Listing always carry from the same posting.
func MarkAlreadySaved(listings []Listing, existingURLs []string) []ListingWithSaved {
	existing := make(map[string]bool, len(existingURLs))
	for _, u := range existingURLs {
		existing[u] = true
	}

	result := make([]ListingWithSaved, len(listings))
	for i, l := range listings {
		result[i] = ListingWithSaved{Listing: l, AlreadySaved: existing[l.URL]}
	}
	return result
}
