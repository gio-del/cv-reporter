package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

type saveJobListingRequest struct {
	Company           string `json:"company"`
	URL               string `json:"url"`
	JobDescription    string `json:"jobDescription"`
	JobDescriptionURL string `json:"jobDescriptionUrl"`
}

type saveJobListingResponse struct {
	JobListing  tracking.JobListing  `json:"jobListing"`
	Application tracking.Application `json:"application"`
}

// captureJobListingRequest is what a browser extension's content script can
// trivially read off a job posting page it's already viewing (PRD "Browser
// Extension (LinkedIn Capture)", story 2). Title and Location aren't part of
// tracking.JobListing (see PRD 4's precedent of folding a captured Listing
// down to just company/url/jobDescription before calling Save), so they're
// accepted here but not persisted as separate fields.
type captureJobListingRequest struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func listJobListingsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listings, err := tracking.List(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, listings)
	}
}

func getJobListingHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		listing, err := tracking.GetJobListing(dataDir, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "job listing not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, listing)
	}
}

// suggestContactHandler researches a Contact suggestion for the Job
// Listing identified by id (story 7). It never persists — the FE must
// PATCH /api/applications/{id}/contact to save it once the user confirms.
func suggestContactHandler(dataDir string, client tracking.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		contact, err := tracking.SuggestContact(r.Context(), dataDir, client, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "job listing not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, contact)
	}
}

func createJobListingHandler(dataDir string, client tracking.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveJobListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		listing, application, err := tracking.Save(r.Context(), dataDir, client, tracking.SaveRequest{
			Company:           req.Company,
			URL:               req.URL,
			JobDescription:    req.JobDescription,
			JobDescriptionURL: req.JobDescriptionURL,
		})
		if errors.Is(err, tracking.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, saveJobListingResponse{JobListing: listing, Application: application})
	}
}

// captureJobListingFromExtensionHandler is the browser extension's ingestion
// endpoint (story 3): it normalizes a page capture into tracking.SaveRequest
// and reuses PRD 3's Save path exactly, the same way PRD 4's ATS browse view
// reuses POST /api/job-listings from the frontend.
func captureJobListingFromExtensionHandler(dataDir string, client tracking.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req captureJobListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		listing, application, err := tracking.Save(r.Context(), dataDir, client, tracking.SaveRequest{
			Company:        req.Company,
			URL:            req.URL,
			JobDescription: req.Description,
		})
		if errors.Is(err, tracking.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, saveJobListingResponse{JobListing: listing, Application: application})
	}
}
