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
	"time"

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

// LogoURL lets the ATS-browse save path (frontend/src/pages/AtsBrowsePage.tsx)
// pass through a Company Logo it already knows about from atsboard.Listing
// (issue #42) — the manual-paste save path simply never sets it.
type saveJobListingRequest struct {
	Title             string `json:"title"`
	Company           string `json:"company"`
	URL               string `json:"url"`
	JobDescription    string `json:"jobDescription"`
	JobDescriptionURL string `json:"jobDescriptionUrl"`
	LogoURL           string `json:"logoUrl"`
}

type saveJobListingResponse struct {
	JobListing  tracking.JobListing  `json:"jobListing"`
	Application tracking.Application `json:"application"`
	// DuplicateWarning is set when this save looks like a role already
	// tracked under a different Job Listing (issue #43) — a non-blocking
	// hint, never a reason the save above was rejected.
	DuplicateWarning *tracking.DuplicateMatch `json:"duplicateWarning,omitempty"`
}

// findDuplicateWarningBestEffort checks the just-saved listing against every
// other tracked Job Listing and returns a warning if one looks like a
// likely duplicate. It never fails the save: a listing-read error here is
// swallowed (nil warning) rather than surfaced as a 500, since the save
// itself already succeeded by the time this runs.
func findDuplicateWarningBestEffort(dataDir string, saved tracking.JobListing) *tracking.DuplicateMatch {
	all, err := tracking.List(dataDir)
	if err != nil {
		return nil
	}
	existing := make([]tracking.JobListing, 0, len(all))
	for _, lwa := range all {
		existing = append(existing, lwa.JobListing)
	}
	match, found := tracking.FindLikelyDuplicate(saved, existing)
	if !found {
		return nil
	}
	return &match
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

// parseJobListingsFilter reads the optional status/company/savedFrom/savedTo
// query parameters (issue #45), returning a descriptive error for any value
// that can't be parsed rather than silently ignoring it.
func parseJobListingsFilter(query url.Values) (tracking.FilterParams, error) {
	var params tracking.FilterParams

	if status := query.Get("status"); status != "" {
		switch tracking.Status(status) {
		case tracking.StatusSaved, tracking.StatusTailoring, tracking.StatusSent,
			tracking.StatusInterviewing, tracking.StatusRejected, tracking.StatusOffer:
			params.Status = tracking.Status(status)
		default:
			return params, fmt.Errorf("invalid status: %q", status)
		}
	}

	params.Company = query.Get("company")

	if raw := query.Get("savedFrom"); raw != "" {
		from, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return params, fmt.Errorf("invalid savedFrom date: %q", raw)
		}
		params.SavedFrom = &from
	}

	if raw := query.Get("savedTo"); raw != "" {
		to, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return params, fmt.Errorf("invalid savedTo date: %q", raw)
		}
		params.SavedTo = &to
	}

	return params, nil
}

func listJobListingsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := parseJobListingsFilter(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		listings, err := tracking.List(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		listings = tracking.FilterListings(listings, filter)

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

// createJobListingHandler backs both the manual-paste save path and the
// ATS-browse save path — the latter may supply a LogoURL (issue #42, story
// 4-9), downloaded via doer the same way the extension-capture path
// already does (see captureJobListingFromExtensionHandler).
func createJobListingHandler(dataDir string, client tracking.Client, doer tracking.HTTPDoer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveJobListingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		listing, application, err := tracking.Save(r.Context(), dataDir, client, doer, tracking.SaveRequest{
			Title:             req.Title,
			Company:           req.Company,
			URL:               req.URL,
			JobDescription:    req.JobDescription,
			JobDescriptionURL: req.JobDescriptionURL,
			LogoURL:           req.LogoURL,
		})
		if errors.Is(err, tracking.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, saveJobListingResponse{
			JobListing:       listing,
			Application:      application,
			DuplicateWarning: findDuplicateWarningBestEffort(dataDir, listing),
		})
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
		writeJSON(w, http.StatusCreated, saveJobListingResponse{
			JobListing:       listing,
			Application:      application,
			DuplicateWarning: findDuplicateWarningBestEffort(dataDir, listing),
		})
	}
}
