package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func exportDataHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := fmt.Sprintf("cv-reporter-export-%s.zip", time.Now().UTC().Format("20060102T150405Z"))

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		if err := tracking.ExportData(dataDir, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
