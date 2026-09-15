package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gio-del/sumisura/backend/internal/masterdata"
	"github.com/gio-del/sumisura/backend/internal/recordversion"
)

func getProfileHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := masterdata.GetProfile(dataDir)
		if err != nil {
			// A missing profile.yaml is a setup step the user has not done
			// yet (ADR-0037), not a server fault — say so, and say what to do.
			if errors.Is(err, masterdata.ErrNoProfile) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attachProfileVersion(&profile, dataDir)
		writeJSON(w, http.StatusOK, profileForWire(profile))
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
		writeJSON(w, http.StatusOK, profileForWire(updated))
	}
}
