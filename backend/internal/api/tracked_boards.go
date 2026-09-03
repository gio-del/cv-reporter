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

// listTrackedBoardsHandler lists the companies/boards the user checks
// regularly, so the FE can offer them instead of re-entering a slug every
// time (story 5).
func listTrackedBoardsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boards, err := atsboard.ListTrackedBoards(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, boards)
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

func deleteTrackedBoardHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := atsboard.RemoveTrackedBoard(dataDir, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
