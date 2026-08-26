package api

import (
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/claude"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// NewRouter builds the HTTP handler for the app's API. dataDir is the root
// directory containing profile.yaml, experience/, and projects/ — the same
// files the tailor-cv skill reads and writes. Generation calls the real
// Claude API (ADR-0005), reading ANTHROPIC_API_KEY from the environment.
func NewRouter(dataDir string) http.Handler {
	return NewRouterWithGenerationClient(dataDir, claude.New())
}

// NewRouterWithGenerationClient builds the HTTP handler with an explicit
// generation.Client, so tests can inject a fake instead of calling the real
// Claude API (see the PRD's Testing Decisions).
func NewRouterWithGenerationClient(dataDir string, generationClient generation.Client) http.Handler {
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
	mux.HandleFunc("POST /api/generations", createGenerationHandler(dataDir, generationClient))
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
