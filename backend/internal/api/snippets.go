package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
	"github.com/gio-del/cv-reporter/backend/internal/recordversion"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// snippetWithUsage enriches a Snippet with when it was last used, per issue
// #48 — computed from recorded Generations, not hand-maintained, so it can't
// drift out of sync (see tracking.SnippetLastUsed). Absent/omitted means "no
// usage signal", which covers both "never used" and "recorded before this
// field existed" identically, per the PRD's retroactive-data story.
type snippetWithUsage struct {
	masterdata.Snippet
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

func listSnippetsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snippets, err := masterdata.ListSnippets(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		listings, err := tracking.List(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applications := make([]tracking.Application, len(listings))
		for i, l := range listings {
			applications[i] = l.Application
		}
		lastUsed := tracking.SnippetLastUsed(applications)

		enriched := make([]snippetWithUsage, len(snippets))
		for i, s := range snippets {
			attachSnippetVersion(&s, dataDir)
			enriched[i] = snippetWithUsage{Snippet: s, LastUsedAt: lastUsed[s.ID]}
		}
		writeJSON(w, http.StatusOK, enriched)
	}
}

func getSnippetHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		snippet, err := masterdata.GetSnippet(dataDir, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "snippet not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attachSnippetVersion(&snippet, dataDir)
		writeJSON(w, http.StatusOK, snippet)
	}
}

// attachSnippetVersion populates snippet.Version, the read-time token a
// later conditional write is checked against (issue #89).
func attachSnippetVersion(snippet *masterdata.Snippet, dataDir string) {
	version, err := masterdata.SnippetVersion(dataDir, snippet.ID)
	if err != nil {
		return
	}
	snippet.Version = version
}

func createSnippetHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var snippet masterdata.Snippet
		if err := json.NewDecoder(r.Body).Decode(&snippet); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		created, err := masterdata.CreateSnippet(dataDir, snippet)
		if errors.Is(err, masterdata.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attachSnippetVersion(&created, dataDir)
		writeJSON(w, http.StatusCreated, created)
	}
}

func putSnippetHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var snippet masterdata.Snippet
		if err := json.NewDecoder(r.Body).Decode(&snippet); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := masterdata.UpdateSnippetIfMatch(dataDir, id, snippet, requestVersion(r))
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, recordversion.ErrMismatch) {
			writeConflict(w, "Cover Letter Snippet")
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
		attachSnippetVersion(&updated, dataDir)
		writeJSON(w, http.StatusOK, updated)
	}
}

func deleteSnippetHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := masterdata.DeleteSnippetIfMatch(dataDir, id, requestVersion(r))
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, recordversion.ErrMismatch) {
			writeConflict(w, "Cover Letter Snippet")
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
