package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
	"github.com/gio-del/cv-reporter/backend/internal/recordversion"
)

func getProfileHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := masterdata.GetProfile(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attachProfileVersion(&profile, dataDir)
		writeJSON(w, http.StatusOK, profile)
	}
}

// attachProfileVersion populates profile.Version, the read-time token a
// later conditional write is checked against (issue #89).
func attachProfileVersion(profile *masterdata.Profile, dataDir string) {
	version, err := masterdata.ProfileVersion(dataDir)
	if err != nil {
		return
	}
	profile.Version = version
}

func putProfileHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var profile masterdata.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := masterdata.UpdateProfileIfMatch(dataDir, profile, requestVersion(r))
		if errors.Is(err, recordversion.ErrMismatch) {
			writeConflict(w, "Profile")
			return
		}
		if errors.Is(err, masterdata.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attachProfileVersion(&updated, dataDir)
		writeJSON(w, http.StatusOK, updated)
	}
}
