package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

func getProfileHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := masterdata.GetProfile(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	}
}

func putProfileHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var profile masterdata.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := masterdata.UpdateProfile(dataDir, profile)
		if errors.Is(err, masterdata.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}
