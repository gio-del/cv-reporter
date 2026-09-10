package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// defaultRALCurrency mirrors generation.ParseStatedRAL's own default: RAL
// is an Italian-market term, so a ral_min/ral_max filter with no explicit
// ral_currency is assumed EUR rather than left ambiguous.
const defaultRALCurrency = "EUR"

// parseRALFilter reads the optional ral_min/ral_max/ral_currency query
// params (issue #51). active is false when neither ral_min nor ral_max is
// present, meaning no RAL filter should be applied at all — a listing
// missing a comparable figure is only ever excluded by an active filter,
// never by an absent one.
func parseRALFilter(q url.Values) (min, max int, currency string, active bool, err error) {
	minStr, maxStr := q.Get("ral_min"), q.Get("ral_max")
	if minStr == "" && maxStr == "" {
		return 0, 0, "", false, nil
	}

	min = 0
	if minStr != "" {
		if min, err = strconv.Atoi(minStr); err != nil {
			return 0, 0, "", false, fmt.Errorf("invalid ral_min: %w", err)
		}
	}
	max = math.MaxInt32
	if maxStr != "" {
		if max, err = strconv.Atoi(maxStr); err != nil {
			return 0, 0, "", false, fmt.Errorf("invalid ral_max: %w", err)
		}
	}
	currency = q.Get("ral_currency")
	if currency == "" {
		currency = defaultRALCurrency
	}
	return min, max, currency, true, nil
}

type saveJobListingRequest struct {
	Title             string `json:"title"`
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
// Extension (LinkedIn Capture)", story 2). Location isn't part of
// tracking.JobListing (see PRD 4's precedent of folding a captured Listing
// down to just company/url/jobDescription before calling Save), so it's
// accepted here but not persisted as a separate field; Title and LogoURL
// are (PRD "Job Listing data fidelity").
type captureJobListingRequest struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	// ListingSalaryText is LinkedIn's own salary-insight badge text,
	// captured separately from Description (ADR-0014) — empty when the
	// listing has no such badge, which resolveRALBestEffort treats
	// exactly as today (Job-Description-text-only resolution).
	ListingSalaryText string `json:"listingSalaryText"`
}

// listJobListingsHandler additionally supports optional sort=ral&order=asc|
// desc and ral_min/ral_max/ral_currency query params (issue #51), applied
// on top of tracking.List's own newest-first ordering: filtering (if
// active) runs before sorting, and an unrecognized sort value is ignored
// (List's default order stands) rather than erroring, since only the RAL
// filter's numeric params can be malformed.
func listJobListingsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listings, err := tracking.List(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query()
		if min, max, currency, active, err := parseRALFilter(q); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if active {
			listings = tracking.FilterListingsByRAL(listings, min, max, currency)
		}

		if q.Get("sort") == "ral" {
			order := tracking.SortOrderAsc
			if q.Get("order") == "desc" {
				order = tracking.SortOrderDesc
			}
			listings = tracking.SortListingsByRAL(listings, order)
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

// getJobListingLogoHandler serves the Company Logo file tracking.Save
// downloaded for the Job Listing identified by id (story 8), 404 when the
// listing doesn't exist or has no logo (story 9). Content-Type is left to
// http.ServeFile's own extension-based sniffing, since the file on disk
// already carries the extension downloadLogoBestEffort picked for it.
func getJobListingLogoHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		listing, err := tracking.GetJobListing(dataDir, id)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if listing.Logo == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dataDir, "jobs", listing.Logo))
	}
}

// deleteJobListingHandler removes a Job Listing and its 1:1 Application
// together (story 9), mirroring deleteEntryHandler's shape exactly.
func deleteJobListingHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := tracking.Delete(dataDir, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "job listing not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

// resolveJobListingHandler re-attempts RAL Range resolution and/or
// Application Method inference for the Job Listing identified by id,
// whichever is currently Unresolved (stories 9-13), returning the same
// {jobListing, application} shape Save's endpoints return. 404 when id
// doesn't exist (story 17), matching the existing GetJobListing/
// suggest-contact pattern.
func resolveJobListingHandler(dataDir string, client tracking.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		listing, application, err := tracking.Resolve(r.Context(), dataDir, client, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "job listing not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, saveJobListingResponse{JobListing: listing, Application: application})
	}
}

func createJobListingHandler(dataDir string, client tracking.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveJobListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// No doer: neither the manual-paste nor the ATS-browse save path
		// (this handler's two callers) ever supplies a LogoURL to download.
		listing, application, err := tracking.Save(r.Context(), dataDir, client, nil, tracking.SaveRequest{
			Title:             req.Title,
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

// captureJobListingCORSPreflightHandler answers the browser's CORS preflight
// for the extension's ingestion endpoint. The caller runs on a
// moz-extension:// or chrome-extension:// origin, which is CORS-checked
// like any other cross-origin fetch on at least some Firefox versions,
// regardless of the extension's declared host_permissions (observed in manual
// verification), so this endpoint must answer preflight and carry CORS
// headers itself rather than relying on that exemption.
func captureJobListingCORSPreflightHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

// captureJobListingFromExtensionHandler is the browser extension's ingestion
// endpoint (story 3): it normalizes a page capture into tracking.SaveRequest
// and reuses PRD 3's Save path exactly, the same way PRD 4's ATS browse view
// reuses POST /api/job-listings from the frontend.
func captureJobListingFromExtensionHandler(dataDir string, client tracking.Client, doer tracking.HTTPDoer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var req captureJobListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		listing, application, err := tracking.Save(r.Context(), dataDir, client, doer, tracking.SaveRequest{
			Title:             req.Title,
			Company:           req.Company,
			URL:               req.URL,
			JobDescription:    req.Description,
			LogoURL:           req.LogoURL,
			ListingSalaryText: req.ListingSalaryText,
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
