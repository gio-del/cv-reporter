package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// listAtsListingsHandler fetches boardSlug's open roles from provider's
// public job board API and returns them normalized (story 1) and marked
// with whether each is already saved as a Job Listing (story 7), or a
// clear 404 if the board slug doesn't exist on that provider rather than a
// silent empty list (story 6).
func listAtsListingsHandler(dataDir string, atsHTTPDoer atsboard.HTTPDoer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := atsboard.Provider(r.PathValue("provider"))
		slug := r.PathValue("slug")

		listings, err := atsboard.Fetch(r.Context(), atsHTTPDoer, provider, slug)
		if errors.Is(err, atsboard.ErrBoardNotFound) {
			http.Error(w, fmt.Sprintf(
				"No %s board found for slug %q. Check the slug, or try a different provider.",
				provider, slug,
			), http.StatusNotFound)
			return
		}
		if errors.Is(err, atsboard.ErrUnknownProvider) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		existing, err := tracking.List(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		existingURLs := make([]string, 0, len(existing))
		for _, e := range existing {
			if e.JobListing.URL != "" {
				existingURLs = append(existingURLs, e.JobListing.URL)
			}
		}

		writeJSON(w, http.StatusOK, atsboard.MarkAlreadySaved(listings, existingURLs))
	}
}
