package api

import (
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// getUsageHandler serves the running-total Claude API usage/cost across
// every Generation and every standalone (non-Generation) call this dataDir
// has recorded (issue #39: cost/usage visibility, story 11).
func getUsageHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usage, err := tracking.TotalUsage(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, usage)
	}
}
