package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

func listSnippetsHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snippets, err := masterdata.ListSnippets(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, snippets)
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
		writeJSON(w, http.StatusOK, snippet)
	}
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

		updated, err := masterdata.UpdateSnippet(dataDir, id, snippet)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "snippet not found", http.StatusNotFound)
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
		writeJSON(w, http.StatusOK, updated)
	}
}

func deleteSnippetHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := masterdata.DeleteSnippet(dataDir, id)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "snippet not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
