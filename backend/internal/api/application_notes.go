package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// noteRequest is the body of adding a Note, and of editing one: its
// Markdown text (issue #96).
type noteRequest struct {
	Body string `json:"body"`
}

// addApplicationNoteHandler adds a Note to the Application identified by id,
// answering 201 with the whole updated Application, as the other
// per-Application actions do, so the FE replaces its state from one response.
func addApplicationNoteHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req noteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		application, err := tracking.AddNote(dataDir, r.PathValue("id"), req.Body)
		if writeNoteError(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, application)
	}
}

// writeNoteError maps a Note operation's error to its status code and
// reports whether it wrote a response: a missing Application is 404, an
// empty body 400.
func writeNoteError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "application not found", http.StatusNotFound)
	case errors.Is(err, tracking.ErrValidation):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return true
}
