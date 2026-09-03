package api

import (
	"errors"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

// listAtsListingsHandler fetches boardSlug's open roles from provider's
// public job board API and returns them normalized (story 1), or a clear
// 404 if the board slug doesn't exist on that provider rather than a
// silent empty list (story 6).
func listAtsListingsHandler(atsHTTPDoer atsboard.HTTPDoer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := atsboard.Provider(r.PathValue("provider"))
		slug := r.PathValue("slug")

		listings, err := atsboard.Fetch(r.Context(), atsHTTPDoer, provider, slug)
		if errors.Is(err, atsboard.ErrBoardNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
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
		if listings == nil {
			listings = []atsboard.Listing{}
		}
		writeJSON(w, http.StatusOK, listings)
	}
}
