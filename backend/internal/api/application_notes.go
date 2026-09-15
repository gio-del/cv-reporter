package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gio-del/sumisura/backend/internal/recordversion"
	"github.com/gio-del/sumisura/backend/internal/tracking"
)

// noteRequest is the body of adding a Note, and of editing one: its
// Markdown text (issue #96).
type noteRequest struct {
	Body string `json:"body"`
}

// addApplicationNoteHandler adds a Note to the Application identified by id,
// answering 201 with the whole updated Application, as the other
// per-Application actions do, so the FE replaces its state from one response.
// Like every Note route it honours an optional If-Match carrying the
// Application's version token (issue #89).
func addApplicationNoteHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req noteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		application, err := tracking.AddNoteIfMatch(dataDir, r.PathValue("id"), req.Body, requestVersion(r))
		if writeNoteError(w, err) {
			return
		}
		attachApplicationVersion(&application, dataDir)
		writeJSON(w, http.StatusCreated, application)
	}
}

// editApplicationNoteHandler replaces the body of the Note noteId on the
// Application id, answering 200 with the whole updated Application. The
// Note's creation timestamp is never changed.
func editApplicationNoteHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req noteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		application, err := tracking.EditNoteIfMatch(dataDir, r.PathValue("id"), r.PathValue("noteId"), req.Body, requestVersion(r))
		if writeNoteError(w, err) {
			return
		}
		attachApplicationVersion(&application, dataDir)
		writeJSON(w, http.StatusOK, application)
	}
}

// deleteApplicationNoteHandler hard-deletes the Note noteId from the
// Application id, answering 200 with the whole updated Application.
func deleteApplicationNoteHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		application, err := tracking.DeleteNoteIfMatch(dataDir, r.PathValue("id"), r.PathValue("noteId"), requestVersion(r))
		if writeNoteError(w, err) {
			return
		}
		attachApplicationVersion(&application, dataDir)
		writeJSON(w, http.StatusOK, application)
	}
}

// writeNoteError maps a Note operation's error to its status code and
// reports whether it wrote a response: a missing Application or Note is
// 404, an empty body 400, a stale If-Match 409.
func writeNoteError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, tracking.ErrNoteNotFound):
		http.Error(w, "note not found", http.StatusNotFound)
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "application not found", http.StatusNotFound)
	case errors.Is(err, recordversion.ErrMismatch):
		writeConflict(w, "Application")
	case errors.Is(err, tracking.ErrValidation):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return true
}
