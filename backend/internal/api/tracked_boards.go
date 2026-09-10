package api

import (
	"encoding/json"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

type addTrackedBoardRequest struct {
	Provider atsboard.Provider `json:"provider"`
	Slug     string            `json:"slug"`
	Label    string            `json:"label"`
}

// trackedBoardWithNewCount adds a read-only new-since-last-check count
// preview to a TrackedBoard, so the FE can badge boards worth opening
// without fetching every board's full listing (Implementation Decisions).
type trackedBoardWithNewCount struct {
	atsboard.TrackedBoard
	NewCount int `json:"newCount"`
}

// listTrackedBoardsHandler lists the companies/boards the user checks
// regularly, so the FE can offer them instead of re-entering a slug every
// time (story 5), each annotated with a new-listing count. Computing the
// count fetches each board's current listings but — unlike the listings
// endpoint itself — never mutates seen-state, so just viewing this list
// doesn't silently mark anything as seen (story 3 ties "seen" to actually
// viewing a board's fetched list, not to this summary). A board whose
// live fetch fails gets newCount=0 rather than failing the whole list.
func listTrackedBoardsHandler(dataDir string, atsHTTPDoer atsboard.HTTPDoer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boards, err := atsboard.ListTrackedBoards(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := make([]trackedBoardWithNewCount, len(boards))
		for i, b := range boards {
			result[i] = trackedBoardWithNewCount{TrackedBoard: b, NewCount: 0}

			listings, err := atsboard.Fetch(r.Context(), atsHTTPDoer, b.Provider, b.Slug)
			if err != nil {
				continue
			}
			seenURLs, existed, err := atsboard.SeenURLs(dataDir, b.ID)
			if err != nil || !existed {
				// No seen-state yet means this board has never actually
				// been opened — treat its count as 0 rather than the
				// whole backlog, matching the listings endpoint's own
				// first-fetch seeding behavior (story 5).
				continue
			}
			for _, l := range atsboard.MarkNewSinceLastCheck(listings, seenURLs) {
				if l.New {
					result[i].NewCount++
				}
			}
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func createTrackedBoardHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addTrackedBoardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Slug == "" || req.Provider == "" {
			http.Error(w, "provider and slug are required", http.StatusBadRequest)
			return
		}

		board, err := atsboard.AddTrackedBoard(dataDir, req.Provider, req.Slug, req.Label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, board)
	}
}

// deleteTrackedBoardHandler untracks a board and drops its seen-state
// (story 6), so re-tracking it later starts a fresh baseline instead of
// resurrecting stale seen-history.
func deleteTrackedBoardHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := atsboard.RemoveTrackedBoard(dataDir, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := atsboard.RemoveSeenState(dataDir, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
