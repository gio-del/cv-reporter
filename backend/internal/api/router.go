package api

import "net/http"

// NewRouter builds the HTTP handler for the Master Data API. dataDir is the
// root directory containing profile.yaml, experience/, and projects/ — the
// same files the tailor-cv skill reads and writes.
func NewRouter(dataDir string) http.Handler {
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
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
