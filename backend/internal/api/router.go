package api

import (
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
	"github.com/gio-del/cv-reporter/backend/internal/claude"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// NewRouter builds the HTTP handler for the app's API. dataDir is the root
// directory containing profile.yaml, experience/, and projects/ — the same
// files the tailor-cv skill reads and writes. Generation calls the real
// Claude API (ADR-0005), reading ANTHROPIC_API_KEY from the environment.
// Render uses the current directory as its project root (see NewRouterFull
// for tests/deployments needing a different one).
func NewRouter(dataDir string) http.Handler {
	return NewRouterFull(dataDir, ".", claude.New())
}

// NewRouterWithGenerationClient builds the HTTP handler with an explicit
// tracking.Client (which embeds generation.Client), so tests can inject a
// fake instead of calling the real Claude API (see the PRD's Testing
// Decisions).
func NewRouterWithGenerationClient(dataDir string, generationClient tracking.Client) http.Handler {
	return NewRouterFull(dataDir, ".", generationClient)
}

// NewRouterWithClients builds the HTTP handler with both an explicit
// tracking.Client and an explicit atsboard.HTTPDoer, so tests can fake the
// outbound calls to ATS public job board APIs alongside the Claude API
// (see the ATS Job Board Aggregation PRD's Testing Decisions).
func NewRouterWithClients(dataDir string, generationClient tracking.Client, atsHTTPDoer atsboard.HTTPDoer) http.Handler {
	return NewRouterFullWithATS(dataDir, ".", generationClient, atsHTTPDoer)
}

// NewRouterFull builds the HTTP handler with an explicit projectRoot: the
// directory containing template/ and output/ that Render's typst
// invocation needs as its --root (see CLAUDE.md and ADR-0012). ATS calls
// use http.DefaultClient — see NewRouterWithClients for tests needing a
// fake.
func NewRouterFull(dataDir, projectRoot string, generationClient tracking.Client) http.Handler {
	return NewRouterFullWithATS(dataDir, projectRoot, generationClient, http.DefaultClient)
}

// NewRouterFullWithATS is NewRouterFull plus an explicit atsboard.HTTPDoer.
func NewRouterFullWithATS(dataDir, projectRoot string, generationClient tracking.Client, atsHTTPDoer atsboard.HTTPDoer) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /api/master-data/entries", listEntriesHandler(dataDir))
	mux.HandleFunc("POST /api/master-data/entries", createEntryHandler(dataDir))
	mux.HandleFunc("GET /api/master-data/entries/{id...}", getEntryHandler(dataDir))
	mux.HandleFunc("PUT /api/master-data/entries/{id...}", putEntryHandler(dataDir))
	mux.HandleFunc("DELETE /api/master-data/entries/{id...}", deleteEntryHandler(dataDir))
	mux.HandleFunc("GET /api/master-data/profile", getProfileHandler(dataDir))
	mux.HandleFunc("PUT /api/master-data/profile", putProfileHandler(dataDir))
	mux.HandleFunc("GET /api/master-data/cover-letter-snippets", listSnippetsHandler(dataDir))
	mux.HandleFunc("POST /api/master-data/cover-letter-snippets", createSnippetHandler(dataDir))
	mux.HandleFunc("GET /api/master-data/cover-letter-snippets/{id...}", getSnippetHandler(dataDir))
	mux.HandleFunc("PUT /api/master-data/cover-letter-snippets/{id...}", putSnippetHandler(dataDir))
	mux.HandleFunc("DELETE /api/master-data/cover-letter-snippets/{id...}", deleteSnippetHandler(dataDir))
	mux.HandleFunc("GET /api/job-listings", listJobListingsHandler(dataDir))
	mux.HandleFunc("POST /api/job-listings", createJobListingHandler(dataDir, generationClient))
	mux.HandleFunc("POST /api/job-listings/from-extension", captureJobListingFromExtensionHandler(dataDir, generationClient))
	mux.HandleFunc("OPTIONS /api/job-listings/from-extension", captureJobListingCORSPreflightHandler)
	mux.HandleFunc("GET /api/job-listings/{id}", getJobListingHandler(dataDir))
	mux.HandleFunc("POST /api/job-listings/{id}/suggest-contact", suggestContactHandler(dataDir, generationClient))
	mux.HandleFunc("PATCH /api/applications/{id}/status", updateApplicationStatusHandler(dataDir))
	mux.HandleFunc("PATCH /api/applications/{id}/method", updateApplicationMethodHandler(dataDir))
	mux.HandleFunc("PATCH /api/applications/{id}/contact", updateApplicationContactHandler(dataDir))
	mux.HandleFunc("GET /api/applications/{id}/mailto", getApplicationMailtoHandler(dataDir))
	mux.HandleFunc("POST /api/applications/{id}/generations", recordApplicationGenerationHandler(dataDir))
	mux.HandleFunc("POST /api/generations", createGenerationHandler(dataDir, generationClient))
	mux.HandleFunc("POST /api/generations/render", renderGenerationHandler(dataDir, projectRoot))
	mux.HandleFunc("GET /api/generations/{slug}/{file}", getGenerationFileHandler(projectRoot))
	mux.HandleFunc("GET /api/ats/{provider}/{slug}/listings", listAtsListingsHandler(dataDir, atsHTTPDoer))
	mux.HandleFunc("GET /api/ats/tracked-boards", listTrackedBoardsHandler(dataDir))
	mux.HandleFunc("POST /api/ats/tracked-boards", createTrackedBoardHandler(dataDir))
	mux.HandleFunc("DELETE /api/ats/tracked-boards/{id}", deleteTrackedBoardHandler(dataDir))
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
