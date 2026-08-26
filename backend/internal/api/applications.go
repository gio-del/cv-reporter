package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

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

type recordGenerationRequest struct {
	Slug            string `json:"slug"`
	CVPath          string `json:"cvPath"`
	CoverLetterPath string `json:"coverLetterPath"`
}

// recordApplicationGenerationHandler records a Generation the FE already
// rendered (via POST /api/generations/render) against the Application
// identified by id (story 11) — the linking step, since the render
// pipeline itself has no notion of which Application it's for.
func recordApplicationGenerationHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req recordGenerationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Slug) == "" {
			http.Error(w, "slug is required", http.StatusBadRequest)
			return
		}

		record := tracking.GenerationRecord{
			Slug:            req.Slug,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
			CVPath:          req.CVPath,
			CoverLetterPath: req.CoverLetterPath,
		}
		application, err := tracking.RecordGeneration(dataDir, id, record)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, application)
	}
}

func updateApplicationMethodHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req tracking.ApplicationMethod
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		application, err := tracking.UpdateApplicationMethod(dataDir, id, req)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, tracking.ErrValidation) {
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
