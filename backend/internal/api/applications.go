package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

type updateApplicationStatusRequest struct {
	Status tracking.Status `json:"status"`
}

func updateApplicationStatusHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req updateApplicationStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		application, err := tracking.UpdateApplicationStatus(dataDir, id, req.Status)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, tracking.ErrInvalidTransition) || errors.Is(err, tracking.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, application)
	}
}
